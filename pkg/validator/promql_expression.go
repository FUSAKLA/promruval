package validator

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func newExpressionIsValidPromQL(_ yaml.Node) (Validator, error) {
	return &expressionIsValidPromQL{}, nil
}

type expressionIsValidPromQL struct{}

func (h expressionIsValidPromQL) String() string {
	return "expression is a valid PromQL query"
}

func (h expressionIsValidPromQL) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if _, err := parser.ParseExpr(rule.Expr); err != nil {
		return []error{fmt.Errorf("expression %s is not a valid PromQL query: %w", rule.Expr, err)}
	}
	return []error{}
}

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

type expressionUsesOnlyAllowedLabelsForMetricRegexp struct {
	allowedLabels    map[string]struct{}
	metricNameRegexp *regexp.Regexp
}

func newExpressionUsesOnlyAllowedLabelsForMetricRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		AllowedLabels    []string              `yaml:"allowedLabels"`
		MetricNameRegexp RegexpWildcardDefault `yaml:"metricNameRegexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.AllowedLabels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	allowedLabels := labelsMap(params.AllowedLabels)
	// Metric name label is implicitly allowed
	allowedLabels[metricNameLabel] = struct{}{}
	return &expressionUsesOnlyAllowedLabelsForMetricRegexp{allowedLabels: allowedLabels, metricNameRegexp: params.MetricNameRegexp.Regexp}, nil
}

func (h expressionUsesOnlyAllowedLabelsForMetricRegexp) String() string {
	allowedLabelsSlice := []string{}
	for l := range h.allowedLabels {
		allowedLabelsSlice = append(allowedLabelsSlice, l)
	}
	return fmt.Sprintf("expression only uses allowed labels `%s` for metrics matching regexp %s", strings.Join(allowedLabelsSlice, "`,`"), h.metricNameRegexp)
}

func (h expressionUsesOnlyAllowedLabelsForMetricRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	usedLabels, err := getExpressionUsedLabelsForMetric(rule.Expr, h.metricNameRegexp)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, l := range usedLabels {
		if _, ok := h.allowedLabels[l]; !ok {
			errs = append(errs, fmt.Errorf("forbidden label `%s` used in expression in combination with metric %s (regexp)", l, h.metricNameRegexp))
		}
	}
	return errs
}

type expressionUsesOnlyAllowedLabelValuesForMetricRegexp struct {
	allowedLabelValues map[string][]string
	metricNameRegexp   *regexp.Regexp
}

func newExpressionUsesOnlyAllowedLabelValuesForMetricRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		AllowedLabelValues map[string][]string   `yaml:"allowedLabelValues"`
		MetricNameRegexp   RegexpWildcardDefault `yaml:"metricNameRegexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.AllowedLabelValues) == 0 {
		return nil, fmt.Errorf("missing allowed label values")
	}
	return &expressionUsesOnlyAllowedLabelValuesForMetricRegexp{allowedLabelValues: params.AllowedLabelValues, metricNameRegexp: params.MetricNameRegexp.Regexp}, nil
}

func (h expressionUsesOnlyAllowedLabelValuesForMetricRegexp) String() string {
	var allowedLabelValues string
	for label, values := range h.allowedLabelValues {
		allowedLabelValues += fmt.Sprintf(" %s: [%s]", label, strings.Join(values, ","))
	}
	return fmt.Sprintf("for metrics matching regexp '%s', given lables use only specified values:%s\n", h.metricNameRegexp, allowedLabelValues)
}

func (h expressionUsesOnlyAllowedLabelValuesForMetricRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	matchers, err := getLabelMatchersForMetricRegexp(rule.Expr, h.metricNameRegexp)
	errs := []error{}
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, matcher := range matchers {
		var allowedLabels []string
		var ok, matchFound bool
		if allowedLabels, ok = h.allowedLabelValues[matcher.Name]; !ok {
			// no whitelist is configured for this label
			continue
		}
		if matcher.Type == labels.MatchNotRegexp {
			// It does not make sense to validate label values whitelist for negative regexp matchers
			continue
		}
		for _, value := range allowedLabels {
			if matcher.Matches(value) {
				matchFound = true
				break
			}
		}
		if !matchFound {
			errs = append(errs, fmt.Errorf("none of the whitelisted label values matched for label %s", matcher.Name))
		}

	}
	return errs
}

type expressionDoesNotUseLabelsForMetricRegexp struct {
	labels           map[string]struct{}
	metricNameRegexp *regexp.Regexp
}

func newExpressionDoesNotUseLabelsForMetricRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels           []string              `yaml:"labels"`
		MetricNameRegexp RegexpWildcardDefault `yaml:"metricNameRegexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &expressionDoesNotUseLabelsForMetricRegexp{labels: labelsMap(params.Labels), metricNameRegexp: params.MetricNameRegexp.Regexp}, nil
}

func (h expressionDoesNotUseLabelsForMetricRegexp) String() string {
	labelsSlice := []string{}
	for l := range h.labels {
		labelsSlice = append(labelsSlice, l)
	}
	return fmt.Sprintf("expression does not use labels `%s` for metrics matching regexp %s in the expr", strings.Join(labelsSlice, "`,`"), h.metricNameRegexp)
}

func (h expressionDoesNotUseLabelsForMetricRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	usedLabels, err := getExpressionUsedLabelsForMetric(rule.Expr, h.metricNameRegexp)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, l := range usedLabels {
		if _, ok := h.labels[l]; ok {
			errs = append(errs, fmt.Errorf("forbidden label `%s` used in expression in combination with metric %s (regexp)", l, h.metricNameRegexp))
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
		if v, ok := n.(*parser.Call); ok {
			if v != nil && v.Func != nil && v.Func.Name == "irate" {
				errs = []error{fmt.Errorf("you should not use the `irate` function in rules, for more info see https://www.robustperception.io/avoid-irate-in-alerts/")}
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
		if v, ok := n.(*parser.Call); ok {
			if v == nil || v.Func == nil || (v.Func.Name != "rate" && v.Func.Name != "increase") {
				return nil
			}
			for _, ch := range parser.Children(n) {
				if m, ok := ch.(*parser.MatrixSelector); ok {
					vs, ok := m.VectorSelector.(*parser.VectorSelector)
					if !ok {
						continue
					}
					if !match.MatchString(vs.Name) {
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
		if n, ok := n.(*parser.AggregateExpr); ok {
			agg := n.Op
			if !agg.IsAggregator() {
				return nil
			}
			for _, p := range ns {
				if p, ok := p.(*parser.Call); ok {
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
	params := struct {
		timeSeriesLimit         int           `yaml:"timeSeriesLimit"`
		evaluationDurationLimit time.Duration `yaml:"evaluationDurationLimit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionCanBeEvaluated{
		timeSeriesLimit:         params.timeSeriesLimit,
		evaluationDurationLimit: params.evaluationDurationLimit,
	}, nil
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

func (h expressionCanBeEvaluated) Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	var errs []error
	if prometheusClient == nil {
		log.Error("missing the `prometheus` section of configuration for querying prometheus, skipping check that requires it...")
		return nil
	}
	count, duration, err := prometheusClient.QueryStats(rule.Expr, group.SourceTenants)
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

func (h expressionUsesExistingLabels) Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	if prometheusClient == nil {
		log.Error("missing the `prometheus` section of configuration for querying prometheus, skipping check that requires it...")
		return nil
	}
	usedLabels, err := getExpressionUsedLabels(rule.Expr)
	if err != nil {
		return []error{err}
	}
	var errs []error
	knownLabels, err := prometheusClient.Labels(group.SourceTenants)
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
	params := struct {
		maximumMatchingSeries int `yaml:"maximumMatchingSeries"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionSelectorsMatchesAnything{
		maximumMatchingSeries: params.maximumMatchingSeries,
	}, nil
}

type expressionSelectorsMatchesAnything struct {
	maximumMatchingSeries int `yaml:"maximumMatchingSeries"`
}

func (h expressionSelectorsMatchesAnything) String() string {
	return "expression selectors actually matches any series in Prometheus"
}

func (h expressionSelectorsMatchesAnything) Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
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
		matchingSeries, err := prometheusClient.SelectorMatchingSeries(s, group.SourceTenants)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if matchingSeries == 0 {
			errs = append(errs, fmt.Errorf("selector `%s` does not match any actual series in Prometheus", s))
		}
		if h.maximumMatchingSeries != 0 && matchingSeries > h.maximumMatchingSeries {
			errs = append(errs, fmt.Errorf("selector `%s` matches %d series which exceeds the limit %d", s, matchingSeries, h.maximumMatchingSeries))
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
		MetricNameRegexps []RegexpEmptyDefault `yaml:"metricNameRegexps"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	v := expressionDoesNotUseMetrics{}
	for _, r := range params.MetricNameRegexps {
		v.metricNameRegexps = append(v.metricNameRegexps, r.Regexp)
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

func newExpressionIsWellFormatted(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		SkipExpressionsWithComments bool `yaml:"skipExpressionsWithComments"`
		ShowFormatted               bool `yaml:"showExpectedForm"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionIsWellFormatted{showFormatted: params.ShowFormatted, skipExpressionsWithComments: params.SkipExpressionsWithComments}, nil
}

type expressionIsWellFormatted struct {
	skipExpressionsWithComments bool
	showFormatted               bool
}

func (h expressionIsWellFormatted) String() string {
	return "expression is well formatted as would `promtool promql format` do or similar online tool such as https://o11y.tools/promqlparser/"
}

var commentRegexp = regexp.MustCompile(`\s*#.*`)

func (h expressionIsWellFormatted) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if h.skipExpressionsWithComments && commentRegexp.MatchString(rule.Expr) {
		return nil
	}
	originalExpr := commentRegexp.ReplaceAllString(strings.TrimSpace(rule.Expr), "")
	expr, err := parser.ParseExpr(originalExpr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	prettified := expr.Pretty(0)
	if originalExpr == prettified {
		return []error{}
	}
	errorText := "expression is not well formatted, use `promtool promql format`, Prometheus UI or some online tool such as https://o11y.tools/promqlparser/"
	if h.showFormatted {
		errorText += fmt.Sprintf(", the expected form is:\n%s", prettified)
	}
	return []error{errors.New(errorText)}
}

func newExpressionDoesNotUseExperimentalFunctions(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionDoesNotUseExperimentalFunctions{}, nil
}

type expressionDoesNotUseExperimentalFunctions struct{}

func (h expressionDoesNotUseExperimentalFunctions) String() string {
	return "expression does not use any experimental PromQL functions"
}

func (h expressionDoesNotUseExperimentalFunctions) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	promQl, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	forbiddenFuncs := []string{}
	parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
		if fnCall, ok := n.(*parser.Call); ok {
			if fnCall.Func != nil && fnCall.Func.Experimental {
				forbiddenFuncs = append(forbiddenFuncs, fnCall.Func.Name)
			}
		}
		return nil
	})
	if len(forbiddenFuncs) > 0 {
		return []error{fmt.Errorf("expression uses experimental functions: %s", strings.Join(forbiddenFuncs, ", "))}
	}
	return []error{}
}

func newExpressionUsesUnderscoresInLargeNumbers(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionUsesUnderscoresInLargeNumbers{}, nil
}

type expressionUsesUnderscoresInLargeNumbers struct{}

func (h expressionUsesUnderscoresInLargeNumbers) String() string {
	return "expression uses underscores as separators in large numbers in PromQL expression. Example: 1_000_000"
}

var numberRegexp = regexp.MustCompile(`^\d+$`)

func (h expressionUsesUnderscoresInLargeNumbers) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	promQl, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	invalidNumbers := []string{}
	parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
		if number, ok := n.(*parser.NumberLiteral); ok {
			numberStr := rule.Expr[number.PosRange.Start:number.PosRange.End]
			// Ignore numbers that use 10e2 notation and duration notation (1m, 1h, etc.) and numbers that are less than 1000 where the underscore is not needed
			if numberRegexp.MatchString(numberStr) && number.Val >= 1000 && !strings.Contains(numberStr, "_") {
				invalidNumbers = append(invalidNumbers, number.String())
			}
		}
		return nil
	})
	if len(invalidNumbers) > 0 {
		return []error{fmt.Errorf("expression should use _ in large numbers (example: 1_000_0000) for better readability in these numbers: %s", strings.Join(invalidNumbers, ", "))}
	}
	return []error{}
}

func newExpressionDoesNotUseClassicHistogramBucketOperations(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &expressionDoesNotUseClassicHistogramBucketOperations{}, nil
}

type expressionDoesNotUseClassicHistogramBucketOperations struct{}

func (h expressionDoesNotUseClassicHistogramBucketOperations) String() string {
	return "expression does not do any binary operations between histogram buckets, it can be dangerous because of inconsistency in the data if sent over remote write for example"
}

const (
	histogramBucketSuffix    = "_bucket"
	histogramBucketLabelName = "le"
)

func (h expressionDoesNotUseClassicHistogramBucketOperations) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	promQl, err := parser.ParseExpr(rule.Expr)
	if err != nil {
		return []error{fmt.Errorf("failed to parse expression `%s`: %w", rule.Expr, err)}
	}
	var errs []error
	parser.Inspect(promQl, func(n parser.Node, _ []parser.Node) error {
		if op, ok := n.(*parser.BinaryExpr); ok {
			lhsVectorSelectors, err := getExpressionVectorSelectors(op.LHS.String())
			if err != nil {
				return err
			}
			rhsVectorSelectors, err := getExpressionVectorSelectors(op.RHS.String())
			if err != nil {
				return err
			}
			// If there is more then one vector selector on either side of the binary operation, we can't be sure that it is a histogram bucket operation
			if len(lhsVectorSelectors) == 1 && len(rhsVectorSelectors) == 1 {
				lhs := lhsVectorSelectors[0]
				rhs := rhsVectorSelectors[0]
				lhsLabelsUsedInSelector, _ := labelsUsedInSelectorForMetric(lhs, nil)
				rhsLabelsUsedInSelector, _ := labelsUsedInSelectorForMetric(rhs, nil)
				// Only detect if the metric name is the same and both selectors use the le label in selector
				if lhs.Name == rhs.Name && strings.HasSuffix(lhs.Name, histogramBucketSuffix) && slices.Contains(lhsLabelsUsedInSelector, histogramBucketLabelName) && slices.Contains(rhsLabelsUsedInSelector, "le") {
					errs = append(errs, fmt.Errorf("expression does binary operation between histogram buckets `%s` and `%s`", lhs.Name, rhs.Name))
				}
			}
		}
		return nil
	})
	return errs
}
