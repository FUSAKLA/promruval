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

// Returns true in case selector matches given metric and a list of other labels beside __name__, false and nil otherwise.
func containsMetric(selector *parser.VectorSelector, metric string) (bool, []string) {
	var usedLabels []string
	var metricUsed bool
	for _, m := range selector.LabelMatchers {
		if m.Name == "__name__" && m.Type == labels.MatchEqual && m.Value == metric {
			metricUsed = true
			continue
		}
		usedLabels = append(usedLabels, m.Name)
	}
	return metricUsed, usedLabels
}

// Returns a list of labels which are used in given expr in relation to given metric.
// Beside labels within vector selector itself, it adds labels used in Aggregate expressions and labels used in Binary expression.
// For Binary expressions it may report false positives as the current implementation does not consider on which side of group_left/group_right is the given metric.
func getExpressionUsedLabelsForMetric(expr, metric string) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	var metricInExpr bool
	var usedLabels []string

	labelsUpInExpr := func(path []parser.Node) []string {
		usedLabels := []string{}
		for _, n := range path {
			switch v := n.(type) {
			case *parser.AggregateExpr:
				usedLabels = append(usedLabels, v.Grouping...)
			case *parser.BinaryExpr:
				if v.VectorMatching != nil {
					usedLabels = append(usedLabels, v.VectorMatching.Include...)
					usedLabels = append(usedLabels, v.VectorMatching.MatchingLabels...)
				}
			}
		}
		return usedLabels
	}

	parser.Inspect(promQl, func(n parser.Node, path []parser.Node) error {
		switch v := n.(type) {
		case *parser.VectorSelector:
			containsMetric, labels := containsMetric(v, metric)
			if containsMetric {
				metricInExpr = true
				usedLabels = append(usedLabels, labels...)
				usedLabels = append(usedLabels, labelsUpInExpr(path)...)
			}
		}
		return nil
	})
	if metricInExpr {
		slices.Sort(usedLabels)
		return slices.Compact(usedLabels), nil
	} else {
		return []string{}, nil
	}
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
