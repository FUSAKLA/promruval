package extractvalidators

import (
	"fmt"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/fusakla/promruval/v3/pkg/validationrule"
	"github.com/fusakla/promruval/v3/pkg/validator"
)

func ValidatorFromConfig(scope config.ValidationScope, validatorType string, validatorConfig config.ValidatorConfig) (validator.Validator, error) {
	if err := validator.KnownValidators(scope, []string{validatorType}); err != nil {
		return nil, fmt.Errorf("error loading config for validator `%s`: %w", validatorType, err)
	}
	newValidator, err := validator.NewFromConfig(scope, validatorConfig)
	if err != nil {
		return nil, fmt.Errorf("loading only if config for validator `%s`: %w", validatorType, err)
	}
	return newValidator, nil
}

func ValidationRulesFromConfig(validationConfig *config.Config, disabledRules, enabledRules []string) ([]*validationrule.ValidationRule, error) {
	var validationRules []*validationrule.ValidationRule
rulesIteration:
	for _, validationRule := range validationConfig.ValidationRules {
		if validationRule.Scope == "" {
			return nil, fmt.Errorf("scope is missing in the validation rule `%s`", validationRule.Name)
		}
		for _, disabledRule := range disabledRules {
			if disabledRule == validationRule.Name {
				continue rulesIteration
			}
		}
		for _, enabledRule := range enabledRules {
			if enabledRule != validationRule.Name {
				continue rulesIteration
			}
		}
		newRule := validationrule.New(validationRule.Name, validationRule.Scope)
		for _, validatorConfig := range validationRule.OnlyIf {
			// Do not limit the scope of onlyIf validators, will be applied only to the entities where possible
			v, err := ValidatorFromConfig(config.AllScope, validatorConfig.ValidatorType, validatorConfig)
			if err != nil {
				return nil, fmt.Errorf("loading config for onlyIf validator in the `%s` rule: %w", validationRule.Name, err)
			}
			if v == nil {
				continue
			}
			newRule.AddOnlyIfValidator(v, validatorConfig.AdditionalDetails)
		}
		for _, validatorConfig := range validationRule.Validations {
			v, err := ValidatorFromConfig(validationRule.Scope, validatorConfig.ValidatorType, validatorConfig)
			if err != nil {
				return nil, fmt.Errorf("loading config for validator in the `%s` rule: %w", validationRule.Name, err)
			}
			if v == nil {
				continue
			}
			newRule.AddValidator(v, validatorConfig.AdditionalDetails)
		}
		validationRules = append(validationRules, newRule)
	}
	return validationRules, nil
}
