package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gotest.tools/assert"
)

func mustCompileAnchoredRegexp(regexpString string) *regexp.Regexp {
	compiled, err := compileAnchoredRegexp(regexpString)
	if err != nil {
		panic(err)
	}
	return compiled
}

var testCases = []struct {
	name           string
	validator      Validator
	group          unmarshaler.RuleGroup
	rule           rulefmt.Rule
	promClient     *prometheus.Client
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
	{name: "ruleLabelDoesNotMatchRegexp", validator: labelMatchesRegexp{label: "foo", regexp: regexp.MustCompile(`\d+`)}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// annotationMatchesRegexp
	{name: "ruleAnnotationMatchesRegexp", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleAnnotationMissingRegexValidatedLabel", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile(".*")}, rule: rulefmt.Rule{}, expectedErrors: 0},
	{name: "ruleAnnotationDoesNotMatchRegexp", validator: annotationMatchesRegexp{annotation: "foo", regexp: regexp.MustCompile(`\d+`)}, rule: rulefmt.Rule{Annotations: map[string]string{"foo": "bar"}}, expectedErrors: 1},

	// labelHasAllowedValue
	{name: "ruleHasLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "ruleHasCsvLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx,bar"}}, expectedErrors: 0},
	{name: "ruleDoesNotHaveLabelWithAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx"}}, expectedErrors: 1},
	{name: "ruleHasCsvLabelWithoutAllowedValue", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, commaSeparatedValue: true}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "xxx,yyy"}}, expectedErrors: 1},
	{name: "ruleHasTemplatedLabelAndCannotHave", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, ignoreTemplatedValues: false}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "{{ .Labels.foo }}"}}, expectedErrors: 1},
	{name: "ruleHasTemplatedLabelAndCannotHave", validator: labelHasAllowedValue{label: "foo", allowedValues: []string{"bar"}, ignoreTemplatedValues: true}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "{{ .Labels.foo }}"}}, expectedErrors: 0},

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

	// expressionUsesOnlyAllowedLabelsForMetricRegexp
	{name: "ruleExprDoesNotUseAnyLabels", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{"label_app"}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels"}, expectedErrors: 0},
	{name: "ruleExprDoesUseForbiddenLabelInSelector", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels{app=~'foo'}"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInSelector", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels{app='foo'}"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInSelectorWithMetricNameAsRegexp", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_.*_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels{app='foo'}"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInSelectorWithMetricNameAsRegexpExtraAnchors", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("^kube_.*_labels$")}, rule: rulefmt.Rule{Expr: "kube_pod_labels{app='foo'}"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInSelectorWithMetricNameAsRegexpNotFullMatch", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_.*_label")}, rule: rulefmt.Rule{Expr: "kube_pod_labels{app='foo'}"}, expectedErrors: 0},
	{name: "ruleExprDoesUseForbiddenLabelInSelectorWithMetricNameAsRegexp", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "{__name__=~'kube_.*_labels', app='foo'}"}, expectedErrors: 0},
	{name: "ruleExprDoesUseForbiddenLabelInBy", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "sum(kube_pod_labels) by (app)"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInOn", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels * on(app) up"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInGroup", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "group(kube_pod_labels) by (label_app)"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInBinaryExprWithLabelTransfer1", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels * on(app) group_left(foo) up"}, expectedErrors: 1},
	{name: "ruleExprDoesUseForbiddenLabelInBinaryExprWithLabelTransfer2", validator: expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabelsMap([]string{}), metricNameRegexp: mustCompileAnchoredRegexp("kube_pod_labels")}, rule: rulefmt.Rule{Expr: "kube_pod_labels * on(app) group_right(foo) up"}, expectedErrors: 2},

	// expressionDoesNotUseOlderDataThan
	{name: "ruleExprDoesNotUseOlderData", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'}"}, expectedErrors: 0},
	{name: "ruleExprUsesOldDataInRangeSelector", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "last_over_time(up{xxx='yyy'}[2h])"}, expectedErrors: 1},
	{name: "ruleExprUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'} offset 2h"}, expectedErrors: 1},
	{name: "ruleExprSubqueryUsesOldDataInRangeOffset", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "increase(delta(up{xxx='yyy'}[1m])[2h:1m])"}, expectedErrors: 1},
	{name: "ruleExprAtZero", validator: expressionDoesNotUseOlderDataThan{limit: time.Hour}, rule: rulefmt.Rule{Expr: "up{xxx='yyy'} @0"}, expectedErrors: 1},

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
	{name: "increaseOnCounter", validator: validFunctionsOnCounters{}, rule: rulefmt.Rule{Expr: `increase(foo_bar_total{namespace="foo"}[1m])`}, expectedErrors: 0},
	{name: "increaseOnNonCounter", validator: validFunctionsOnCounters{}, rule: rulefmt.Rule{Expr: `increase(foo_bar{namespace="foo"}[1m])`}, expectedErrors: 1},
	{name: "increaseOnHistogram", validator: validFunctionsOnCounters{allowHistograms: true}, rule: rulefmt.Rule{Expr: `increase(foo_bar_count{namespace="foo"}[1m])`}, expectedErrors: 0},
	{name: "increaseOnHistogramNotAllowed", validator: validFunctionsOnCounters{allowHistograms: false}, rule: rulefmt.Rule{Expr: `increase(foo_bar_count{namespace="foo"}[1m])`}, expectedErrors: 1},

	// rateBeforeAggregation
	{name: "IncreaseWithoutAggregation", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: `increase(foo_bar{label="value"}[1m]) * count(foo_bar{label="value"})`}, expectedErrors: 0},
	{name: "IncreaseWithAggregation", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: `increase(count(foo_bar{label="value"})[1m:]) * count(foo_bar{label="value"})`}, expectedErrors: 1},
	{name: "SumAfterRate", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "sum(rate(foo_bar_total[1m]))"}, expectedErrors: 0},
	{name: "SumBeforeRate", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "rate(sum(foo_bar_total)[1m])"}, expectedErrors: 1},
	{name: "MinBeforeIncrease", validator: rateBeforeAggregation{}, rule: rulefmt.Rule{Expr: "increase(min(foo_bar_total)[1m])"}, expectedErrors: 1},

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

	// expressionCanBeEvaluated
	{name: "evaluationOk", validator: expressionCanBeEvaluated{evaluationDurationLimit: time.Second, timeSeriesLimit: 10}, promClient: prometheus.NewClientMock(prometheus.NewQueryVectorResponseMock(2), 0, false, false), rule: rulefmt.Rule{Expr: "1"}, expectedErrors: 0},
	{name: "evaluationWithWarning", validator: expressionCanBeEvaluated{}, promClient: prometheus.NewClientMock(prometheus.NewQueryVectorResponseMock(2), 0, true, false), rule: rulefmt.Rule{Expr: "1"}, expectedErrors: 0},
	{name: "evaluationWithError", validator: expressionCanBeEvaluated{}, promClient: prometheus.NewClientMock(prometheus.NewQueryVectorResponseMock(2), 0, false, true), rule: rulefmt.Rule{Expr: "1"}, expectedErrors: 1},
	{name: "evaluationTooSlow", validator: expressionCanBeEvaluated{evaluationDurationLimit: time.Second}, promClient: prometheus.NewClientMock(prometheus.NewQueryVectorResponseMock(2), time.Second*2, false, false), rule: rulefmt.Rule{Expr: "1"}, expectedErrors: 1},
	{name: "evaluationTooManySeries", validator: expressionCanBeEvaluated{timeSeriesLimit: 1}, promClient: prometheus.NewClientMock(prometheus.NewQueryVectorResponseMock(2), 0, false, false), rule: rulefmt.Rule{Expr: "1"}, expectedErrors: 1},

	// expressionSelectorsMatchesAnything
	{name: "matches", validator: expressionSelectorsMatchesAnything{}, promClient: prometheus.NewClientMock(prometheus.NewSeriesResponseMock(2), 0, false, false), rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "noMatches", validator: expressionSelectorsMatchesAnything{}, promClient: prometheus.NewClientMock(prometheus.NewSeriesResponseMock(0), 0, false, false), rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},
	{name: "queryError", validator: expressionSelectorsMatchesAnything{}, promClient: prometheus.NewClientMock(prometheus.NewSeriesResponseMock(2), 0, false, true), rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},

	// expressionUsesExistingLabels
	{name: "labelsExists", validator: expressionUsesExistingLabels{}, promClient: prometheus.NewClientMock([]string{"__name__", "foo"}, 0, false, false), rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "labelsDoesNotExist", validator: expressionUsesExistingLabels{}, promClient: prometheus.NewClientMock([]string{"__name__"}, 0, false, false), rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},

	{name: "withName", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "withNameInLabel", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `{__name__="up", foo="bar"}`}, expectedErrors: 0},
	{name: "noName", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `{foo="bar"}`}, expectedErrors: 1},
	{name: "complexExpressionsWithName", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `(
	 sum(rate(http_requests_total{code=~"5..", job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	/
	 sum(rate(http_requests_total{job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	) * 100 > 10
	and
	sum(rate(http_requests_total{job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler) > 2`}, expectedErrors: 0},
	{name: "complexExpressionsNoName", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `(
	  sum(rate(http_requests_total{code=~"5..", job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	/
	  sum(rate( {job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	) * 100 > 10
	and
	sum(rate(http_requests_total{job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler) > 2`}, expectedErrors: 1},
	{name: "complexExpressionsMultipleNoName", validator: expressionWithNoMetricName{}, promClient: nil, rule: rulefmt.Rule{Expr: `(
	   sum(rate(http_requests_total{code=~"5..", job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	 /
	   sum(rate( {job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	 ) * 100 > 10
	 and
	 sum(rate( {job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler) > 2`}, expectedErrors: 2},

	// expressionDoesNotUseMetrics
	{name: "emptyList", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "doesNotUseForbiddenMetric", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{regexp.MustCompile(`foo_bar`)}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "usesForbiddenMetric", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{regexp.MustCompile(`foo_bar`)}}, rule: rulefmt.Rule{Expr: `foo_bar{foo="bar"}`}, expectedErrors: 1},
	{name: "usesTwoOfThreeForbiddenMetrics", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{regexp.MustCompile(`foo_bar`), regexp.MustCompile(`foo_baz`), regexp.MustCompile(`^foo$`)}}, rule: rulefmt.Rule{Expr: `foo_baz{foo="bar"} and foo_bar`}, expectedErrors: 2},
	{name: "usesMetricMatchingRegexp", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{regexp.MustCompile(`foo_bar.*`)}}, rule: rulefmt.Rule{Expr: `foo_baz_baz{foo="bar"} and foo_bar`}, expectedErrors: 1},
	{name: "regexpIsFullyAnchored", validator: expressionDoesNotUseMetrics{metricNameRegexps: []*regexp.Regexp{regexp.MustCompile(`^foo_bar$`)}}, rule: rulefmt.Rule{Expr: `foo_baz_baz{foo="bar"} and foo_bar`}, expectedErrors: 1},

	// hasSourceTenantsForMetrics
	{name: "emptyMapping", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "usesMetricWithSourceTenantAndGroupHasSourceTenant", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1"}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 0},
	{name: "usesMetricWithSourceTenantAndGroupDoesNotHaveSourceTenant", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant2"}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 1},
	{name: "usesMetricWithSourceTenantAndGroupHasMultipleSourceTenants", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant2", "tenant1"}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 0},
	{name: "usesMetricWithSourceTenantAndGroupHasMultipleSourceTenantsAndOneIsMissing", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant2", "tenant3"}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 1},
	{name: "doesNotHaveSourceTenantForMetricButIsDefault", validator: hasSourceTenantsForMetrics{defaultTenant: "tenant1", sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 0},
	{name: "doesNotHaveSourceTenantForMetricAndIsNotDefault", validator: hasSourceTenantsForMetrics{defaultTenant: "tenant2", sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{}}, rule: rulefmt.Rule{Expr: `teanant1_metric{foo="bar"}`}, expectedErrors: 1},
	{name: "notMatchingNegativeRegexp", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric_.*$`), negativeRegexp: regexp.MustCompile(`^teanant1_metric_bar$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1"}}, rule: rulefmt.Rule{Expr: `teanant1_metric_foo{foo="bar"}`}, expectedErrors: 0},
	{name: "MatchingNegativeRegexp", validator: hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{"tenant1": {{regexp: regexp.MustCompile(`^teanant1_metric_.*$`), negativeRegexp: regexp.MustCompile(`^teanant1_metric_bar$`)}}}}, group: unmarshaler.RuleGroup{SourceTenants: []string{}}, rule: rulefmt.Rule{Expr: `teanant1_metric_bar{foo="bar"}`}, expectedErrors: 0},

	// hasAllowedSourceTenants
	{name: "emptyAllowedSourceTenantsAndGroupSourceTenants", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{}}, group: unmarshaler.RuleGroup{SourceTenants: []string{}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "emptyAllowedSourceTenantsAndGroupSourceTenantsWithOneTenant", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1"}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},
	{name: "emptyAllowedSourceTenantsAndGroupSourceTenantsWithMultipleTenants", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1", "tenant2"}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},
	{name: "allowedSourceTenantsAndGroupSourceTenantsWithOneTenant", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{"tenant1"}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1"}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "allowedSourceTenantsAndGroupSourceTenantsWithMultipleTenants", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{"tenant1", "tenant2"}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1", "tenant2"}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "allowedSourceTenantsAndGroupSourceTenantsWithMultipleTenantsAndOneIsMissing", validator: hasAllowedSourceTenants{allowedSourceTenants: []string{"tenant1", "tenant2"}}, group: unmarshaler.RuleGroup{SourceTenants: []string{"tenant1", "tenant3"}}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 1},

	// hasAllowedEvaluationInterval
	{name: "validInterval", validator: hasAllowedEvaluationInterval{minimum: model.Duration(time.Second), maximum: model.Duration(time.Minute), mustBeSet: true}, group: unmarshaler.RuleGroup{Interval: model.Duration(time.Minute)}, expectedErrors: 0},
	{name: "unsetRequiredInterval", validator: hasAllowedEvaluationInterval{minimum: model.Duration(time.Second), maximum: model.Duration(time.Minute), mustBeSet: true}, group: unmarshaler.RuleGroup{Interval: 0}, expectedErrors: 1},
	{name: "unsetNotRequiredInterval", validator: hasAllowedEvaluationInterval{minimum: model.Duration(time.Second), maximum: model.Duration(time.Minute), mustBeSet: false}, group: unmarshaler.RuleGroup{Interval: 0}, expectedErrors: 0},
	{name: "tooShortInterval", validator: hasAllowedEvaluationInterval{minimum: model.Duration(time.Minute), maximum: model.Duration(time.Hour), mustBeSet: true}, group: unmarshaler.RuleGroup{Interval: model.Duration(time.Second)}, expectedErrors: 1},
	{name: "tooHighInterval", validator: hasAllowedEvaluationInterval{minimum: model.Duration(time.Minute), maximum: model.Duration(time.Hour), mustBeSet: true}, group: unmarshaler.RuleGroup{Interval: model.Duration(time.Hour * 2)}, expectedErrors: 1},

	// hasValidPartialResponseStrategy
	{name: "validPartialResponseStrategy", validator: hasValidPartialResponseStrategy{}, group: unmarshaler.RuleGroup{PartialResponseStrategy: "warn"}, expectedErrors: 0},
	{name: "validPartialResponseStrategy", validator: hasValidPartialResponseStrategy{}, group: unmarshaler.RuleGroup{PartialResponseStrategy: "abort"}, expectedErrors: 0},
	{name: "invalidPartialResponseStrategy", validator: hasValidPartialResponseStrategy{}, group: unmarshaler.RuleGroup{PartialResponseStrategy: "foo"}, expectedErrors: 1},
	{name: "unsetPartialResponseStrategyAllowed", validator: hasValidPartialResponseStrategy{mustBeSet: false}, group: unmarshaler.RuleGroup{}, expectedErrors: 0},
	{name: "unsetPartialResponseStrategyDisallowed", validator: hasValidPartialResponseStrategy{mustBeSet: true}, group: unmarshaler.RuleGroup{}, expectedErrors: 1},

	// expressionIsWellFormatted
	{name: "validExpression", validator: expressionIsWellFormatted{showFormatted: true}, rule: rulefmt.Rule{Expr: `up{foo="bar"}`}, expectedErrors: 0},
	{name: "invalidExpression", validator: expressionIsWellFormatted{showFormatted: true}, rule: rulefmt.Rule{Expr: `up     == 1`}, expectedErrors: 1},
	{name: "validWithCommentThatShouldBeIgnored", validator: expressionIsWellFormatted{showFormatted: true}, rule: rulefmt.Rule{Expr: `up == 1 # fooo`}, expectedErrors: 0},
	{name: "invalidButWithCommentAndShouldBeSkipped", validator: expressionIsWellFormatted{showFormatted: true, skipExpressionsWithComments: true}, rule: rulefmt.Rule{Expr: `up           == 1 # fooo`}, expectedErrors: 0},

	// maxRulesPerGroup
	{name: "allowedNumberOfGroups", validator: maxRulesPerGroup{limit: 2}, group: unmarshaler.RuleGroup{Rules: []unmarshaler.RuleWithComment{{}, {}}}, expectedErrors: 0},
	{name: "tooManyRules", validator: maxRulesPerGroup{limit: 1}, group: unmarshaler.RuleGroup{Rules: []unmarshaler.RuleWithComment{{}, {}}}, expectedErrors: 1},

	// hasAllowedLimit
	{name: "limitOK", validator: hasAllowedLimit{limit: 2, mustBeSet: false}, group: unmarshaler.RuleGroup{Limit: 1}, expectedErrors: 0},
	{name: "limitNotSet", validator: hasAllowedLimit{limit: 2, mustBeSet: false}, group: unmarshaler.RuleGroup{Limit: 0}, expectedErrors: 1},
	{name: "limitSetButHigh", validator: hasAllowedLimit{limit: 2, mustBeSet: false}, group: unmarshaler.RuleGroup{Limit: 5}, expectedErrors: 1},

	// validateLabelTemplates
	{name: "noTemplate", validator: validateLabelTemplates{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "bar"}}, expectedErrors: 0},
	{name: "validLabelTemplate", validator: validateLabelTemplates{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "foo {{ $value | humanizeDuration }} bar"}}, expectedErrors: 0},
	{name: "invalidLabelTemplate", validator: validateLabelTemplates{}, rule: rulefmt.Rule{Labels: map[string]string{"foo": "foo {{ $value | huuuuumanizeDuration }} bar"}}, expectedErrors: 1},

	// keepFiringForIsNotLongerThan
	{name: "keepFiringForIsNotLongerThanOK", validator: keepFiringForIsNotLongerThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{KeepFiringFor: model.Duration(time.Second)}, expectedErrors: 0},
	{name: "keepFiringForIsNotLongerThanWrong", validator: keepFiringForIsNotLongerThan{limit: model.Duration(time.Minute)}, rule: rulefmt.Rule{KeepFiringFor: model.Duration(time.Minute * 2)}, expectedErrors: 1},

	// expressionIsValidPromQL
	{name: "expressionIsValidPromQL_OK", validator: expressionIsValidPromQL{}, rule: rulefmt.Rule{Expr: "sum(rate(foo{bar='baz'}[1m]))"}, expectedErrors: 0},
	{name: "expressionIsValidPromQL_Invalid", validator: expressionIsValidPromQL{}, rule: rulefmt.Rule{Expr: "sum(rate(foo{bar='baz'} | ??? [1m]))"}, expectedErrors: 1},

	// expressionIsValidLogQL
	{name: "expressionIsValidLogQL_OK", validator: expressionIsValidLogQL{}, rule: rulefmt.Rule{Expr: `sum(rate({job="foo"} |= "foo"[1m]))`}, expectedErrors: 0},
	{name: "expressionIsValidLogQL_Invalid", validator: expressionIsValidLogQL{}, rule: rulefmt.Rule{Expr: "increase(foo_bar{foo='bar'}[5m])"}, expectedErrors: 1},

	// logQlExpressionUsesRangeAggregation
	{name: "logQlExpressionUsesRangeAggregation_OK", validator: logQLExpressionUsesRangeAggregation{}, rule: rulefmt.Rule{Expr: `sum(rate({job="foo"} |= "foo"[1m]))`}, expectedErrors: 0},
	{name: "logQlExpressionUsesRangeAggregation_Invalid", validator: logQLExpressionUsesRangeAggregation{}, rule: rulefmt.Rule{Expr: `{job="foo"} |= "foo"`}, expectedErrors: 1},

	// logQlExpressionUsesFiltersFirst
	{name: "logQlExpressionUsesFiltersFirst_OK", validator: logQlExpressionUsesFiltersFirst{}, rule: rulefmt.Rule{Expr: `{job="foo"} |= "foo" | logfmt`}, expectedErrors: 0},
	{name: "logQlExpressionUsesFiltersFirst_Invalid", validator: logQlExpressionUsesFiltersFirst{}, rule: rulefmt.Rule{Expr: `{job="foo"} | logfmt |= "foo"`}, expectedErrors: 1},
	{name: "logQlExpressionUsesFiltersFirst_Invalid", validator: logQlExpressionUsesFiltersFirst{}, rule: rulefmt.Rule{Expr: `{job="foo"} |= "foo" | logfmt |= "bar"`}, expectedErrors: 1},

	// alertNameMatchesRegexp
	{name: "alertNameMatchesRegexp_Valid", validator: alertNameMatchesRegexp{pattern: regexp.MustCompile("Foo.*")}, rule: rulefmt.Rule{Alert: `FooBAr`}, expectedErrors: 0},
	{name: "alertNameMatchesRegexp_NotMatch", validator: alertNameMatchesRegexp{pattern: regexp.MustCompile("Foo.*")}, rule: rulefmt.Rule{Alert: `Bar`}, expectedErrors: 1},

	// recordedMetricNameMatchesRegexp
	{name: "recordedMetricNameMatchesRegexp_Matches", validator: recordedMetricNameMatchesRegexp{pattern: regexp.MustCompile("[^:]+:[^:]+:[^:]+")}, rule: rulefmt.Rule{Record: `cluster:foo_bar:avg`}, expectedErrors: 0},
	{name: "recordedMetricNameMatchesRegexp_notMatches", validator: recordedMetricNameMatchesRegexp{pattern: regexp.MustCompile("[^:]+:[^:]+:[^:]+")}, rule: rulefmt.Rule{Record: `foo_bar`}, expectedErrors: 1},

	// recordedMetricNameDoesNotMatchRegexp
	{name: "recordedMetricNameDoesNotMatchRegexp_Matches", validator: recordedMetricNameDoesNotMatchRegexp{pattern: regexp.MustCompile("^foo_bar$")}, rule: rulefmt.Rule{Record: `cluster:foo_bar:avg`}, expectedErrors: 0},
	{name: "recordedMetricNameDoesNotMatchRegexp_notMatches", validator: recordedMetricNameDoesNotMatchRegexp{pattern: regexp.MustCompile("^foo_bar$")}, rule: rulefmt.Rule{Record: `foo_bar`}, expectedErrors: 1},

	// hasAllowedQueryOffset
	{name: "hasAllowedQueryOffset_valid", validator: hasAllowedQueryOffset{min: model.Duration(time.Second), max: model.Duration(time.Minute)}, group: unmarshaler.RuleGroup{QueryOffset: model.Duration(time.Second * 30)}, expectedErrors: 0},
	{name: "hasAllowedQueryOffset_tooHigh", validator: hasAllowedQueryOffset{min: model.Duration(time.Second), max: model.Duration(time.Minute)}, group: unmarshaler.RuleGroup{QueryOffset: model.Duration(time.Minute * 2)}, expectedErrors: 1},
	{name: "hasAllowedQueryOffset_tooLow", validator: hasAllowedQueryOffset{min: model.Duration(time.Minute), max: model.Duration(time.Hour)}, group: unmarshaler.RuleGroup{QueryOffset: model.Duration(time.Second)}, expectedErrors: 1},

	// groupNameMatchesRegexp
	{name: "groupNameMatchesRegexp_valid", validator: groupNameMatchesRegexp{pattern: regexp.MustCompile(`^[A-Z]\S+$`)}, group: unmarshaler.RuleGroup{Name: "TestGroup"}, expectedErrors: 0},
	{name: "groupNameMatchesRegexp_invalid", validator: groupNameMatchesRegexp{pattern: regexp.MustCompile(`^[A-Z]\S+$`)}, group: unmarshaler.RuleGroup{Name: "Test Group"}, expectedErrors: 1},

	// expressionDoesNotUseExperimentalFunctions
	{name: "expressionDoesNotUseExperimentalFunctions_valid", validator: expressionDoesNotUseExperimentalFunctions{}, rule: rulefmt.Rule{Expr: `sort_desc(up)`}, expectedErrors: 0},
	{name: "expressionDoesNotUseExperimentalFunctions_invalid", validator: expressionDoesNotUseExperimentalFunctions{}, rule: rulefmt.Rule{Expr: `sort_by_label(up, "instance")`}, expectedErrors: 1},

	// expressionUsesUnderscoresInLargeNumbers
	{name: "expressionUsesUnderscoresInLargeNumbers_valid", validator: expressionUsesUnderscoresInLargeNumbers{}, rule: rulefmt.Rule{Expr: `vector(time())  > 100`}, expectedErrors: 0},
	{name: "expressionUsesUnderscoresInLargeNumbers_valid_duration", validator: expressionUsesUnderscoresInLargeNumbers{}, rule: rulefmt.Rule{Expr: `vector(time())  > 100h`}, expectedErrors: 0},
	{name: "expressionUsesUnderscoresInLargeNumbers_valid_exp", validator: expressionUsesUnderscoresInLargeNumbers{}, rule: rulefmt.Rule{Expr: `vector(time())  > 10e12`}, expectedErrors: 0},
	{name: "expressionUsesUnderscoresInLargeNumbers_invalid", validator: expressionUsesUnderscoresInLargeNumbers{}, rule: rulefmt.Rule{Expr: `time() > 1000`}, expectedErrors: 1},

	// expressionDoesNotUseOperationsBetweenClassicHistogramBuckets
	{name: "expressionDoesNotUseOperationsBetweenClassicHistogramBuckets_valid", validator: expressionDoesNotUseOperationsBetweenClassicHistogramBuckets{}, rule: rulefmt.Rule{Expr: `foo_bucket{le="+Inf"} - bar_bucket{le="1"}`}, expectedErrors: 0},
	{name: "expressionDoesNotUseOperationsBetweenClassicHistogramBuckets_invalid", validator: expressionDoesNotUseOperationsBetweenClassicHistogramBuckets{}, rule: rulefmt.Rule{Expr: `request_duration_seconds_bucket{le="+Inf"} - ignoring(le) request_duration_seconds_bucket{le="1"}`}, expectedErrors: 1},
	{name: "expressionDoesNotUseOperationsBetweenClassicHistogramBuckets_complicated_valid", validator: expressionDoesNotUseOperationsBetweenClassicHistogramBuckets{}, rule: rulefmt.Rule{Expr: `(request_duration_seconds_bucket{app="foo", le="+Inf"} * up{app="foo"}) - ignoring(le) request_duration_seconds_bucket{le="1"}`}, expectedErrors: 0},

	// doesNotUseEmoji
	{name: "doesNotUseEmoji_valid", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Expr: `foo_bar{emoji="foo"}`}, expectedErrors: 0},
	{name: "doesNotUseEmoji_expr_label_value_poop", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Expr: `foo_bar{emoji="💩"}`}, expectedErrors: 1},
	{name: "doesNotUseEmoji_expr_label_name_poop", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Expr: `foo_bar{"💩"="foo"}`}, expectedErrors: 1},
	{name: "doesNotUseEmoji_rule_label_value_poop", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Expr: "1", Labels: map[string]string{"foo": "💩"}}, expectedErrors: 1},
	{name: "doesNotUseEmoji_rule_label_name_poop", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Expr: "1", Labels: map[string]string{"💩": "foo"}}, expectedErrors: 1},
	{name: "doesNotUseEmoji_rule_record_poop", validator: doesNotUseEmoji{}, rule: rulefmt.Rule{Record: "foo:💩:bar", Expr: "1"}, expectedErrors: 1},

	// doesNotUseUTF8
	{name: "doesNotUseUTF8_valid", validator: doesNotUseUTF8{}, rule: rulefmt.Rule{Expr: `foo_bar{foo="bar"}`}, expectedErrors: 0},
	{name: "doesNotUseUTF8_invalid_label_name", validator: doesNotUseUTF8{}, rule: rulefmt.Rule{Expr: `foo_bar{"žluťoulinký"="bar"}`}, expectedErrors: 1},
	{name: "doesNotUseUTF8_invalid_metric_name", validator: doesNotUseUTF8{}, rule: rulefmt.Rule{Expr: `{"žluťoulinký",foo="bar"}`}, expectedErrors: 1},
	{name: "doesNotUseUTF8_invalid_rule_label_name", validator: doesNotUseUTF8{}, rule: rulefmt.Rule{Expr: `1`, Labels: map[string]string{"žluťoulinký": "foo"}}, expectedErrors: 1},
	{name: "doesNotUseUTF8_invalid_recorded_metric_name", validator: doesNotUseUTF8{}, rule: rulefmt.Rule{Expr: `1`, Record: "foo:žluťoulinký:bar"}, expectedErrors: 1},
}

func Test(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%s", reflect.TypeOf(tc.validator), tc.name), func(t *testing.T) {
			errs := tc.validator.Validate(tc.group, tc.rule, tc.promClient)
			assert.Equal(t, len(errs), tc.expectedErrors, "unexpected number of errors, expected %d but got: %s", tc.expectedErrors, errs)
		})
	}
}
