package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
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
	"github.com/ybbus/httpretry"
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
		token, err := os.ReadFile(promConfig.BearerTokenFile)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", promConfig.BearerTokenFile, err)
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
		Address: promConfig.URL,
		RoundTripper: http.RoundTripper(&httpretry.RetryRoundtripper{
			Next:          prom_config.NewHeadersRoundTripper(&headers, tripper),
			MaxRetryCount: promConfig.MaxRetries,
			ShouldRetry: func(statusCode int, err error) bool {
				switch statusCode {
				case // status codes that should be retried
					http.StatusRequestTimeout,
					http.StatusConflict,
					http.StatusLocked,
					http.StatusTooManyRequests,
					http.StatusInternalServerError,
					http.StatusBadGateway,
					http.StatusServiceUnavailable,
					http.StatusGatewayTimeout:
					return true
				case 0: // means we did not get a response. we need to retry
					return true
				}
				return false
			},
			CalculateBackoff: httpretry.ExponentialBackoff(1*time.Second, promConfig.MaxRetryWait, 200*time.Millisecond),
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus client: %w", err)
	}
	v1cli := v1.NewAPI(cli)
	var cacheInstance *cache
	if promConfig.DisableCache {
		cacheInstance = nil
	} else {
		cacheInstance = newCache(promConfig.CacheFile, promConfig.URL, promConfig.MaxCacheAge)
	}
	promClient := Client{
		apiClient:             v1cli,
		httpHeaders:           &headers,
		originalSourceTenants: originalSourceTenants,
		httpHeadersMtx:        sync.Mutex{},
		url:                   promConfig.URL,
		timeout:               promConfig.Timeout,
		queryOffset:           promConfig.QueryOffset,
		queryLookback:         promConfig.QueryLookback,
		cache:                 cacheInstance,
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
	if s.cache == nil {
		return
	}
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
	var cache *cacheData
	if s.cache != nil {
		cache = s.cache.SourceTenantsData(sourceTenants)
		if count, found := cache.MatchingSeriesForSelector(selector); found {
			return count, nil
		}
	}
	series, err := s.SelectorMatch(selector, sourceTenants)
	if err != nil {
		return 0, err
	}
	if s.cache != nil {
		cache.SetSelectorMatchingSeries(selector, len(series))
	}
	return len(series), nil
}

func (s *Client) Labels(sourceTenants []string) ([]string, error) {
	var cachedLabels []string
	var cache *cacheData
	if s.cache != nil {
		cache = s.cache.SourceTenantsData(sourceTenants)
		cachedLabels = cache.GetKnownLabels()
	}
	if s.cache == nil || len(cachedLabels) == 0 {
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
		if cache != nil {
			cache.SetKnownLabels(result)
		}
		return result, nil
	}
	return cachedLabels, nil
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
	var cache *cacheData
	if s.cache != nil {
		cache = s.cache.SourceTenantsData(sourceTenants)
		if stats, found := cache.GetQueryStats(query); found {
			return stats.Series, stats.Duration, stats.Error
		}
	}
	_, series, duration, err := s.Query(query, sourceTenants)
	if cache != nil {
		stats := queryStats{Series: series, Duration: duration, Error: err}
		cache.SetQueryStats(query, stats)
	}
	return series, duration, err
}
