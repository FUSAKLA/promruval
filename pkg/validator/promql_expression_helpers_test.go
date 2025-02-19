package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/assert"
)

func TestGetExpressionUsedLabels(t *testing.T) {
	tests := []struct {
		expr        string
		expected    []string
		expectedErr error
	}{
		{expr: "up{foo='bar'}", expected: []string{"foo", "__name__"}},
		{expr: "sum(up) by (foo)", expected: []string{"foo", "__name__"}},
		{expr: "up * on(foo) up", expected: []string{"foo", "__name__"}},
		{expr: "up{foo='bar'} + up{bar='baz'}", expected: []string{"foo", "bar", "__name__"}},
		{expr: "avg_over_time(up{foo='bar'}[1h])", expected: []string{"foo", "__name__"}},
		{expr: "up{foo=~'bar.*'}", expected: []string{"foo", "__name__"}},
		{expr: "up{foo!~'bar.*'}", expected: []string{"foo", "__name__"}},
		{expr: "up{foo='bar'} offset 1h", expected: []string{"foo", "__name__"}},
	}

	for _, test := range tests {
		l, err := getExpressionUsedLabels(test.expr)
		assert.ElementsMatch(t, l, test.expected, "Expected labels %v, but got %v", test.expected, l)
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}

func TestGetExpressionUsedLabelsForMetric(t *testing.T) {
	tests := []struct {
		expr        string
		metric      string
		expected    []string
		expectedErr error
	}{
		{expr: "up{bar='foo'}", metric: "kube_pod_labels", expected: []string{}},
		{expr: "kube_pod_labels{label_app='foo'}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app"}},
		{expr: "count(kube_pod_labels{label_app='foo'}) by (label_team)", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "label_team"}},
		{expr: "kube_pod_labels{label_app!='foo'}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app"}},
		{expr: "kube_pod_labels{label_app='foo'} * on(pod) kube_pod_info{}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "pod"}},
		{expr: "kube_pod_info{} * on(pod) group_left(label_workload) kube_pod_labels{label_app='foo'}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "label_workload", "pod"}},
		{expr: "kube_pod_info{} * on(pod) group_right(pod_ip) kube_pod_labels{label_app='foo'}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "pod"}},
		{expr: "kube_pod_info{} * on(pod) group_right(pod_ip) kube_pod_labels{label_app='foo'} offset 1h", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "pod"}},
		{expr: "sum(kube_pod_info * kube_pod_labels) by (foo)", metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo"}},
		{expr: "label_replace(kube_pod_labels, 'bar', '$1', 'foo', '.*')", metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo"}},
		{expr: `sum(label_join(kube_pod_labels, "foo", ",", "l1", "l2", "l3")) by (foo)`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "l1", "l2", "l3"}},
		{expr: `sum(label_join(kube_pod_labels, "foo", ",", "l1", "l2", "l3")) by (bar)`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "l1", "l2", "l3", "bar"}},
		{expr: `sum(kube_pod_labels * on (foo, bar) group_right(baz) kube_pod_info) by (to_be_dropped)`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo", "bar", "baz"}},
		{expr: `sum(kube_pod_labels * on (foo, bar) group_left(baz) kube_pod_info) by (to_be_dropped)`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo", "bar"}},
		{expr: `sum(kube_pod_labels * ignoring (foo, bar) group_left() kube_pod_info) by (to_be_dropped)`, metric: "kube_pod_labels", expected: []string{metricNameLabel}},
		{expr: `sum(kube_pod_labels * on (foo, bar) kube_pod_info) by (baz)`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo", "bar", "baz"}},
		{expr: `sort_by_label(kube_pod_labels, "foo", "bar", "baz")`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo", "bar", "baz"}},
		{expr: `sort_by_label_desc(kube_pod_labels, "foo", "bar", "baz")`, metric: "kube_pod_labels", expected: []string{metricNameLabel, "foo", "bar", "baz"}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_case_%d", i), func(t *testing.T) {
			l, err := getExpressionUsedLabelsForMetric(test.expr, regexp.MustCompile(test.metric))
			assert.ElementsMatch(t, l, test.expected, "Expected labels %v, but got %v", test.expected, l)
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
			}
		})
	}
}

func TestGetLabelMatchersForMetricRegexp(t *testing.T) {
	MustNewMatcher := func(typ labels.MatchType, label, value string) *labels.Matcher {
		matcher, err := labels.NewMatcher(typ, label, value)
		if err != nil {
			t.Fatalf("failed to create matcher: %v", err)
		}
		return matcher
	}

	MatchersToString := func(matchers []*labels.Matcher) []string {
		result := make([]string, 0, len(matchers))
		for _, m := range matchers {
			result = append(result, m.String())
		}
		return result
	}

	tests := []struct {
		expr         string
		metricRegexp string
		expected     []*labels.Matcher
		expectedErr  error
	}{
		{expr: "up{bar='foo'}", metricRegexp: "kube_pod_labels", expected: []*labels.Matcher{}, expectedErr: nil},
		{expr: "up{bar='foo'}", metricRegexp: "up", expected: []*labels.Matcher{
			MustNewMatcher(labels.MatchEqual, "__name__", "up"), MustNewMatcher(labels.MatchEqual, "bar", "foo"),
		}, expectedErr: nil},
		{expr: "up{bar!='foo'}", metricRegexp: ".*", expected: []*labels.Matcher{
			MustNewMatcher(labels.MatchEqual, "__name__", "up"), MustNewMatcher(labels.MatchNotEqual, "bar", "foo"),
		}, expectedErr: nil},
		{expr: "up{bar='foo', bar2!~'foo'}", metricRegexp: "up", expected: []*labels.Matcher{
			MustNewMatcher(labels.MatchEqual, "bar", "foo"), MustNewMatcher(labels.MatchNotRegexp, "bar2", "foo"), MustNewMatcher(labels.MatchEqual, "__name__", "up"),
		}, expectedErr: nil},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_case_%d", i), func(t *testing.T) {
			metricRegexp := regexp.MustCompile(test.metricRegexp)
			selectors, err := getLabelMatchersForMetricRegexp(test.expr, metricRegexp)
			if err != nil {
				t.Fatalf("failed to get label matchers for metric regexp: %v", err)
			}
			// Convert matchers to string for comparison, as we can not compare labels.Matcher directly due to regexp field
			selectorsString := MatchersToString(selectors)
			expectedMatchersString := MatchersToString(test.expected)
			assert.ElementsMatch(t, selectorsString, expectedMatchersString, "Expected label matchers %v, but got %v", expectedMatchersString, selectorsString)
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
			}
		})
	}
}

func TestGetExpressionMetrics(t *testing.T) {
	type res struct {
		Name     string
		Selector string
	}
	tests := []struct {
		expr        string
		expected    []res
		expectedErr error
	}{
		{expr: "up{foo='bar'}", expected: []res{{Name: "up", Selector: `up{foo="bar"}`}}},
		{expr: "sum(up) by (foo)", expected: []res{{Name: "up", Selector: `up`}}},
		{expr: "up * on(foo) up", expected: []res{{Name: "up", Selector: `up`}, {Name: "up", Selector: `up`}}},
		{expr: "up{foo='bar'} + up{bar='baz'}", expected: []res{{Name: "up", Selector: `up{foo="bar"}`}, {Name: "up", Selector: `up{bar="baz"}`}}},
		{expr: "avg_over_time(up{foo='bar'}[1h])", expected: []res{{Name: "up", Selector: `up{foo="bar"}`}}},
		{expr: "up{foo=~'bar.*'}", expected: []res{{Name: "up", Selector: `up{foo=~"bar.*"}`}}},
		{expr: "up{foo!~'bar.*'}", expected: []res{{Name: "up", Selector: `up{foo!~"bar.*"}`}}},
		{expr: "up{foo='bar'} offset 1h", expected: []res{{Name: "up", Selector: `up{foo="bar"}`}}},
	}

	for _, test := range tests {
		metrics, err := getExpressionMetrics(test.expr)
		var results []res
		for _, metric := range metrics {
			results = append(results, res{Name: metric.Name, Selector: metric.VectorSelector.String()})
		}
		if !reflect.DeepEqual(results, test.expected) {
			t.Errorf("Expected metric names %v, but got %v", test.expected, results)
		}
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}

func TestGetExpressionSelectors(t *testing.T) {
	tests := []struct {
		expr        string
		expected    []string
		expectedErr error
	}{
		{expr: "up{foo='bar'}", expected: []string{"up{foo=\"bar\"}"}},
		{expr: "sum(up) by (foo)", expected: []string{"up"}},
		{expr: "up * on(foo) up", expected: []string{"up", "up"}},
		{expr: "up{foo='bar'} + up{bar='baz'}", expected: []string{"up{foo=\"bar\"}", "up{bar=\"baz\"}"}},
		{expr: "avg_over_time(up{foo='bar'}[1h])", expected: []string{"up{foo=\"bar\"}"}},
		{expr: "up{foo=~'bar.*'}", expected: []string{"up{foo=~\"bar.*\"}"}},
		{expr: "up{foo!~'bar.*'}", expected: []string{"up{foo!~\"bar.*\"}"}},
		{expr: "up{foo='bar'} offset 1h", expected: []string{"up{foo=\"bar\"}"}},
	}

	for _, test := range tests {
		selectors, err := getExpressionSelectors(test.expr)
		if !reflect.DeepEqual(selectors, test.expected) {
			t.Errorf("Expected selectors %v, but got %v", test.expected, selectors)
		}
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}

func TestGetVectorSelectorMetricName(t *testing.T) {
	tests := []struct {
		vectorSelectorString string
		expected             string
	}{
		{vectorSelectorString: "up", expected: "up"},
		{vectorSelectorString: "up{foo='bar'}", expected: "up"},
		{vectorSelectorString: "up{foo='bar'}[1h]", expected: "up"},
		{vectorSelectorString: "up{foo=~'bar.*'}", expected: "up"},
		{vectorSelectorString: "up{foo!~'bar.*'}", expected: "up"},
		{vectorSelectorString: "{__name__='up'}", expected: "up"},
		{vectorSelectorString: "{__name__=~'up.*'}", expected: ""},
		{vectorSelectorString: "{__name__!~'up.*', foo='bar'}", expected: ""},
		{vectorSelectorString: "{__name__='up'}[1h]", expected: "up"},
	}

	for _, test := range tests {
		promQl, err := parser.ParseExpr(test.vectorSelectorString)
		assert.NoError(t, err)
		var selectors []*parser.VectorSelector
		parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
			if v, ok := n.(*parser.VectorSelector); ok {
				selectors = append(selectors, &parser.VectorSelector{Name: v.Name, LabelMatchers: v.LabelMatchers})
			}
			return nil
		})
		assert.Len(t, selectors, 1)
		result := getVectorSelectorMetricName(selectors[0])
		if result != test.expected {
			t.Errorf("Expected metric name %q, but got %q", test.expected, result)
		}
	}
}
