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
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	var usedLabels []string
	parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
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

func getExpressionVectorSelectors(expr string) ([]*parser.VectorSelector, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []*parser.VectorSelector{}, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	var selectors []*parser.VectorSelector
	parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
		if v, ok := n.(*parser.VectorSelector); ok {
			selectors = append(selectors, &parser.VectorSelector{Name: v.Name, LabelMatchers: v.LabelMatchers})
		}
		return nil
	})
	return selectors, nil
}

func getVectorSelectorMetricName(selector *parser.VectorSelector) string {
	if selector.Name == "" {
		for _, m := range selector.LabelMatchers {
			if m.Name == "__name__" && m.Type == labels.MatchEqual {
				return m.Value
			}
		}
	}
	return selector.Name
}

// MetricWithVectorSelector is a struct that contains a metric name and a vector selector where it is used, to give a context, in the error messages.
type MetricWithVectorSelector struct {
	VectorSelector *parser.VectorSelector
	Name           string
}

func getExpressionMetrics(expr string) ([]MetricWithVectorSelector, error) {
	metrics := []MetricWithVectorSelector{}
	vectorSelectors, err := getExpressionVectorSelectors(expr)
	if err != nil {
		return metrics, err
	}
	for _, s := range vectorSelectors {
		metrics = append(metrics, MetricWithVectorSelector{VectorSelector: s, Name: getVectorSelectorMetricName(s)})
	}
	return metrics, nil
}

func getExpressionSelectors(expr string) ([]string, error) {
	vectorSelectors, err := getExpressionVectorSelectors(expr)
	if err != nil {
		return []string{}, err
	}
	selectors := make([]string, 0, len(vectorSelectors))
	for _, s := range vectorSelectors {
		selectors = append(selectors, s.String())
	}
	return selectors, nil
}
