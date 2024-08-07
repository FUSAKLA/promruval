package validator

import (
	"fmt"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

func newExpressionIsValidLogQL(_ yaml.Node) (Validator, error) {
	return &expressionIsValidLogQL{}, nil
}

type expressionIsValidLogQL struct{}

func (h expressionIsValidLogQL) String() string {
	return "expression is a valid LogQL query"
}

func (h expressionIsValidLogQL) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if _, err := syntax.ParseExpr(rule.Expr); err != nil {
		return []error{fmt.Errorf("expression %s is not a valid LogQL query: %w", rule.Expr, err)}
	}
	return []error{}
}

func newLogQLExpressionUsesRangeAggregation(_ yaml.Node) (Validator, error) {
	return &logQLExpressionUsesRangeAggregation{}, nil
}

type logQLExpressionUsesRangeAggregation struct{}

func (h logQLExpressionUsesRangeAggregation) String() string {
	return "LogQL expression in rules must use rate, count_over_time or any range aggregation, see https://grafana.com/docs/loki/latest/query/metric_queries/#log-range-aggregations"
}

func (h logQLExpressionUsesRangeAggregation) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := syntax.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("expression %s is not a valid LogQL query: %w", rule.Expr, err)}
	}
	usesRangeAggregation := false
	expr.Walk(func(e syntax.Expr) {
		if _, ok := e.(*syntax.RangeAggregationExpr); ok {
			usesRangeAggregation = true
		}
	})
	if usesRangeAggregation {
		return []error{}
	}
	return []error{fmt.Errorf("expression %s does not use any of the range aggregation which is required in rules, see https://grafana.com/docs/loki/latest/query/metric_queries/#log-range-aggregations", rule.Expr)}
}

func newlogQlExpressionUsesFiltersFirst(_ yaml.Node) (Validator, error) {
	return &logQlExpressionUsesFiltersFirst{}, nil
}

type logQlExpressionUsesFiltersFirst struct{}

func (h logQlExpressionUsesFiltersFirst) String() string {
	return "LogQL expression should use a line filter expressions as first in the pipeline to optimize the query efficiency, see https://grafana.com/docs/loki/latest/query/log_queries/#line-filter-expression"
}

func (h logQlExpressionUsesFiltersFirst) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := syntax.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("expression %s is not a valid LogQL query: %w", rule.Expr, err)}
	}

	filterAfterNonFilter := false
	expr.Walk(func(e syntax.Expr) {
		if _, ok := e.(*syntax.PipelineExpr); ok {
			nonFilterEncountered := false
			e.Walk(func(e syntax.Expr) {
				switch e.(type) {
				case *syntax.PipelineExpr:
					return
				case *syntax.MatchersExpr:
					return
				case *syntax.LineFilterExpr:
					if nonFilterEncountered {
						filterAfterNonFilter = true
					}
					return
				}
				nonFilterEncountered = true
			})
		}
	})
	if !filterAfterNonFilter {
		return []error{}
	}
	return []error{fmt.Errorf("LogQL expression should use a line filter expressions as first in the pipeline to optimize the query efficiency: %s", rule.Expr)}
}
