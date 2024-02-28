package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func newExpressionDoesNotUseOlderDataThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		return nil, fmt.Errorf("missing limit")
	}
	return &expressionDoesNotUseOlderDataThan{limit: time.Duration(params.Limit)}, nil
}

type expressionDoesNotUseOlderDataThan struct {
	limit time.Duration
}

func (h expressionDoesNotUseOlderDataThan) String() string {
	return fmt.Sprintf("expression does not use data older than `%s`", h.limit)
}

func (h expressionDoesNotUseOlderDataThan) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(expr, func(n parser.Node, _ []parser.Node) error {
		// TODO(FUSAKLA) Having range query in subquery should have the time added.
		switch n := n.(type) {
		case *parser.MatrixSelector:
			if n.Range > h.limit {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in matrix selector which is more than limit `%s`", model.Duration(n.Range), h.limit))
			}
		case *parser.VectorSelector:
			if n.OriginalOffset > h.limit {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in vector selector which is more than limit `%s`", model.Duration(n.OriginalOffset), h.limit))
			}
			if n.Timestamp != nil && time.Since(time.Unix(*n.Timestamp, 0)) > h.limit {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data because of @timestamp in vector selector which is more than limit `%s`", time.Since(time.Unix(*n.Timestamp, 0)), h.limit))
			}
		case *parser.SubqueryExpr:
			if n.Range+n.Offset > h.limit {
				errs = append(errs, fmt.Errorf("expr uses `%s` old data in subquery which is more than limit `%s`", model.Duration(n.Range+n.Offset), h.limit))
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

func (h expressionDoesNotUseLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
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
	return fmt.Sprintf("expression does not use range selector shorter than `%s`", h.limit)
}

func (h expressionDoesNotUseRangeShorterThan) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(expr, func(n parser.Node, _ []parser.Node) error {
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

func newExpressionDoesNotUseIrate(_ yaml.Node) (Validator, error) {
	return &expressionDoesNotUseIrate{}, nil
}

type expressionDoesNotUseIrate struct{}

func (h expressionDoesNotUseIrate) String() string {
	return "expression does not use irate"
}

func (h expressionDoesNotUseIrate) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(expr, func(n parser.Node, _ []parser.Node) error {
		switch v := n.(type) {
		case *parser.Call:
			if v != nil && v.Func != nil && v.Func.Name == "irate" {
				errs = []error{fmt.Errorf("you should not use the `irate` function in rules, for more info see https://prometheus.io/docs/prometheus/latest/querying/functions/#irate")}
			}
		}
		return nil
	})
	return errs
}

func newValidFunctionsOnCounters(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		AllowHistograms bool `yaml:"allowHistograms"`
	}{}
	params.AllowHistograms = true
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &validFunctionsOnCounters{allowHistograms: params.AllowHistograms}, nil
}

type validFunctionsOnCounters struct {
	allowHistograms bool `yaml:"allowHistograms"`
}

func (h validFunctionsOnCounters) String() string {
	msg := "uses functions `rate` and `increase` only on metrics with the `_total` suffix"
	if h.allowHistograms {
		msg += " (metrics ending with _count are exceptions since those are used by histograms)"
	}
	return msg
}

func (h validFunctionsOnCounters) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	match := regexp.MustCompile(`_total$`)
	if h.allowHistograms {
		match = regexp.MustCompile(`(_total|_count|_bucket|_sum)$`)
	}
	parser.Inspect(expr, func(n parser.Node, _ []parser.Node) error {
		switch v := n.(type) {
		case *parser.Call:
			if v == nil || v.Func == nil || (v.Func.Name != "rate" && v.Func.Name != "increase") {
				return nil
			}
			for _, ch := range parser.Children(n) {
				switch m := ch.(type) {
				case *parser.MatrixSelector:
					if !match.MatchString(m.VectorSelector.(*parser.VectorSelector).Name) {
						errs = append(errs, fmt.Errorf("`%s` function should be used only on counters and those should end with the `_total` suffix, which is not this case `%s`", v.Func.Name, n.String()))
					}
				}
			}
		}
		return nil
	})
	return errs
}

func newRateBeforeAggregation(_ yaml.Node) (Validator, error) {
	return &rateBeforeAggregation{}, nil
}

type rateBeforeAggregation struct{}

func (h rateBeforeAggregation) String() string {
	return "does not use aggregation functions before the `rate` or `increase` functions, see https://www.robustperception.io/rate-then-sum-never-sum-then-rate"
}

func (h rateBeforeAggregation) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	parser.Inspect(expr, func(n parser.Node, ns []parser.Node) error {
		switch n := n.(type) {
		case *parser.AggregateExpr:
			agg := n.Op
			if !agg.IsAggregator() {
				return nil
			}
			for _, p := range ns {
				switch p := p.(type) {
				case *parser.Call:
					funcName := p.Func.Name
					if funcName == "increase" || funcName == "rate" {
						errs = append(errs, fmt.Errorf("you should not use aggregation functions before calling the `rate` or `increase` functions as in: %s", funcName))
					}
				}
			}
		}
		return nil
	})
	return errs
}

func newExpressionCanBeEvaluated(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionCanBeEvaluated{}, nil
}

type expressionCanBeEvaluated struct {
	timeSeriesLimit         int           `yaml:"timeSeriesLimit"`
	evaluationDurationLimit time.Duration `yaml:"evaluationDurationLimit"`
}

func (h expressionCanBeEvaluated) String() string {
	msg := "expression can be successfully evaluated on the live Prometheus instance"
	if h.timeSeriesLimit > 0 {
		msg += fmt.Sprintf(" and the number of time series in the result is not higher than %d", h.timeSeriesLimit)
	}
	if h.evaluationDurationLimit != 0 {
		msg += fmt.Sprintf(" and the evaluation is not longer than %s", h.evaluationDurationLimit)
	}
	return msg
}

func (h expressionCanBeEvaluated) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	var errs []error
	if prometheusClient == nil {
		log.Error("missing the `prometheus` section of configuration for querying prometheus, skipping check that requires it...")
		return nil
	}
	_, count, duration, err := prometheusClient.Query(rule.Expr)
	if err != nil {
		return append(errs, err)
	}
	if h.timeSeriesLimit != 0 && count > h.timeSeriesLimit {
		errs = append(errs, fmt.Errorf("query returned %d series exceeding the limit %d", count, h.timeSeriesLimit))
	}
	if h.evaluationDurationLimit != 0 && duration > h.evaluationDurationLimit {
		errs = append(errs, fmt.Errorf("query took %s which exceeds the configured maximum %s", duration, h.evaluationDurationLimit))
	}
	return errs
}

func newExpressionUsesExistingLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionUsesExistingLabels{}, nil
}

type expressionUsesExistingLabels struct{}

func (h expressionUsesExistingLabels) String() string {
	return "expression uses only labels that are actually present in Prometheus"
}

func (h expressionUsesExistingLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	if prometheusClient == nil {
		log.Error("missing the `prometheus` section of configuration for querying prometheus, skipping check that requires it...")
		return nil
	}
	usedLabels, err := getExpressionUsedLabels(rule.Expr)
	if err != nil {
		return []error{err}
	}
	var errs []error
	knownLabels, err := prometheusClient.Labels()
	if err != nil {
		return []error{err}
	}
	for _, l := range usedLabels {
		known := false
		for _, k := range knownLabels {
			if l == k {
				known = true
				break
			}
		}
		if !known {
			errs = append(errs, fmt.Errorf("the label `%s` does not exist in the actual Prometheus instance", l))
		}
	}
	return errs
}

func newExpressionSelectorsMatchesAnything(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionSelectorsMatchesAnything{}, nil
}

type expressionSelectorsMatchesAnything struct{}

func (h expressionSelectorsMatchesAnything) String() string {
	return "expression selectors actually matches any series in Prometheus"
}

func (h expressionSelectorsMatchesAnything) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	if prometheusClient == nil {
		log.Error("missing the `prometheus` section of configuration for querying prometheus, skipping check that requires it...")
		return nil
	}
	var errs []error
	selectors, err := getExpressionSelectors(rule.Expr)
	if err != nil {
		return []error{err}
	}
	for _, s := range selectors {
		match, err := prometheusClient.SelectorMatch(s)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(match) == 0 {
			errs = append(errs, fmt.Errorf("selector `%s` does not match any actual series in Prometheus", s))
		}
	}
	return errs
}

func newExpressionWithNoMetricName(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionWithNoMetricName{}, nil
}

type expressionWithNoMetricName struct{}

func (e expressionWithNoMetricName) String() string {
	return "expression uses metric name in selectors"
}

func (e expressionWithNoMetricName) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	metrics, err := getExpressionMetrics(rule.Expr)
	if err != nil {
		return []error{err}
	}
	for _, v := range metrics {
		if v.Name == "" {
			errs = append(errs, fmt.Errorf("missing metric name for vector `%s`", v.VectorSelector.String()))
		}
	}
	return errs
}

func newExpressionDoesNotUseMetrics(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		MetricNameRegexps []string `yaml:"metricNameRegexps"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	v := expressionDoesNotUseMetrics{}
	for _, r := range params.MetricNameRegexps {
		compiled, err := regexp.Compile("^" + r + "$")
		if err != nil {
			return nil, err
		}
		v.metricNameRegexps = append(v.metricNameRegexps, compiled)
	}
	return &v, nil
}

type expressionDoesNotUseMetrics struct {
	metricNameRegexps []*regexp.Regexp
}

func (h expressionDoesNotUseMetrics) String() string {
	return "expression does not use any of these metrics regexps: " + strings.Join(func() []string {
		var res []string
		for _, r := range h.metricNameRegexps {
			res = append(res, r.String())
		}
		return res
	}(), ",")
}

func (h expressionDoesNotUseMetrics) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	expr, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	usedMetrics, err := getExpressionMetrics(expr.String())
	if err != nil {
		return []error{err}
	}
	for _, m := range usedMetrics {
		for _, r := range h.metricNameRegexps {
			if r.MatchString(m.Name) {
				errs = append(errs, fmt.Errorf("expression vector selector `%s` uses metric `%s` which is forbidden", m.VectorSelector.String(), m.Name))
			}
		}
	}
	return errs
}
