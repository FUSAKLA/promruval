package validator

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/config"
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
}

func NewFromConfig(config config.ValidatorConfig) (Validator, error) {
	validatorFactory, ok := registeredValidators[config.ValidatorType]
	if !ok {
		return nil, fmt.Errorf("unknown validator type `%s`", config.ValidatorType)
	}
	return validatorFactory(config.Params)
}
