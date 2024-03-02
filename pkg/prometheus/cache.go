package prometheus

import (
	"encoding/json"
	"os"
	"time"

	"github.com/prometheus/common/model"

	log "github.com/sirupsen/logrus"
)

func newCache(file string, maxAge time.Duration) *cache {
	c := &cache{
		file:    file,
		Queries: map[string][]*model.Sample{},
		Labels:  []string{},
		Series:  map[string][]model.LabelSet{},
	}
	info, err := os.Stat(file)
	if err != nil {
		log.Warnf("cache file %s not found, skipping", file)
		return c
	}
	cacheAge := time.Since(info.ModTime())
	if maxAge != 0 && cacheAge > maxAge {
		log.Warnf("%s old cache %s is outdated, limit is %s", cacheAge, file, maxAge)
		return c
	}
	f, err := os.Open(file)
	if err != nil {
		log.Warnf("error opening cache file %s, skipping: %s", file, err)
		return c
	}
	err = json.NewDecoder(f).Decode(c)
	if err != nil {
		log.Warnf("invalid cache file `%s` format: %s", file, err)
		return c
	}
	return c
}

type cache struct {
	file    string
	Queries map[string][]*model.Sample  `json:"queries"`
	Labels  []string                    `json:"labels"`
	Series  map[string][]model.LabelSet `json:"series"`
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
