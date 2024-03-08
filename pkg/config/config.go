package config

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
)

const (
	AlertScope         ValidationScope = "Alert"
	RecordingRuleScope ValidationScope = "Recording rule"
	AllRulesScope      ValidationScope = "All rules"
	GroupScope         ValidationScope = "Group"
	AllScope           ValidationScope = "All"
)

var ValidationScopes = []ValidationScope{GroupScope, AlertScope, RecordingRuleScope, AllRulesScope}

// Ugly hack with a global variable to be able to use it in UnmarshalYAML.
// Not sure how to better propagate some context to the UnmarshalYAML function.
var (
	configDir    string
	configDirMtx sync.Mutex
)

func init() {
	configDirMtx = sync.Mutex{}
}

func NewLoader(cfgPath string) Loader {
	return Loader{ConfigPath: cfgPath}
}

type Loader struct {
	ConfigPath string
}

func (l *Loader) Load() (*Config, error) {
	configFile, err := os.Open(l.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	configDirMtx.Lock()
	configDir = path.Dir(l.ConfigPath)
	defer func() {
		configDir = ""
		configDirMtx.Unlock()
	}()
	validationConfig := Config{}
	decoder := yaml.NewDecoder(configFile)
	decoder.KnownFields(true)
	if err := decoder.Decode(&validationConfig); err != nil {
		return nil, fmt.Errorf("loading config file: %w", err)
	}
	return &validationConfig, nil
}

type Config struct {
	CustomExcludeAnnotation string           `yaml:"customExcludeAnnotation"`
	CustomDisableComment    string           `yaml:"customDisableComment"`
	ValidationRules         []ValidationRule `yaml:"validationRules"`
	Prometheus              PrometheusConfig `yaml:"prometheus"`
}

type PrometheusConfig struct {
	URL                   string        `yaml:"url"`
	Timeout               time.Duration `yaml:"timeout" default:"30s"`
	InsecureSkipTLSVerify bool          `yaml:"insecureSkipTlsVerify"`
	CacheFile             string        `yaml:"cacheFile,omitempty" default:".promruval_cache.json"`
	MaxCacheAge           time.Duration `yaml:"maxCacheAge,omitempty" default:"1h"`
}

func (c *PrometheusConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := defaults.Set(c)
	if err != nil {
		return err
	}

	type plain PrometheusConfig
	err = unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	return nil
}

type ValidationRule struct {
	Name        string            `yaml:"name"`
	Scope       ValidationScope   `yaml:"scope"`
	Validations []ValidatorConfig `yaml:"validations"`
}

type ValidatorConfig struct {
	ValidatorType     string    `yaml:"type"`
	AdditionalDetails string    `yaml:"additionalDetails"`
	Params            yaml.Node `yaml:"params"`
	ParamsFromFile    string    `yaml:"paramsFromFile"`
}

func (c *ValidatorConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ValidatorConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.ParamsFromFile != "" {
		if !c.Params.IsZero() {
			return fmt.Errorf("cannot use both `params` and `paramsFromFile`")
		}
		if path.IsAbs(c.ParamsFromFile) {
			return fmt.Errorf("`paramsFromFile` must be a relative path to the config file")
		}
		fileData, err := os.ReadFile(path.Join(configDir, c.ParamsFromFile))
		if err != nil {
			return fmt.Errorf("cannot read params from file %s: %w", c.ParamsFromFile, err)
		}
		err = yaml.Unmarshal(fileData, &c.Params)
		if err != nil {
			return err
		}
	}
	return nil
}

type ValidationScope string

func (t *ValidationScope) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ruleType string
	err := unmarshal(&ruleType)
	if err != nil {
		return err
	}
	for _, scope := range ValidationScopes {
		if string(scope) == ruleType {
			*t = scope
			return nil
		}
	}
	return fmt.Errorf("invalid validation scope `%s`", ruleType)
}
