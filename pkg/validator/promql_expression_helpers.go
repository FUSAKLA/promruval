package validator

import (
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func init() {
	// Enable experimental functions in promql parser.
	parser.EnableExperimentalFunctions = true
}

const metricNameLabel = "__name__"

func labelsMap(l []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, label := range l {
		m[label] = struct{}{}
	}
	return m
}

// Returns true in case metric name selector matches given regexp and a list used labels, false and empty list otherwise.
func labelsUsedInSelectorForMetric(selector *parser.VectorSelector, metricRegexp *regexp.Regexp) (usedLabels []string, metricUsed bool) {
	for _, m := range selector.LabelMatchers {
		if metricRegexp != nil && m.Name == metricNameLabel && m.Type == labels.MatchEqual && metricRegexp.MatchString(m.Value) {
			metricUsed = true
		}
		usedLabels = append(usedLabels, m.Name)
	}
	return usedLabels, metricUsed
}

func parserCallStringArgValue(e parser.Expr) string {
	val, ok := e.(*parser.StringLiteral)
	if !ok {
		return "" // Ignore the error, this shouldn't happen anyway, parser should already catch this.
	}
	return val.Val
}

func getLabelMatchersForMetricRegexp(expr string, metricRegexp *regexp.Regexp) ([]*labels.Matcher, error) {
	var err error
	matchers := []*labels.Matcher{}

	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return matchers, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	parser.Inspect(promQl, func(node parser.Node, _ []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if ok {
			name := getVectorSelectorMetricName(vs)
			if metricRegexp.MatchString(name) {
				matchers = append(matchers, vs.LabelMatchers...)
			}
			return nil
		}
		return nil
	})
	return matchers, nil
}

// Returns a list of labels which are used in given expr in relation to given metric.
// It traverses the whole expression tree top to bottom and collects all labels used in selectors, operators, functions etc.
// In case of vector matching, it also collects labels used in vector matching only relevant to the part of the expression where the metric is used.
// If the vector matching uses grouping, any labels used on top of the expression are not validated, since they might come from the other side of the expression.
func getExpressionUsedLabelsForMetric(expr string, metricRegexp *regexp.Regexp) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	var metricInExpr bool
	var usedLabels []string

	labelsUpInExpr := func(path []parser.Node) []string {
		usedLabels := map[string]struct{}{}
		for i, n := range path {
			switch v := n.(type) {
			case *parser.AggregateExpr:
				for _, l := range v.Grouping {
					usedLabels[l] = struct{}{}
				}
			case *parser.BinaryExpr:
				if v.VectorMatching == nil {
					continue
				}
				// If any group_left/group_right is used, we need to reset the used labels, since any labels used on top of this expression might come from the other side of the expression.
				if v.VectorMatching.Include != nil {
					usedLabels = map[string]struct{}{}
				}
				// Validate only the on(...) labels. The ignoring(...) might target the other side of the binary expression.
				if v.VectorMatching.On {
					for _, l := range v.VectorMatching.MatchingLabels {
						usedLabels[l] = struct{}{}
					}
				}
				// We want to validate the group_left/group_right labels only if the validated metric is on the "one" of the many-one/one-to.many side.
				nextExpr := path[i+1].String()
				if (v.VectorMatching.Card == parser.CardManyToOne && v.RHS.String() == nextExpr) || (v.VectorMatching.Card == parser.CardOneToMany && v.LHS.String() == nextExpr) {
					for _, l := range v.VectorMatching.Include {
						usedLabels[l] = struct{}{}
					}
				}
			case *parser.Call:
				switch v.Func.Name {
				case "label_replace":
					// Any PromQL "above" this label_replace can use the destination synthetic label, so drop it from the list of already used labels.
					delete(usedLabels, parserCallStringArgValue(v.Args[1]))
					usedLabels[parserCallStringArgValue(v.Args[3])] = struct{}{} // The source_label is interesting for us
				case "label_join":
					delete(usedLabels, parserCallStringArgValue(v.Args[1]))
					// label_join is variadic, so we need to iterate over all labels that are used in the expression
					for _, l := range v.Args[3:] {
						usedLabels[parserCallStringArgValue(l)] = struct{}{}
					}
				case "sort_by_label":
					for _, l := range v.Args[1:] {
						usedLabels[parserCallStringArgValue(l)] = struct{}{}
					}
				case "sort_by_label_desc":
					for _, l := range v.Args[1:] {
						usedLabels[parserCallStringArgValue(l)] = struct{}{}
					}
				}
			}
		}
		delete(usedLabels, "") // Used in case of errors so just drop it
		return slices.Collect(maps.Keys(usedLabels))
	}

	parser.Inspect(promQl, func(n parser.Node, path []parser.Node) error {
		v, isVectorSelector := n.(*parser.VectorSelector)
		if !isVectorSelector {
			return nil
		}
		selectorUsedLabels, ok := labelsUsedInSelectorForMetric(v, metricRegexp)
		if ok {
			metricInExpr = true
			usedLabels = append(usedLabels, selectorUsedLabels...)
			// The path does not contain the current node, so we need to append it since some cases need also the last node.
			usedLabels = append(usedLabels, labelsUpInExpr(append(path, n))...)
		}
		return nil
	})
	if !metricInExpr {
		return []string{}, nil
	}
	slices.Sort(usedLabels)
	return slices.Compact(usedLabels), nil
}

func getExpressionUsedLabels(expr string) ([]string, error) {
	return getExpressionUsedLabelsForMetric(expr, regexp.MustCompile(".*"))
}

func getExpressionUsedLabelsForEveryAggregation(expr string) (map[string]struct{}, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return map[string]struct{}{}, fmt.Errorf("failed to parse expression `%s`: %w", expr, err)
	}
	var usedLabels map[string]struct{}

	labelsUpInExpr := func(path []parser.Node) map[string]struct{} {
		for _, n := range path {
			switch v := n.(type) {
			case *parser.AggregateExpr:
				usedLabels := map[string]struct{}{}
				for _, l := range v.Grouping {
					usedLabels[l] = struct{}{}
				}
				return usedLabels
			}
		}

		return nil
	}

	parser.Inspect(promQl, func(n parser.Node, path []parser.Node) error {
		usedLabelsInExpr := labelsUpInExpr(path)
		if usedLabels == nil {
			usedLabels = usedLabelsInExpr
		}
		if usedLabelsInExpr != nil {
			for k := range usedLabels {
				if _, exists := usedLabelsInExpr[k]; !exists {
					delete(usedLabels, k)
				}
			}

		}
		return nil
	})

	return usedLabels, nil
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
			if m.Name == metricNameLabel && m.Type == labels.MatchEqual {
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
