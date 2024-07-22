package validationrule

import (
	"reflect"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/fusakla/promruval/v3/pkg/validator"
)

type ValidatorWithDetails interface {
	validator.Validator
	AdditionalDetails() string
	Name() string
}

type validatorWithAdditionalDetails struct {
	validator.Validator
	additionalDetails string
	name              string
}

func (v validatorWithAdditionalDetails) AdditionalDetails() string {
	return v.additionalDetails
}

func (v validatorWithAdditionalDetails) Name() string {
	return v.name
}

func New(name string, scope config.ValidationScope) *ValidationRule {
	return &ValidationRule{
		name:       name,
		scope:      scope,
		validators: make([]ValidatorWithDetails, 0),
	}
}

type ValidationRule struct {
	name       string
	scope      config.ValidationScope
	validators []ValidatorWithDetails
}

func (r *ValidationRule) Validators() []ValidatorWithDetails {
	return r.validators
}

func (r *ValidationRule) AddValidator(newValidator validator.Validator, additionalDetails string) {
	r.validators = append(r.validators, &validatorWithAdditionalDetails{
		Validator:         newValidator,
		additionalDetails: additionalDetails,
		name:              reflect.TypeOf(newValidator).Elem().Name(),
	})
}

func (r *ValidationRule) Name() string {
	return r.name
}

func (r *ValidationRule) Scope() config.ValidationScope {
	return r.scope
}

func (r *ValidationRule) ValidationTexts() []string {
	validationTexts := make([]string, 0, len(r.validators))
	for _, v := range r.validators {
		validationTexts = append(validationTexts, v.String())
	}
	return validationTexts
}
