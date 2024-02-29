package validator

import (
	"fmt"
	"maps"

	"github.com/fusakla/promruval/v2/pkg/config"
	"gopkg.in/yaml.v3"
)

type validatorCreator func(params yaml.Node) (Validator, error)

var registeredRuleValidators = map[string]validatorCreator{
	"hasLabels":                            newHasLabels,
	"hasAnnotations":                       newHasAnnotations,
	"doesNotHaveLabels":                    newDoesNotHaveLabels,
	"doesNotHaveAnnotations":               newDoesNotHaveAnnotations,
	"hasAnyOfLabels":                       newHasAnyOfLabels,
	"hasAnyOfAnnotations":                  newHasAnyOfAnnotations,
	"labelMatchesRegexp":                   newLabelMatchesRegexp,
	"annotationMatchesRegexp":              newAnnotationMatchesRegexp,
	"labelHasAllowedValue":                 newLabelHasAllowedValue,
	"annotationHasAllowedValue":            newAnnotationHasAllowedValue,
	"annotationIsValidURL":                 newAnnotationIsValidURL,
	"expressionDoesNotUseLabels":           newExpressionDoesNotUseLabels,
	"expressionDoesNotUseOlderDataThan":    newExpressionDoesNotUseOlderDataThan,
	"expressionDoesNotUseRangeShorterThan": newExpressionDoesNotUseRangeShorterThan,
	"expressionDoesNotUseMetrics":          newExpressionDoesNotUseMetrics,
	"annotationIsValidPromQL":              newAnnotationIsValidPromQL,
	"validateAnnotationTemplates":          newValidateAnnotationTemplates,
	"forIsNotLongerThan":                   newForIsNotLongerThan,
	"expressionDoesNotUseIrate":            newExpressionDoesNotUseIrate,
	"validFunctionsOnCounters":             newValidFunctionsOnCounters,
	"rateBeforeAggregation":                newRateBeforeAggregation,
	"nonEmptyLabels":                       newNonEmptyLabels,
	"exclusiveLabels":                      newExclusiveLabels,
	"expressionCanBeEvaluated":             newExpressionCanBeEvaluated,
	"expressionUsesExistingLabels":         newExpressionUsesExistingLabels,
	"expressionSelectorsMatchesAnything":   newExpressionSelectorsMatchesAnything,
	"expressionWithNoMetricName":           newExpressionWithNoMetricName,
	"hasSourceTenantsForMetrics":           newHasSourceTenantsForMetrics,
}

var registeredGroupValidators = map[string]validatorCreator{
	"hasAllowedSourceTenants":      newHasAllowedSourceTenants,
	"hasAllowedEvaluationInterval": newHasAllowedEvaluationInterval,
	"hasValidPartialStrategy":      newHasValidPartialStrategy,
}

var registeredValidators = map[string]validatorCreator{}

func init() {
	maps.Copy(registeredValidators, registeredRuleValidators)
	maps.Copy(registeredValidators, registeredGroupValidators)
}

func NewFromConfig(scope config.ValidationScope, config config.ValidatorConfig) (Validator, error) {
	factory, ok := creator(scope, config.ValidatorType)
	if !ok {
		return nil, fmt.Errorf("unknown validator type `%s`", config.ValidatorType)
	}
	return factory(config.Params)
}

func creator(scope config.ValidationScope, name string) (validatorCreator, bool) {
	validators := registeredRuleValidators
	if scope == config.Group {
		validators = registeredGroupValidators
	}
	creator, ok := validators[name]
	return creator, ok
}

func KnownValidators(scope config.ValidationScope, validatorNames []string) error {
	for _, validatorName := range validatorNames {
		if _, ok := creator(scope, validatorName); !ok {
			return fmt.Errorf("unknown validator `%s` for given validation rule scope %s", validatorName, scope)
		}
	}
	return nil
}
