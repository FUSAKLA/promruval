package prometheus

import (
	"encoding/json"
	"os"
	"slices"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func newCache(file, prometheusURL string, maxAge time.Duration) *cache {
	emptyCache := cache{
		file:          file,
		PrometheusURL: prometheusURL,
		Created:       time.Now(),
		SourceTenants: make(map[string]*cacheData),
		mtx:           sync.RWMutex{},
	}
	previousCache := &emptyCache
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(file)
			if err != nil {
				log.WithError(err).WithField("file", file).Warn("error creating cache file")
				return &emptyCache
			}
			emptyCacheJSON, err := json.Marshal(&emptyCache)
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
	return previousCache
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
	mtx                    sync.RWMutex          `json:"-"`
}

func (c *cacheData) MatchingSeriesForSelector(selector string) (int, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	count, found := c.SelectorMatchingSeries[selector]
	return count, found
}

func (c *cacheData) SetSelectorMatchingSeries(selector string, count int) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.SelectorMatchingSeries[selector] = count
}

func (c *cacheData) GetKnownLabels() []string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return slices.Clone(c.KnownLabels)
}

func (c *cacheData) SetKnownLabels(labels []string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.KnownLabels = slices.Clone(labels)
}

func (c *cacheData) GetQueryStats(query string) (queryStats, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	stats, found := c.QueriesStats[query]
	return stats, found
}

func (c *cacheData) SetQueryStats(query string, stats queryStats) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.QueriesStats[query] = stats
}

type cache struct {
	file          string
	PrometheusURL string                `json:"prometheus_url"`
	Created       time.Time             `json:"created"`
	SourceTenants map[string]*cacheData `json:"source_tenants"`
	mtx           sync.RWMutex          `json:"-"`
}

func (c *cache) SourceTenantsData(sourceTenants []string) *cacheData {
	c.mtx.RLock()
	key := sourceTenantsToHeader(sourceTenants)
	data, found := c.SourceTenants[key]
	if found {
		c.mtx.RUnlock()
		return data
	}
	c.mtx.RUnlock()
	data = &cacheData{
		QueriesStats:           make(map[string]queryStats),
		SelectorMatchingSeries: make(map[string]int),
		KnownLabels:            []string{},
		mtx:                    sync.RWMutex{},
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.SourceTenants[key] = data
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
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	err = e.Encode(c)
	if err != nil {
		log.WithError(err).Warn("failed to write cache data")
		return
	}
	log.WithField("file_name", c.file).Info("successfully dumped cache to file")
}
