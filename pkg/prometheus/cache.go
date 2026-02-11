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

// cacheData holds cached query data for a specific set of source tenants.
// All fields are protected by the embedded mutex for thread-safety.
type cacheData struct {
	QueriesStats           map[string]queryStats `json:"queries_stats"`            // Protected by mtx
	KnownLabels            []string              `json:"known_labels"`             // Protected by mtx
	SelectorMatchingSeries map[string]int        `json:"selector_matching_series"` // Protected by mtx
	mtx                    sync.RWMutex          `json:"-"`                        // Protects all above fields
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

// cache represents a thread-safe cache for Prometheus query data.
//
// Concurrency Design:
// - cache.mtx protects the SourceTenants map (main cache structure)
// - Each cacheData.mtx protects the individual cache data fields
// - Lock ordering: always acquire cache.mtx before cacheData.mtx to prevent deadlocks
// - SourceTenantsData uses double-checked locking to prevent race conditions
// - Dump creates a snapshot to avoid holding locks during JSON serialization.
type cache struct {
	file          string
	PrometheusURL string                `json:"prometheus_url"`
	Created       time.Time             `json:"created"`
	SourceTenants map[string]*cacheData `json:"source_tenants"` // Protected by mtx
	mtx           sync.RWMutex          `json:"-"`              // Protects SourceTenants map
}

func (c *cache) SourceTenantsData(sourceTenants []string) *cacheData {
	key := sourceTenantsToHeader(sourceTenants)

	// First check - read lock
	c.mtx.RLock()
	data, found := c.SourceTenants[key]
	if found {
		c.mtx.RUnlock()
		return data
	}
	c.mtx.RUnlock()

	// Create new cacheData outside of locks
	newData := &cacheData{
		QueriesStats:           make(map[string]queryStats),
		SelectorMatchingSeries: make(map[string]int),
		KnownLabels:            []string{},
		mtx:                    sync.RWMutex{},
	}

	// Second check - write lock (double-checked locking pattern)
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Check again in case another goroutine created it while we were waiting for the lock
	data, found = c.SourceTenants[key]
	if found {
		return data // Another goroutine already created it
	}

	// We're the first to create it
	c.SourceTenants[key] = newData
	return newData
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

	// Create a snapshot of the cache data to avoid race conditions during JSON encoding
	c.mtx.RLock()
	snapshot := cache{
		file:          c.file,
		PrometheusURL: c.PrometheusURL,
		Created:       c.Created,
		SourceTenants: make(map[string]*cacheData),
	}

	// Create deep copies of all cacheData with proper locking
	for key, data := range c.SourceTenants {
		// Defensive nil check (shouldn't happen but good practice)
		if data == nil {
			log.WithField("key", key).Warn("Found nil cacheData in SourceTenants, skipping")
			continue
		}

		data.mtx.RLock()
		snapshot.SourceTenants[key] = &cacheData{
			QueriesStats:           make(map[string]queryStats),
			KnownLabels:            make([]string, len(data.KnownLabels)),
			SelectorMatchingSeries: make(map[string]int),
		}

		// Copy maps and slices
		for k, v := range data.QueriesStats {
			snapshot.SourceTenants[key].QueriesStats[k] = v
		}
		copy(snapshot.SourceTenants[key].KnownLabels, data.KnownLabels)
		for k, v := range data.SelectorMatchingSeries {
			snapshot.SourceTenants[key].SelectorMatchingSeries[k] = v
		}
		data.mtx.RUnlock()
	}
	c.mtx.RUnlock()

	// Now encode the snapshot without any locks held
	e := json.NewEncoder(f)
	e.SetIndent("", "")
	err = e.Encode(&snapshot)
	if err != nil {
		log.WithError(err).Warn("failed to write cache data")
		return
	}
	log.WithField("file_name", c.file).Info("successfully dumped cache to file")
}
