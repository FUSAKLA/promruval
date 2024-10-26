package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prom_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	bearerTokenEnvVar = "PROMETHEUS_BEARER_TOKEN"
	userAgent         = "promruval"
)

func sourceTenantsToHeader(sourceTenants []string) string {
	return strings.Join(sort.StringSlice(sourceTenants), "|")
}

func loadBearerToken(promConfig config.PrometheusConfig) (string, error) {
	bearerToken := ""
	if promConfig.BearerTokenFile != "" {
		if path.IsAbs(promConfig.BearerTokenFile) {
			return "", fmt.Errorf("`bearerTokenFile` must be a relative path to the config file")
		}
		p := path.Join(config.BaseDirPath(), promConfig.BearerTokenFile)
		token, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", p, err)
		}
		bearerToken = string(token)
	}
	tokenFromEnv := os.Getenv(bearerTokenEnvVar)
	if tokenFromEnv != "" {
		bearerToken = tokenFromEnv
	}
	return strings.TrimSpace(bearerToken), nil
}

func NewClient(promConfig config.PrometheusConfig) (*Client, error) {
	var tripper http.RoundTripper = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: promConfig.InsecureSkipTLSVerify}}
	bearerToken, err := loadBearerToken(promConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load bearer token: %w", err)
	}
	if bearerToken != "" {
		tripper = prom_config.NewAuthorizationCredentialsRoundTripper("Bearer", prom_config.NewInlineSecret(bearerToken), tripper)
	}
	return NewClientWithRoundTripper(promConfig, tripper)
}

func NewClientWithRoundTripper(promConfig config.PrometheusConfig, tripper http.RoundTripper) (*Client, error) {
	headers := prom_config.Headers{
		Headers: map[string]prom_config.Header{
			"User-Agent": {Values: []string{userAgent}},
		},
	}
	originalSourceTenants := []string{}
	for k, v := range promConfig.HTTPHeaders {
		if k == user.OrgIDHeaderName {
			originalSourceTenants = []string{v}
		}
		headers.Headers[k] = prom_config.Header{Values: []string{v}}
	}
	cli, err := api.NewClient(api.Config{
		Address:      promConfig.URL,
		RoundTripper: prom_config.NewHeadersRoundTripper(&headers, tripper),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus client: %w", err)
	}
	v1cli := v1.NewAPI(cli)
	promClient := Client{
		apiClient:             v1cli,
		httpHeaders:           &headers,
		originalSourceTenants: originalSourceTenants,
		httpHeadersMtx:        sync.Mutex{},
		url:                   promConfig.URL,
		timeout:               promConfig.Timeout,
		queryOffset:           promConfig.QueryOffset,
		queryLookback:         promConfig.QueryLookback,
		cache:                 newCache(promConfig.CacheFile, promConfig.URL, promConfig.MaxCacheAge),
	}
	return &promClient, nil
}

type Client struct {
	apiClient             v1.API
	httpHeaders           *prom_config.Headers
	originalSourceTenants []string
	httpHeadersMtx        sync.Mutex
	url                   string
	timeout               time.Duration
	queryOffset           time.Duration
	queryLookback         time.Duration
	cache                 *cache
}

func (s *Client) SetSourceTenants(sourceTenants []string) {
	s.httpHeadersMtx.Lock()
	if len(sourceTenants) == 0 {
		return
	}
	s.httpHeaders.Headers[user.OrgIDHeaderName] = prom_config.Header{Values: []string{sourceTenantsToHeader(sourceTenants)}}
}

func (s *Client) ClearSourceTenants() {
	s.httpHeaders.Headers[user.OrgIDHeaderName] = prom_config.Header{Values: s.originalSourceTenants}
	s.httpHeadersMtx.Unlock()
}

func (s *Client) queryTimeRange() (start, end time.Time) {
	end = time.Now().Add(-s.queryOffset)
	start = end.Add(-s.queryLookback)
	return start, end
}

func (s *Client) DumpCache() {
	start := time.Now()
	s.cache.Dump()
	log.WithField("duration", time.Since(start)).Info("cache dumped")
}

func (s *Client) newContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	if s.timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	return ctx, cancel
}

func (s *Client) SelectorMatch(selector string, sourceTenants []string) ([]model.LabelSet, error) {
	ctx, cancel := s.newContext()
	defer cancel()
	s.SetSourceTenants(sourceTenants)
	defer s.ClearSourceTenants()
	start := time.Now()
	queryStart, queryEnd := s.queryTimeRange()
	result, warnings, err := s.apiClient.Series(ctx, []string{selector}, queryStart, queryEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query series: %w", err)
	}
	log.WithFields(log.Fields{
		"selector":       selector,
		"url":            s.url,
		"sourceTenants":  sourceTenants,
		"duration":       time.Since(start),
		"matchingSeries": len(result),
	}).Debug("queried prometheus for series matching selector")
	if len(warnings) > 0 {
		log.WithField("warnings", warnings).Warn("Prometheus query returned warnings")
	}
	return result, nil
}

func (s *Client) SelectorMatchingSeries(selector string, sourceTenants []string) (int, error) {
	if count, found := s.cache.SourceTenantsData(sourceTenants).SelectorMatchingSeries[selector]; found {
		return count, nil
	}
	series, err := s.SelectorMatch(selector, sourceTenants)
	if err != nil {
		return 0, err
	}
	s.cache.SourceTenantsData(sourceTenants).SelectorMatchingSeries[selector] = len(series)
	return len(series), nil
}

func (s *Client) Labels(sourceTenants []string) ([]string, error) {
	if len(s.cache.SourceTenantsData(sourceTenants).KnownLabels) == 0 {
		ctx, cancel := s.newContext()
		defer cancel()
		s.SetSourceTenants(sourceTenants)
		defer s.ClearSourceTenants()
		start := time.Now()
		queryStart, queryEnd := s.queryTimeRange()
		result, warnings, err := s.apiClient.LabelNames(ctx, []string{}, queryStart, queryEnd)
		if err != nil {
			return nil, err
		}
		log.WithFields(log.Fields{
			"url":           s.url,
			"sourceTenants": sourceTenants,
			"duration":      time.Since(start),
			"labels":        len(result),
			"queryStart":    queryStart,
			"queryEnd":      queryEnd,
		}).Debug("loaded all prometheus label names")
		if len(warnings) > 0 {
			log.WithField("warnings", warnings).Warn("Prometheus query returned warnings")
		}
		s.cache.SourceTenantsData(sourceTenants).KnownLabels = result
	}
	return s.cache.SourceTenantsData(sourceTenants).KnownLabels, nil
}

func (s *Client) Query(query string, sourceTenants []string) ([]*model.Sample, int, time.Duration, error) {
	ctx, cancel := s.newContext()
	defer cancel()
	s.SetSourceTenants(sourceTenants)
	defer s.ClearSourceTenants()
	start := time.Now()
	_, queryEnd := s.queryTimeRange()
	result, warnings, err := s.apiClient.Query(ctx, query, queryEnd)
	duration := time.Since(start)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error querying prometheus: %w", err)
	}
	log.WithFields(log.Fields{
		"url":           s.url,
		"query":         query,
		"at":            queryEnd,
		"sourceTenants": sourceTenants,
		"duration":      time.Since(start),
		"resultType":    result.Type().String(),
	}).Debug("query prometheus")
	if len(warnings) > 0 {
		log.WithField("warnings", warnings).Warn("Prometheus query returned warnings")
	}
	switch result.Type() {
	case model.ValVector:
		vectorResult, ok := result.(model.Vector)
		if !ok {
			return nil, 0, 0, fmt.Errorf("failed to convert result to model.Vector")
		}
		return vectorResult, len(vectorResult), duration, nil
	case model.ValScalar:
		scalarResult, ok := result.(*model.Scalar)
		if !ok {
			return nil, 0, 0, fmt.Errorf("failed to convert result to model.Scalar")
		}
		return []*model.Sample{{Value: scalarResult.Value, Timestamp: model.Now()}}, 1, duration, nil
	}
	return nil, 0, 0, fmt.Errorf("unknown prometheus response type: %s", result)
}

func (s *Client) QueryStats(query string, sourceTenants []string) (int, time.Duration, error) {
	if stats, found := s.cache.SourceTenantsData(sourceTenants).QueriesStats[query]; found {
		return stats.Series, stats.Duration, stats.Error
	}
	_, series, duration, err := s.Query(query, sourceTenants)
	stats := queryStats{Series: series, Duration: duration, Error: err}
	s.cache.SourceTenantsData(sourceTenants).QueriesStats[query] = stats
	return stats.Series, stats.Duration, stats.Error
}
