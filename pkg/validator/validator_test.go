package validator

import (
	"fmt"
	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gotest.tools/assert"
	"reflect"
	"regexp"
	"testing"
	"time"
)

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
	{name: "complexExpressionsNoName", validator: expressionWithNoMetricName{}, promClient: prometheus.NewClientMock(prometheus.NewSeriesResponseMock(2), 0, false, false), rule: rulefmt.Rule{Expr: `(
	  sum(rate(http_requests_total{code=~"5..", job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	/
	  sum(rate( {job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler)
	) * 100 > 10
	and
	sum(rate(http_requests_total{job=~"thanos-query",handler!="exemplars"}[5m])) by (role,handler) > 2`}, expectedErrors: 1},
	{name: "complexExpressionsMultipleNoName", validator: expressionWithNoMetricName{}, promClient: prometheus.NewClientMock(prometheus.NewSeriesResponseMock(2), 0, false, false), rule: rulefmt.Rule{Expr: `(
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
}

func Test(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%s", reflect.TypeOf(tc.validator), tc.name), func(t *testing.T) {
			errs := tc.validator.Validate(tc.group, tc.rule, tc.promClient)
			assert.Equal(t, len(errs), tc.expectedErrors, "unexpected number of errors, expected %d but got: %s", tc.expectedErrors, errs)
		})
	}
}
