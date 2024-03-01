package validator

import (
	"fmt"
	"maps"

	"github.com/fusakla/promruval/v2/pkg/config"
	"gopkg.in/yaml.v3"
)

type validatorCreator func(params yaml.Node) (Validator, error)

var registeredUniversalRuleValidators = map[string]validatorCreator{
	"hasLabels":            newHasLabels,
	"doesNotHaveLabels":    newDoesNotHaveLabels,
	"hasAnyOfLabels":       newHasAnyOfLabels,
	"labelMatchesRegexp":   newLabelMatchesRegexp,
	"labelHasAllowedValue": newLabelHasAllowedValue,
	"nonEmptyLabels":       newNonEmptyLabels,
	"exclusiveLabels":      newExclusiveLabels,

	"validFunctionsOnCounters":             newValidFunctionsOnCounters,
	"rateBeforeAggregation":                newRateBeforeAggregation,
	"expressionDoesNotUseLabels":           newExpressionDoesNotUseLabels,
	"expressionDoesNotUseOlderDataThan":    newExpressionDoesNotUseOlderDataThan,
	"expressionDoesNotUseRangeShorterThan": newExpressionDoesNotUseRangeShorterThan,
	"expressionDoesNotUseMetrics":          newExpressionDoesNotUseMetrics,
	"expressionDoesNotUseIrate":            newExpressionDoesNotUseIrate,
	"expressionCanBeEvaluated":             newExpressionCanBeEvaluated,
	"expressionUsesExistingLabels":         newExpressionUsesExistingLabels,
	"expressionSelectorsMatchesAnything":   newExpressionSelectorsMatchesAnything,
	"expressionWithNoMetricName":           newExpressionWithNoMetricName,
	"expressionIsWellFormatted":            newExpressionIsWellFormatted,

	"hasSourceTenantsForMetrics": newHasSourceTenantsForMetrics,
}

var registeredRecordingRuleValidators = map[string]validatorCreator{}

var registeredAlertValidators = map[string]validatorCreator{
	"forIsNotLongerThan": newForIsNotLongerThan,

	"validateAnnotationTemplates": newValidateAnnotationTemplates,
	"annotationIsValidPromQL":     newAnnotationIsValidPromQL,
	"annotationHasAllowedValue":   newAnnotationHasAllowedValue,
	"annotationIsValidURL":        newAnnotationIsValidURL,
	"hasAnnotations":              newHasAnnotations,
	"doesNotHaveAnnotations":      newDoesNotHaveAnnotations,
	"annotationMatchesRegexp":     newAnnotationMatchesRegexp,
	"hasAnyOfAnnotations":         newHasAnyOfAnnotations,
	"validateLabelTemplates":      newValidateLabelTemplates,
}

var registeredGroupValidators = map[string]validatorCreator{
	"hasAllowedSourceTenants":      newHasAllowedSourceTenants,
	"hasAllowedEvaluationInterval": newHasAllowedEvaluationInterval,
	"hasValidPartialStrategy":      newHasValidPartialStrategy,
	"maxRulesPerGroup":             newMaxRulesPerGroup,
	"hasAllowedLimit":              newHasAllowedLimit,
}

var (
	alertValidators         = map[string]validatorCreator{}
	recordingRuleValidators = map[string]validatorCreator{}
)

func init() {
	maps.Copy(alertValidators, registeredUniversalRuleValidators)
	maps.Copy(alertValidators, registeredAlertValidators)

	maps.Copy(recordingRuleValidators, registeredUniversalRuleValidators)
	maps.Copy(recordingRuleValidators, registeredRecordingRuleValidators)
}

func NewFromConfig(scope config.ValidationScope, config config.ValidatorConfig) (Validator, error) {
	factory, ok := creator(scope, config.ValidatorType)
	if !ok {
		return nil, fmt.Errorf("unknown validator type `%s`", config.ValidatorType)
	}
	return factory(config.Params)
}

func creator(scope config.ValidationScope, name string) (validatorCreator, bool) {
	var validators map[string]validatorCreator
	switch scope {
	case config.AlertScope:
		validators = alertValidators
	case config.RecordingRuleScope:
		validators = recordingRuleValidators
	case config.Group:
		validators = registeredGroupValidators
	case config.AllRulesScope:
		validators = registeredUniversalRuleValidators
	}
	creator, ok := validators[name]
	return creator, ok
}

func KnownValidators(scope config.ValidationScope, validatorNames []string) error {
	for _, validatorName := range validatorNames {
		if _, ok := creator(scope, validatorName); !ok {
			return fmt.Errorf("unknown validator `%s` for given validation rule scope %s, see the docs/validations.md for the complete list and allowed scopes", validatorName, scope)
		}
	}
	return nil
}
