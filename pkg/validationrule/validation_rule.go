package validationrule

import (
	"github.com/fusakla/promruval/v2/pkg/config"
	"github.com/fusakla/promruval/v2/pkg/validator"
)

func New(name string, scope config.ValidationScope) *ValidationRule {
	return &ValidationRule{
		name:       name,
		scope:      scope,
		validators: []validator.Validator{},
	}
}

type ValidationRule struct {
	name       string
	scope      config.ValidationScope
	validators []validator.Validator
}

func (r *ValidationRule) Validators() []validator.Validator {
	return r.validators
}

func (r *ValidationRule) AddValidator(newValidator validator.Validator) {
	r.validators = append(r.validators, newValidator)
}

func (r *ValidationRule) Name() string {
	return r.name
}

func (r *ValidationRule) Scope() config.ValidationScope {
	return r.scope
}

func (r *ValidationRule) ValidationTexts() []string {
	var validationTexts []string
	for _, v := range r.validators {
		validationTexts = append(validationTexts, v.String())
	}
	return validationTexts
}
