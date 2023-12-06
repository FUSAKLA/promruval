package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/fusakla/promruval/v2/pkg/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

func NewClient(config config.PrometheusConfig) (*Client, error) {
	return NewClientWithRoundTripper(config, &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.InsecureSkipTLSVerify}})
}

func NewClientWithRoundTripper(config config.PrometheusConfig, tripper http.RoundTripper) (*Client, error) {
	cli, err := api.NewClient(api.Config{
		Address:      config.URL,
		RoundTripper: tripper,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus client: %w", err)
	}
	v1cli := v1.NewAPI(cli)
	promClient := Client{
		apiClient: v1cli,
		url:       config.URL,
		timeout:   config.Timeout,
		cache:     newCache(config.CacheFile, config.MaxCacheAge),
	}
	return &promClient, nil
}

type Client struct {
	apiClient v1.API
	url       string
	timeout   time.Duration
	cache     *cache
}

func (s *Client) DumpCache() {
	start := time.Now()
	s.cache.Dump()
	log.Info("cache dumped in ", time.Since(start))
}

func (s *Client) newContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	if s.timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	return ctx, cancel
}

func (s *Client) SelectorMatch(selector string) ([]model.LabelSet, error) {
	if _, ok := s.cache.Series[selector]; !ok {
		ctx, cancel := s.newContext()
		defer cancel()
		result, warnings, err := s.apiClient.Series(ctx, []string{selector}, time.Now().Add(-time.Minute), time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to query series: %w", err)
		}
		if len(warnings) > 0 {
			log.Warnf("Warning querying Prometheus: %s\n", warnings)
		}
		s.cache.Series[selector] = result
	} else {
		log.Debugf("using cached series match result for `%s`", selector)
	}
	return s.cache.Series[selector], nil
}

func (s *Client) Labels() ([]string, error) {
	if len(s.cache.Labels) == 0 {
		ctx, cancel := s.newContext()
		defer cancel()
		start := time.Now()
		result, warnings, err := s.apiClient.LabelNames(ctx, []string{}, time.Now().Add(-time.Minute), time.Now())
		log.Infof("loaded all prometheus label names from %s in %s", s.url, time.Since(start))
		if err != nil {
			return nil, err
		}
		if len(warnings) > 0 {
			log.Warnf("Warning querying Prometheus: %s\n", warnings)
		}
		s.cache.Labels = result
	}
	return s.cache.Labels, nil
}

func (s *Client) Query(query string) ([]*model.Sample, int, time.Duration, error) {
	var duration time.Duration
	if _, ok := s.cache.Queries[query]; !ok {
		ctx, cancel := s.newContext()
		defer cancel()
		start := time.Now()
		result, warnings, err := s.apiClient.Query(ctx, query, time.Now())
		duration = time.Since(start)
		log.Infof("query `%s` on %s prometheus took %s", query, s.url, duration)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("error querying prometheus: %w", err)
		}
		if len(warnings) > 0 {
			log.Warnf("Warning querying Prometheus: %s\n", warnings)
		}
		switch result.Type() {
		case model.ValVector:
			s.cache.Queries[query] = result.(model.Vector)
		default:
			return nil, 0, 0, fmt.Errorf("unknown prometheus response type: %s", result)
		}
	} else {
		log.Debugf("using cached query result for `%s`", query)
	}
	res := s.cache.Queries[query]
	return res, len(res), duration, nil
}
