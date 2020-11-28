package validator

import (
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"
	"strings"
	"time"
)

func newExpressionDoesNotUseOlderDataThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Limit == model.Duration(0) {
		return nil, fmt.Errorf("missing limit")
	}
	return &expressionDoesNotUseOlderDataThan{limit: params.Limit}, nil
}

type expressionDoesNotUseOlderDataThan struct {
	limit model.Duration
}

func (h expressionDoesNotUseOlderDataThan) String() string {
	return fmt.Sprintf("expression does not use data older than `%s`", h.limit)
}

func (h expressionDoesNotUseOlderDataThan) Validate(rule rulefmt.Rule) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %s", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(expr, func(n parser.Node, ns []parser.Node) error {
		// TODO(FUSAKLA) Having range query in subquery should have the time added.
		switch v := n.(type) {
		case *parser.MatrixSelector:
			if v.Range > time.Duration(h.limit) {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in matrix selector which is more than limit `%s`", model.Duration(v.Range), h.limit))
			}
		case *parser.VectorSelector:
			if v.Offset > time.Duration(h.limit) {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in vector selector which is more than limit `%s`", model.Duration(v.Offset), h.limit))
			}
		case *parser.SubqueryExpr:
			if v.Range+v.Offset > time.Duration(h.limit) {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in subquery which is more than limit `%s`", model.Duration(v.Range+v.Offset), h.limit))
			}
		}
		return nil
	})
	return errs
}

func newExpressionDoesNotUseLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels []string `yaml:"labels"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &expressionDoesNotUseLabels{labels: params.Labels}, nil
}

type expressionDoesNotUseLabels struct {
	labels []string
}

func (h expressionDoesNotUseLabels) String() string {
	return fmt.Sprintf("does not use any of the `%s` labels is in its expression", strings.Join(h.labels, "`,`"))
}

func getExpressionUsedLabels(expr string) ([]string, error) {
	promQl, err := parser.ParseExpr(expr)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse expression `%s`: %s", expr, err)
	}
	var usedLabels []string
	parser.Inspect(promQl, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.AggregateExpr:
			for _, m := range v.Grouping {
				usedLabels = append(usedLabels, m)
			}
		case *parser.VectorSelector:
			for _, m := range v.LabelMatchers {
				usedLabels = append(usedLabels, m.Name)
			}
		case *parser.BinaryExpr:
			if v.VectorMatching != nil {
				for _, m := range v.VectorMatching.Include {
					usedLabels = append(usedLabels, m)
				}
				for _, m := range v.VectorMatching.MatchingLabels {
					usedLabels = append(usedLabels, m)
				}
			}
		}
		return nil
	})
	return usedLabels, nil
}

func (h expressionDoesNotUseLabels) Validate(rule rulefmt.Rule) []error {
	usedLabels, err := getExpressionUsedLabels(rule.Expr)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, l := range usedLabels {
		for _, n := range h.labels {
			if l == n {
				errs = append(errs, fmt.Errorf("forbidden label `%s` used in expression", l))
			}
		}
	}
	return errs
}

func newExpressionDoesNotUseRangeShorterThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Limit == model.Duration(0) {
		return nil, fmt.Errorf("missing limit")
	}
	return &expressionDoesNotUseRangeShorterThan{limit: params.Limit}, nil
}

type expressionDoesNotUseRangeShorterThan struct {
	limit model.Duration
}

func (h expressionDoesNotUseRangeShorterThan) String() string {
	return fmt.Sprintf("expr does not use range selctor shorter than `%s`", h.limit)
}

func (h expressionDoesNotUseRangeShorterThan) Validate(rule rulefmt.Rule) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %s", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(expr, func(n parser.Node, ns []parser.Node) error {
		switch v := n.(type) {
		case *parser.MatrixSelector:
			if v.Range < time.Duration(h.limit) {
				errs = append(errs, fmt.Errorf("query using range `%s` smaller than limit `%s`", model.Duration(v.Range), h.limit))
			}
		case *parser.SubqueryExpr:
			if v.Range < time.Duration(h.limit) {
				errs = append(errs, fmt.Errorf("subquery using range `%s` smaller than limit `%s`", model.Duration(v.Range), h.limit))
			}
		}
		return nil
	})
	return errs
}
