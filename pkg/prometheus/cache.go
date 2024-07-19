package prometheus

import (
	"encoding/json"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func newCache(file, prometheusURL string, maxAge time.Duration) *cache {
	emptyCache := cache{
		file:                   file,
		PrometheusURL:          prometheusURL,
		Created:                time.Now(),
		QueriesStats:           make(map[string]queryStats),
		KnownLabels:            []string{},
		SelectorMatchingSeries: make(map[string]int),
	}
	previousCache := emptyCache
	f, err := os.Open(file)
	if err != nil {
		if !os.IsNotExist(err) {
			f, err = os.Create(file)
			if err != nil {
				log.Warnf("error creating cache file %s: %s", file, err)
				return &emptyCache
			}
		} else {
			log.Warnf("error opening cache file %s, skipping: %s", file, err)
			return &emptyCache
		}
	}
	if err := json.NewDecoder(f).Decode(&previousCache); err != nil {
		log.Warnf("invalid cache file `%s` format: %s", file, err)
		return &emptyCache
	}
	pruneCache := false
	cacheAge := time.Since(previousCache.Created)
	if maxAge != 0 && cacheAge > maxAge {
		log.Infof("%s old cache %s is outdated, limit is %s", cacheAge, file, maxAge)
		pruneCache = true
	}
	if previousCache.PrometheusURL != prometheusURL {
		log.Infof("data in cache file %s are from different Prometheus on URL %s, cannot be used for the instance on %s URL", file, previousCache.PrometheusURL, prometheusURL)
		pruneCache = true
	}
	if pruneCache {
		log.Warnf("Pruning cache file %s", file)
		return &emptyCache
	}
	return &previousCache
}

type queryStats struct {
	Error    error         `json:"error,omitempty"`
	Series   int           `json:"series"`
	Duration time.Duration `json:"duration"`
}
type cache struct {
	file                   string
	PrometheusURL          string                `json:"prometheus_url"`
	Created                time.Time             `json:"created"`
	QueriesStats           map[string]queryStats `json:"queries_stats"`
	KnownLabels            []string              `json:"known_labels"`
	SelectorMatchingSeries map[string]int        `json:"selector_matching_series"`
}

func (c *cache) Dump() {
	f, err := os.Create(c.file)
	if err != nil {
		log.Warnf("failed to create cache file %s: %s", c.file, err)
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(c)
	if err != nil {
		log.Warnf("failed to write cache data: %s", err)
		return
	}
	log.Infof("successfully dumped cache to file %s", c.file)
}
