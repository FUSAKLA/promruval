package validator

import (
	"errors"
	"reflect"
	"regexp"
	"testing"

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
		labels, err := getExpressionUsedLabels(test.expr)
		assert.ElementsMatch(t, labels, test.expected, "Expected labels %v, but got %v", test.expected, labels)
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
		{expr: "kube_pod_info{} * on(pod) group_right(pod_ip) kube_pod_labels{label_app='foo'}", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "pod", "pod_ip"}},
		{expr: "kube_pod_info{} * on(pod) group_right(pod_ip) kube_pod_labels{label_app='foo'} offset 1h", metric: "kube_pod_labels", expected: []string{metricNameLabel, "label_app", "pod", "pod_ip"}},
	}

	for _, test := range tests {
		labels, err := getExpressionUsedLabelsForMetric(test.expr, regexp.MustCompile(test.metric))
		assert.ElementsMatch(t, labels, test.expected, "Expected labels %v, but got %v", test.expected, labels)
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
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
