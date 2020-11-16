package config

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/validator"
)

const (
	AlertScope         ValidationScope = "Alert"
	RecordingRuleScope ValidationScope = "RecordingRule"
	AllRulesScope      ValidationScope = "AllRules"
)

var ValidationScopes = []ValidationScope{AlertScope, RecordingRuleScope, AllRulesScope}

type Config struct {
	ValidationRules []ValidationRule `yaml:"validationRules"`
}

type ValidationRule struct {
	Name        string             `yaml:"name"`
	Scope       ValidationScope    `yaml:"scope"`
	Validations []validator.Config `yaml:"validations"`
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
	return fmt.Errorf("invalid vaidation scope `%s`", ruleType)
}
