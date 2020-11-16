package validator

import (
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"gotest.tools/assert"
	"regexp"
	"testing"
	"time"
)

var testCases = []struct {
	name           string
	validator      Validator
	rule           rulefmt.Rule
	expectedErrors int
}{
	// hasLabels
	{name: "ruleHasExpectedLabel", validator: hasLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar", "foo2": "bar2"}}, expectedErrors: 0},
	{name: "ruleMissingExpectedLabel", validator: hasLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"xxx": "yyy"}}, expectedErrors: 2},

	// hasAnnotations
	{name: "ruleHasExpectedAnnotation", validator: hasAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar", "foo2": "bar2"}}, expectedErrors: 0},
	{name: "ruleMissingExpectedAnnotation", validator: hasAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"xxx": "yyy"}}, expectedErrors: 2},

	// doesNotHaveLabels
	{name: "ruleDoesNotHaveForbiddenLabel", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"xxx": "yyy"}}, expectedErrors: 0},
	{name: "ruleHaveForbiddenLabel", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// doesNotHaveAnnotations
	{name: "ruleDoesNotHaveForbiddenAnnotation", validator: doesNotHaveAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"xxx": "yyy"}}, expectedErrors: 0},
	{name: "ruleHaveForbiddenAnnotation", validator: doesNotHaveAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// hasAnyOfLabels
	{name: "ruleHasOneOfLabelExpectedLabels", validator: hasAnyOfLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveAnyOfExpectedLabels", validator: hasAnyOfLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"xxx": "yyy"}}, expectedErrors: 1},

	// hasAnyOfAnnotations
	{name: "ruleHasOneOfLabelExpectedAnnotations", validator: hasAnyOfAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveAnyOfExpectedAnnotations", validator: hasAnyOfAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"xxx": "yyy"}}, expectedErrors: 1},

	// labelMatchesRegexp
	{name: "ruleLabelMatchesRegexp", validator: labelMatchesRegexp{label: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleLabelMissingRegexValidatedLabel", validator: labelMatchesRegexp{label: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{}, expectedErrors: 0},
	{name: "ruleLabelDoesNotMatchRegexp", validator: labelMatchesRegexp{label: "foo", regexp: regexp.MustCompile("[0-9]+")}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// annotationMatchesRegexp
	{name: "ruleAnnotationMatchesRegexp", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleAnnotationMissingRegexValidatedLabel", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{}, expectedErrors: 0},
	{name: "ruleAnnotationDoesNotMatchRegexp", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile("[0-9]+")}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// labelHasAllowedValue
	{name: "ruleHasLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx"}}, expectedErrors: 1},

	// annotationHasAllowedValue
	{name: "ruleHasAnnotationWithAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveAnnotationWithAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "xxx"}}, expectedErrors: 1},

	// annotationIsValidURL
	{name: "ruleHasAnnotationWithValidURLAnnotation", validator: annotationIsValidURL{annotation: "foo"}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "https://fusakla.cz"}}, expectedErrors: 0},
	{name: "ruleHasAnnotationWithInvalidURLAnnotation", validator: annotationIsValidURL{annotation: "foo"}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// expressionDoesNotUseLabels
	{name: "ruleExprDoesNotUseLabels", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleExprUsesForbiddenLabelInSelector", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up{foo='bar'}"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInBy", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "sum(up) by (foo)"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInWithout", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "sum(up) without (foo)"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInOn", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up * on(foo) up"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInGroup", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up * group_left (foo) up"}, expectedErrors: 1},

	// expressionDoesNotUseOlderDataThan
	{name: "ruleExprDoesNotUseOlderData", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleExprUsesOldDataInRangeSelector", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}[2h]"}, expectedErrors: 1},
	{name: "ruleExprUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'} offset 2h"}, expectedErrors: 1},
	{name: "ruleExprUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "increase(delta(up{xxx='yyy'}[1m])[2h:1m])"}, expectedErrors: 1},
}

func Test(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.validator.Validate(tc.rule)
			assert.Equal(t, len(errs), tc.expectedErrors, "unexpected number of errors, expected %d but got: %s", tc.expectedErrors, errs)
		})
	}
}
