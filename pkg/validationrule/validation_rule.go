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
	OnlyIf() []validator.Validator
}

type validatorWithAdditionalDetails struct {
	validator.Validator
	additionalDetails string
	name              string
	onlyIf            []validator.Validator
}

func (v validatorWithAdditionalDetails) AdditionalDetails() string {
	return v.additionalDetails
}

func (v validatorWithAdditionalDetails) Name() string {
	return v.name
}

func (v validatorWithAdditionalDetails) OnlyIf() []validator.Validator {
	return v.onlyIf
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

func (r *ValidationRule) AddValidator(newValidator validator.Validator, additionalDetails string, onlyIf []validator.Validator) {
	r.validators = append(r.validators, &validatorWithAdditionalDetails{
		Validator:         newValidator,
		additionalDetails: additionalDetails,
		name:              reflect.TypeOf(newValidator).Elem().Name(),
		onlyIf:            onlyIf,
	})
}

func (r *ValidationRule) Name() string {
	return r.name
}

func (r *ValidationRule) Scope() config.ValidationScope {
	return r.scope
}

func (r *ValidationRule) ValidationTexts() map[string][]string {
	validationTexts := make(map[string][]string, len(r.validators))

	for _, validator := range r.validators {
		key := validator.String()
		var value []string
		for _, onlyIfValidator := range validator.OnlyIf() {
		    value = append(value, onlyIfValidator.String())
		}
		validationTexts[key] = value
	}
	return validationTexts
}
