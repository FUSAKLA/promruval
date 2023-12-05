package validator

import (
	"reflect"
	"testing"

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
		if err != test.expectedErr {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}

func TestGetExpressionUsedMetricNames(t *testing.T) {
	tests := []struct {
		expr        string
		expected    []string
		expectedErr error
	}{
		{expr: "up{foo='bar'}", expected: []string{"up"}},
		{expr: "sum(up) by (foo)", expected: []string{"up"}},
		{expr: "up * on(foo) up", expected: []string{"up"}},
		{expr: "up{foo='bar'} + up{bar='baz'}", expected: []string{"up"}},
		{expr: "avg_over_time(up{foo='bar'}[1h])", expected: []string{"up"}},
		{expr: "up{foo=~'bar.*'}", expected: []string{"up"}},
		{expr: "up{foo!~'bar.*'}", expected: []string{"up"}},
		{expr: "up{foo='bar'} offset 1h", expected: []string{"up"}},
	}

	for _, test := range tests {
		metricNames, err := getExpressionUsedMetricNames(test.expr)
		if !reflect.DeepEqual(metricNames, test.expected) {
			t.Errorf("Expected metric names %v, but got %v", test.expected, metricNames)
		}
		if err != test.expectedErr {
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
		if err != test.expectedErr {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}
