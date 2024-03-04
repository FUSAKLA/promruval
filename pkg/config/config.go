package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
)

const (
	AlertScope         ValidationScope = "Alert"
	RecordingRuleScope ValidationScope = "Recording rule"
	AllRulesScope      ValidationScope = "All rules"
	Group              ValidationScope = "Group"
)

var ValidationScopes = []ValidationScope{Group, AlertScope, RecordingRuleScope, AllRulesScope}

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
		fileData, err := os.ReadFile(c.ParamsFromFile)
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
