package prometheus

import (
	"encoding/json"
	"os"
	"time"

	"github.com/prometheus/common/model"

	log "github.com/sirupsen/logrus"
)

func newCache(file, prometheusURL string, maxAge time.Duration) *cache {
	emptyCache := cache{
		file:          file,
		PrometheusURL: prometheusURL,
		Created:       time.Now(),
		Queries:       make(map[string][]*model.Sample),
		Labels:        []string{},
		Series:        make(map[string][]model.LabelSet),
	}
	newCache := emptyCache
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
	if err := json.NewDecoder(f).Decode(&newCache); err != nil {
		log.Warnf("invalid cache file `%s` format: %s", file, err)
		return &emptyCache
	}
	pruneCache := false
	cacheAge := time.Since(newCache.Created)
	if maxAge != 0 && cacheAge > maxAge {
		log.Warnf("%s old cache %s is outdated, limit is %s. Using empty", cacheAge, file, maxAge)
		pruneCache = true
	}
	if newCache.PrometheusURL != prometheusURL {
		log.Warnf("cache %s is for different Prometheus URL %s, expected %s. Using empty", file, newCache.PrometheusURL, prometheusURL)
		pruneCache = true
	}
	if pruneCache {
		return &emptyCache
	}
	return &newCache
}

type cache struct {
	file          string
	PrometheusURL string                      `json:"prometheus_url"`
	Created       time.Time                   `json:"created"`
	Queries       map[string][]*model.Sample  `json:"queries"`
	Labels        []string                    `json:"labels"`
	Series        map[string][]model.LabelSet `json:"series"`
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
