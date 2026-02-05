package prometheus

import (
	"encoding/json"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func newCache(file, prometheusURL string, maxAge time.Duration) *cache {
	emptyCache := cache{
		file:          file,
		PrometheusURL: prometheusURL,
		Created:       time.Now(),
		SourceTenants: make(map[string]*cacheData),
	}
	previousCache := emptyCache
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(file)
			if err != nil {
				log.WithError(err).WithField("file", file).Warn("error creating cache file")
				return &emptyCache
			}
			emptyCacheJSON, err := json.Marshal(emptyCache)
			if err != nil {
				log.WithError(err).WithField("file", file).Warn("error creating cache file")
				return &emptyCache
			}
			_, err = f.Write(emptyCacheJSON)
			if err != nil {
				log.WithError(err).WithField("file", file).Warn("error writing empty cache file")
				return &emptyCache
			}
			f.Close()
		} else {
			log.WithError(err).WithField("file", file).Warn("error opening cache file, skipping")
			return &emptyCache
		}
	}
	if err := json.NewDecoder(f).Decode(&previousCache); err != nil {
		log.WithError(err).WithField("file", file).Warn("invalid cache file format")
		return &emptyCache
	}
	pruneCache := false
	cacheAge := time.Since(previousCache.Created)
	if maxAge != 0 && cacheAge > maxAge {
		log.WithFields(log.Fields{
			"cacheAge":    cacheAge,
			"maxCacheAge": maxAge,
			"file_name":   file,
		}).Info("cache is outdated")
		pruneCache = true
	}
	if previousCache.PrometheusURL != prometheusURL {
		log.WithFields(log.Fields{
			"previousPrometheusURL": previousCache.PrometheusURL,
			"newPrometheusURL":      prometheusURL,
			"file_name":             file,
		}).Info("data in cache file are from different Prometheus, cannot be used")
		pruneCache = true
	}
	if pruneCache {
		log.WithField("file", file).Warn("Pruning cache file")
		return &emptyCache
	}
	return &previousCache
}

type queryStats struct {
	Error    error         `json:"error,omitempty"`
	Series   int           `json:"series"`
	Duration time.Duration `json:"duration"`
}

type cacheData struct {
	QueriesStats           map[string]queryStats `json:"queries_stats"`
	KnownLabels            []string              `json:"known_labels"`
	SelectorMatchingSeries map[string]int        `json:"selector_matching_series"`
}
type cache struct {
	file          string
	PrometheusURL string                `json:"prometheus_url"`
	Created       time.Time             `json:"created"`
	SourceTenants map[string]*cacheData `json:"source_tenants"`
}

func (c *cache) SourceTenantsData(sourceTenants []string) *cacheData {
	key := sourceTenantsToHeader(sourceTenants)
	data, found := c.SourceTenants[key]
	if !found {
		data = &cacheData{
			QueriesStats:           make(map[string]queryStats),
			SelectorMatchingSeries: make(map[string]int),
			KnownLabels:            []string{},
		}
		c.SourceTenants[key] = data
	}
	return data
}

func (c *cache) Dump() {
	f, err := os.Create(c.file)
	if err != nil {
		log.WithError(err).WithField("file", c.file).Warn("failed to create cache file")
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	e := json.NewEncoder(f)
	e.SetIndent("", "")
	err = e.Encode(c)
	if err != nil {
		log.WithError(err).Warn("failed to write cache data")
		return
	}
	log.WithField("file_name", c.file).Info("successfully dumped cache to file")
}
