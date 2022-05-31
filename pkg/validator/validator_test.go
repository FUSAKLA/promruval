package validator

import (
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"gotest.tools/assert"
	"reflect"
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
	{name: "ruleHasExpectedLabelInExpr", validator: hasLabels{labels: []string{"foo", "foo2"}, searchInExpr: true}, rule: rulefmt.Rule{Expr: "up{foo='bar', foo2='bar2'}"}, expectedErrors: 0},
	{name: "ruleMissingExpectedLabelInExpr", validator: hasLabels{labels: []string{"foo", "foo2"}, searchInExpr: true}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 2},

	// hasAnnotations
	{name: "ruleHasExpectedAnnotation", validator: hasAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar", "foo2": "bar2"}}, expectedErrors: 0},
	{name: "ruleMissingExpectedAnnotation", validator: hasAnnotations{annotations: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Annotations: map[string]string{"xxx": "yyy"}}, expectedErrors: 2},

	// doesNotHaveLabels
	{name: "ruleDoesNotHaveForbiddenLabel", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"xxx": "yyy"}}, expectedErrors: 0},
	{name: "ruleHaveForbiddenLabel", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 1},
	{name: "ruleDoesNotHaveForbiddenLabelInExp", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}, searchInExpr: true}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleHasForbiddenLabelInExp", validator: doesNotHaveLabels{labels: []string{"foo", "foo2"}, searchInExpr: true}, rule: rulefmt.Rule{Expr: "up{foo='bar'}"}, expectedErrors: 1},

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
	{name: "ruleHasCsvLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx,bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx"}}, expectedErrors: 1},
	{name: "ruleHasCsvLabelWithoutAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx,yyy"}}, expectedErrors: 1},

	// annotationHasAllowedValue
	{name: "ruleHasAnnotationWithAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleHasCsvAnnotationWithAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "xxx,bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveAnnotationWithAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "xxx"}}, expectedErrors: 1},
	{name: "ruleHasCsvAnnotationWithoutAllowedValue", validator: annotationHasAllowedValue{annotation: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "xxx,yyy"}}, expectedErrors: 1},

	// annotationIsValidURL
	{name: "ruleHasAnnotationWithValidURLAnnotation", validator: annotationIsValidURL{annotation: "foo", resolveURL: false}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "https://fusakla.cz"}}, expectedErrors: 0},
	{name: "ruleHasAnnotationWithInvalidURLAnnotation", validator: annotationIsValidURL{annotation: "foo", resolveURL: false}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "htttpsss:////foo.bbbbar"}}, expectedErrors: 1},

	// expressionDoesNotUseLabels
	{name: "ruleExprDoesNotUseLabels", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleExprUsesForbiddenLabelInSelector", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up{foo='bar'}"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInBy", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "sum(up) by (foo)"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInWithout", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "sum(up) without (foo)"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInOn", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up * on(foo) up"}, expectedErrors: 1},
	{name: "ruleExprUsesForbiddenLabelInGroup", validator: expressionDoesNotUseLabels{labels: []string{"foo"}}, rule: rulefmt.Rule{Expr: "up * group_left (foo) up"}, expectedErrors: 1},

	// expressionDoesNotUseOlderDataThan
	{name: "ruleExprDoesNotUseOlderData", validator: expressionDoesNotUseOlderDataThan{limit: model.Duration(time.Hour)}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleExprUsesOldDataInRangeSelector", validator: expressionDoesNotUseOlderDataThan{limit: model.Duration(time.Hour)}, rule: rulefmt.Rule{Expr: "avg_over_time(up{xxx='yyy'}[2h])"}, expectedErrors: 1},
	{name: "ruleExprUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: model.Duration(time.Hour)}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'} offset 2h"}, expectedErrors: 1},
	{name: "ruleExprUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: model.Duration(time.Hour)}, rule: rulefmt.Rule{Expr: "increase(delta(up{xxx='yyy'}[1m])[2h:1m])"}, expectedErrors: 1},

	// expressionDoesNotUseRangeShorterThan
	{name: "ruleExprDoesNotUseShorterRange", validator: expressionDoesNotUseRangeShorterThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{Expr: "up[2m]"}, expectedErrors: 0},
	{name: "ruleExprUsesShorterRange", validator: expressionDoesNotUseRangeShorterThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{Expr: "up[20s]"}, expectedErrors: 1},
	{name: "ruleExprSubqueryUsesShorterRange", validator: expressionDoesNotUseRangeShorterThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{Expr: "avg_over_model.Duration(time(sum)(up)[20s:20s])"}, expectedErrors: 1},
	{name: "ruleExprSubqueryAndQueryUsesShorterRange", validator: expressionDoesNotUseRangeShorterThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{Expr: "avg_over_time(increase(up[20s])[20s:20s])"}, expectedErrors: 2},

	// annotationIsValidPromQL
	{name: "ruleHasAnnotationWithValidPromQLAnnotation", validator: annotationIsValidPromQL{annotation: "foo"}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "avg_over_time(up{foo='bar'}[2m])"}}, expectedErrors: 0},
	{name: "ruleHasAnnotationWithInvalidPromQLAnnotation", validator: annotationIsValidPromQL{annotation: "foo"}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "avg_over_time(up{foo='bar'})"}}, expectedErrors: 1},

	// validateAnnotationTemplates
	{name: "ruleHasAnnotationWithValidTemplate", validator: validateAnnotationTemplates{}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "foo {{ $value | humanizeDuration }} bar"}}, expectedErrors: 0},
	{name: "ruleHasAnnotationWithValidTemplateAndUnknownVariable", validator: validateAnnotationTemplates{}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "foo {{ .xxx }}"}}, expectedErrors: 0},
	{name: "ruleHasAnnotationWithInvalidTemplate", validator: validateAnnotationTemplates{}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "foo {{ $value | fooBar }} bar"}}, expectedErrors: 1},

	// forIsNotLongerThan
	{name: "alertHasNoFor", validator: forIsNotLongerThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{}, expectedErrors: 0},
	{name: "alertHasShorterFor", validator: forIsNotLongerThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{For: model.Duration(time.Second)}, expectedErrors: 0},
	{name: "alertHasLongerFor", validator: forIsNotLongerThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{For: model.Duration(time.Hour)}, expectedErrors: 1},

	// expressionDoesNotUseIrate
	{name: "expressionDoesNotUseIrate", validator: expressionDoesNotUseIrate{}, rule: rulefmt.Rule{Expr: "rate(foo_bar[1m])"}, expectedErrors: 0},
	{name: "expressionUsesIrate", validator: expressionDoesNotUseIrate{}, rule: rulefmt.Rule{Expr: "irate(foo_bar[1m])"}, expectedErrors: 1},

	// validFunctionsOnCounters
	{name: "rateOnCounter", validator: validFunctionsOnCounters{}, rule: rulefmt.Rule{Expr: "rate(foo_bar_total[1m])"}, expectedErrors: 0},
	{name: "rateOnNonCounter", validator: validFunctionsOnCounters{}, rule: rulefmt.Rule{Expr: "rate(foo_bar[1m])"}, expectedErrors: 1},
	{name: "increaseOnNonCounter", validator: validFunctionsOnCounters{}, rule: rulefmt.Rule{Expr: "increase ( foo_bar [1m] )"}, expectedErrors: 1},

	// rateBeforeAggregation
	{name: "SumAfterRate", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "sum(rate(foo_bar_total[1m]))"}, expectedErrors: 0},
	{name: "SumBeforeRate", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "rate(sum(foo_bar_total)[5m:])"}, expectedErrors: 1},
	{name: "minBeforeIncrease", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "increase(min(foo_bar_total)[5m:])"}, expectedErrors: 1},

	// nonEmptyLabels
	{name: "NonEmptyLabel", validator: nonEmptyLabels{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "EmptyLabel", validator: nonEmptyLabels{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": ""}}, expectedErrors: 1},
	{name: "OneEmptyOneNoneEmpty", validator: nonEmptyLabels{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": ""}}, expectedErrors: 1},
	{name: "BothEmpty", validator: nonEmptyLabels{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "", "bar": ""}}, expectedErrors: 2},

	// newExclusiveLabels
	{name: "MissingLabel1", validator: exclusiveLabels{label1: "foo", label2: "bar"}, rule: rulefmt.Rule{Labels: map[string]string{"bar": "yyy"}}, expectedErrors: 0},
	{name: "label1PresentDifferentValue", validator: exclusiveLabels{label1: "foo", label1Value: "ooo", label2: "bar"}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": "yyy"}}, expectedErrors: 0},
	{name: "label1PresentLabel2Present", validator: exclusiveLabels{label1: "foo", label2: "bar"}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": "yyy"}}, expectedErrors: 1},
	{name: "label1PresentMatchingValueLabel2Present", validator: exclusiveLabels{label1: "foo", label1Value: "xxx", label2: "bar"}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": "yyy"}}, expectedErrors: 1},
	{name: "label1PresentMatchingValueLabel2PresentDifferentValue", validator: exclusiveLabels{label1: "foo", label1Value: "xxx", label2: "bar", label2Value: "ooo"}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": "yyy"}}, expectedErrors: 0},
	{name: "label1PresentMatchingValueLabel2PresentMatchingValue", validator: exclusiveLabels{label1: "foo", label1Value: "xxx", label2: "bar", label2Value: "yyy"}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx", "bar": "yyy"}}, expectedErrors: 1},
}

func Test(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%s", reflect.TypeOf(tc.validator), tc.name), func(t *testing.T) {
			errs := tc.validator.Validate(tc.rule)
			assert.Equal(t, len(errs), tc.expectedErrors, "unexpected number of errors, expected %d but got: %s", tc.expectedErrors, errs)
		})
	}
}
