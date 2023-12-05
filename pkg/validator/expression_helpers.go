package validator

import (
	"fmt"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"golang.org/x/exp/slices"
)

func getExpressionUsedLabels(expr string) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var usedLabels []string
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.AggregateExpr:
			usedLabels = append(usedLabels, v.Grouping...)
		case *parser.VectorSelector:
			for _, m := range v.LabelMatchers {
				usedLabels = append(usedLabels, m.Name)
			}
		case *parser.BinaryExpr:
			if v.VectorMatching != nil {
				usedLabels = append(usedLabels, v.VectorMatching.Include...)
				usedLabels = append(usedLabels, v.VectorMatching.MatchingLabels...)
			}
		}
		return nil
	})
	slices.Sort(usedLabels)
	return slices.Compact(usedLabels), nil
}

func getExpressionVectorSelectors(expr string) ([]parser.VectorSelector, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []parser.VectorSelector{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var selectors []parser.VectorSelector
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.VectorSelector:
			selectors = append(selectors, parser.VectorSelector{Name: v.Name, LabelMatchers: v.LabelMatchers})
		}
		return nil
	})
	return selectors, nil
}

type VectorSelectorWithMetricName struct {
	Vector     *parser.VectorSelector
	MetricName string
}

func getExpressionMetricsNames(expr string) ([]VectorSelectorWithMetricName, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []VectorSelectorWithMetricName{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var vectors []VectorSelectorWithMetricName
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.VectorSelector:
			metricName := getMetricNameFromLabels(v.LabelMatchers)
			vectors = append(vectors, VectorSelectorWithMetricName{Vector: v, MetricName: metricName})
		}
		return nil
	})
	return vectors, nil
}

func getMetricNameFromLabels(labels []*labels.Matcher) string {
	for _, l := range labels {
		if l.Name == "__name__" {
			return l.Value
		}
	}
	return ""
}

func getVectorSelectorMetricName(selector parser.VectorSelector) string {
	if selector.Name == "" {
		for _, m := range selector.LabelMatchers {
			if m.Name == "__name__" && m.Type == labels.MatchEqual {
				return m.Value
			}
		}
	}
	return selector.Name
}

func getExpressionUsedMetricNames(expr string) ([]string, error) {
	vectorSelectors, err := getExpressionVectorSelectors(expr)
	if err != nil {
		return []string{}, err
	}
	var metricNames []string
	for _, s := range vectorSelectors {
		metricNames = append(metricNames, getVectorSelectorMetricName(s))
	}
	slices.Sort(metricNames)
	return slices.Compact(metricNames), nil
}

func getExpressionSelectors(expr string) ([]string, error) {
	vectorSelectors, err := getExpressionVectorSelectors(expr)
	if err != nil {
		return []string{}, err
	}
	var selectors []string
	for _, s := range vectorSelectors {
		selectors = append(selectors, s.String())
	}
	return selectors, nil
}
