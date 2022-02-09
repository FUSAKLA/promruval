package validator

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ValidatorType string    `yaml:"type"`
	Params        yaml.Node `yaml:"params"`
}

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
}

func NewFromConfig(config Config) (Validator, error) {
	validatorFactory, ok := registeredValidators[config.ValidatorType]
	if !ok {
		return nil, fmt.Errorf("unknown validator type `%s`", config.ValidatorType)
	}
	return validatorFactory(config.Params)
}
