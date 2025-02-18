package validationrule

import (
	"fmt"
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
	onlyIf     []ValidatorWithDetails
	validators []ValidatorWithDetails
}

func (r *ValidationRule) Validators() []ValidatorWithDetails {
	return r.validators
}

func (r *ValidationRule) OnlyIf() []ValidatorWithDetails {
	return r.onlyIf
}

func (r *ValidationRule) AddValidator(newValidator validator.Validator, additionalDetails string) {
	r.validators = append(r.validators, &validatorWithAdditionalDetails{
		Validator:         newValidator,
		additionalDetails: additionalDetails,
		name:              reflect.TypeOf(newValidator).Elem().Name(),
	})
}

func (r *ValidationRule) AddOnlyIfValidator(newValidator validator.Validator, additionalDetails string) {
	r.onlyIf = append(r.onlyIf, &validatorWithAdditionalDetails{
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

func validatorTextWithScope(v ValidatorWithDetails, scope config.ValidationScope) string {
	scopeText := string(scope)
	if scope == config.AllRulesScope {
		scopeText = "Rule"
	}
	return fmt.Sprintf("%s %s", scopeText, v.String())
}

func (r *ValidationRule) ValidationTexts() []string {
	validationTexts := make([]string, 0, len(r.validators))
	for _, v := range r.validators {
		validationTexts = append(validationTexts, validatorTextWithScope(v, r.Scope()))
	}
	return validationTexts
}

func (r *ValidationRule) OnlyIfValidationTexts() []string {
	validationTexts := make([]string, 0, len(r.onlyIf))
	for _, v := range r.onlyIf {
		validationTexts = append(validationTexts, validatorTextWithScope(v, validator.Scope(v.Name())))
	}
	return validationTexts
}
