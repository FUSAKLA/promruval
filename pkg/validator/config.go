package validator

import (
	"fmt"

	"github.com/fusakla/promruval/v2/pkg/config"
	"gopkg.in/yaml.v3"
)

type validatorCreator func(params yaml.Node) (Validator, error)

var registeredValidators = map[string]validatorCreator{
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
}

func NewFromConfig(config config.ValidatorConfig) (Validator, error) {
	validatorFactory, ok := registeredValidators[config.ValidatorType]
	if !ok {
		return nil, fmt.Errorf("unknown validator type `%s`", config.ValidatorType)
	}
	return validatorFactory(config.Params)
}

func KnownValidatorName(name string) bool {
	if _, ok := registeredValidators[name]; ok {
		return true
	}
	return false
}
