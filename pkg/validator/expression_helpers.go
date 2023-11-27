package validator

import (
	"fmt"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
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
	return usedLabels, nil
}

func getExpressionSelectors(expr string) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var selectors []string
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.VectorSelector:
			s := &parser.VectorSelector{Name: v.Name, LabelMatchers: v.LabelMatchers}
			selectors = append(selectors, s.String())
		}
		return nil
	})
	return selectors, nil
}

func getExpressionMetricsNames(expr string) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var names []string
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.VectorSelector:
			names = append(names, getMetricNameFromVectorSelector(v))
		}
		return nil
	})
	return names, nil
}

func getMetricNameFromVectorSelector(v *parser.VectorSelector) string {
	s := &parser.VectorSelector{Name: v.Name}
	name := s.String()
	if name == "" {
		name = getMetricNameFromLabels(v.LabelMatchers)
	}
	return name
}

func getMetricNameFromLabels(labels []*labels.Matcher) string {
	for _, l := range labels {
		if l.Name == "__name__" {
			return l.Value
		}
	}
	return ""
}
