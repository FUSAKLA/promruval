package config

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
	"github.com/google/go-jsonnet"
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

func BaseDirPath() string {
	configDirMtx.Lock()
	defer configDirMtx.Unlock()
	return configDir
}

func NewLoader(cfgPath string) Loader {
	return Loader{ConfigPath: cfgPath}
}

type Loader struct {
	ConfigPath string
}

func (l *Loader) Load() (*Config, error) {
	var configFile io.ReadCloser
	configFile, err := os.Open(l.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer configFile.Close()
	configDirMtx.Lock()
	configDir = path.Dir(l.ConfigPath)
	defer func() {
		configDirMtx.Unlock()
	}()
	validationConfig := Config{}

	// If the config file is a jsonnet file, evaluate it first
	if strings.HasSuffix(l.ConfigPath, ".jsonnet") {
		jsonnetVM := jsonnet.MakeVM()
		jsonStr, err := jsonnetVM.EvaluateFile(l.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("evaluating jsonnet in config file %s: %w", l.ConfigPath, err)
		}
		configFile = io.NopCloser(strings.NewReader(jsonStr))
	}

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
	URL                   string            `yaml:"url"`
	Timeout               time.Duration     `yaml:"timeout" default:"30s"`
	InsecureSkipTLSVerify bool              `yaml:"insecureSkipTlsVerify"`
	DisableCache          bool              `yaml:"disableCache"`
	CacheFile             string            `yaml:"cacheFile,omitempty" default:".promruval_cache.json"`
	MaxCacheAge           time.Duration     `yaml:"maxCacheAge,omitempty" default:"1h"`
	BearerTokenFile       string            `yaml:"bearerTokenFile,omitempty"`
	QueryOffset           time.Duration     `yaml:"queryOffset,omitempty" default:"1m"`
	QueryLookback         time.Duration     `yaml:"queryLookback,omitempty" default:"20m"`
	HTTPHeaders           map[string]string `yaml:"httpHeaders,omitempty"`
	MaxRetries            int               `yaml:"maxRetries,omitempty" default:"0"`
	MaxRetryWait          time.Duration     `yaml:"maxRetryWait,omitempty" default:"30s"`
}

func defaultPrometheusConfig() (PrometheusConfig, error) {
	def := PrometheusConfig{}
	if err := defaults.Set(&def); err != nil {
		return def, err
	}
	return def, nil
}

func (c *PrometheusConfig) PatchNonDefault(other PrometheusConfig) error {
	def, err := defaultPrometheusConfig()
	if err != nil {
		return err
	}
	if other.URL != def.URL {
		c.URL = other.URL
	}
	if other.Timeout != def.Timeout {
		c.Timeout = other.Timeout
	}
	if other.InsecureSkipTLSVerify != def.InsecureSkipTLSVerify {
		c.InsecureSkipTLSVerify = other.InsecureSkipTLSVerify
	}
	if other.DisableCache != def.DisableCache {
		c.DisableCache = other.DisableCache
	}
	if other.CacheFile != def.CacheFile {
		c.CacheFile = other.CacheFile
	}
	if other.MaxCacheAge != def.MaxCacheAge {
		c.MaxCacheAge = other.MaxCacheAge
	}
	if other.BearerTokenFile != def.BearerTokenFile {
		c.BearerTokenFile = other.BearerTokenFile
	}
	if other.QueryOffset != def.QueryOffset {
		c.QueryOffset = other.QueryOffset
	}
	if other.QueryLookback != def.QueryLookback {
		c.QueryLookback = other.QueryLookback
	}
	if len(other.HTTPHeaders) > 0 {
		if c.HTTPHeaders == nil {
			c.HTTPHeaders = map[string]string{}
		}
		for k, v := range other.HTTPHeaders {
			c.HTTPHeaders[k] = v
		}
	}
	if other.MaxRetries != def.MaxRetries {
		c.MaxRetries = other.MaxRetries
	}
	if other.MaxRetryWait != def.MaxRetryWait {
		c.MaxRetryWait = other.MaxRetryWait
	}
	return nil
}

func (c *PrometheusConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := defaults.Set(c)
	if err != nil {
		return err
	}

	type plain PrometheusConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if c.BearerTokenFile != "" {
		if path.IsAbs(c.BearerTokenFile) {
			return fmt.Errorf("`bearerTokenFile` must be a relative path to the config file")
		}
		c.BearerTokenFile = path.Join(configDir, c.BearerTokenFile)
	}

	return nil
}

type ValidationRule struct {
	Name        string            `yaml:"name"`
	Scope       ValidationScope   `yaml:"scope"`
	OnlyIf      []ValidatorConfig `yaml:"onlyIf"`
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

func LoadConfigFile(configFilePath string) (*Config, error) {
	configLoader := NewLoader(configFilePath)
	return configLoader.Load()
}

func LoadConfiguration(configFilePaths []string) (*Config, error) {
	if len(configFilePaths) == 0 {
		return nil, fmt.Errorf("required flag --config-file not provided, try --help")
	}
	mainConfig, err := LoadConfigFile(configFilePaths[0])
	if err != nil {
		return nil, fmt.Errorf("error loading config file %s: %w", configFilePaths[0], err)
	}
	for _, cf := range configFilePaths[1:] {
		validationConfig, err := LoadConfigFile(cf)
		if err != nil {
			return nil, fmt.Errorf("error loading config file %s: %w", cf, err)
		}
		if err := mainConfig.Prometheus.PatchNonDefault(validationConfig.Prometheus); err != nil {
			return nil, fmt.Errorf("error merging prometheus config from file %s: %w", cf, err)
		}
		if validationConfig.CustomExcludeAnnotation != "" {
			mainConfig.CustomExcludeAnnotation = validationConfig.CustomExcludeAnnotation
		}
		if validationConfig.CustomDisableComment != "" {
			mainConfig.CustomDisableComment = validationConfig.CustomDisableComment
		}
		mainConfig.ValidationRules = append(mainConfig.ValidationRules, validationConfig.ValidationRules...)
	}

	return mainConfig, nil
}
