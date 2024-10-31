package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/forPelevin/gomoji"
	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type SourceTenantMetrics struct {
	Regexp         string `yaml:"regexp"`
	NegativeRegexp string `yaml:"negativeRegexp"`
	Description    string `yaml:"description"`
}

func newHasSourceTenantsForMetrics(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		SourceTenants map[string][]SourceTenantMetrics `yaml:"sourceTenants"`
		DefaultTenant string                           `yaml:"defaultTenant"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.SourceTenants) == 0 {
		return nil, fmt.Errorf("sourceTenants metrics mapping needs to be set")
	}
	validator := hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{}, defaultTenant: params.DefaultTenant}
	for tenant, metrics := range params.SourceTenants {
		m := make([]tenantMetrics, len(metrics))
		for i, metric := range metrics {
			compiledRegexp, err := compileAnchoredRegexp(metric.Regexp)
			if err != nil {
				return nil, fmt.Errorf("invalid metric name regexp: %s", anchorRegexp(metric.Regexp))
			}
			compiledNegativeRegexp := (*regexp.Regexp)(nil)
			if metric.NegativeRegexp != "" {
				compiledNegativeRegexp, err = compileAnchoredRegexp(metric.NegativeRegexp)
				if err != nil {
					return nil, fmt.Errorf("invalid metric name regexp: %s", anchorRegexp(metric.NegativeRegexp))
				}
			}
			m[i] = tenantMetrics{
				regexp:         compiledRegexp,
				negativeRegexp: compiledNegativeRegexp,
				description:    metric.Description,
			}
		}
		validator.sourceTenants[tenant] = m
	}
	return &validator, nil
}

type tenantMetrics struct {
	regexp         *regexp.Regexp
	negativeRegexp *regexp.Regexp
	description    string
}

type hasSourceTenantsForMetrics struct {
	sourceTenants map[string][]tenantMetrics
	defaultTenant string
}

func (h hasSourceTenantsForMetrics) String() string {
	tenantStrings := []string{}
	tenants := make([]string, 0, len(h.sourceTenants))
	for tenant := range h.sourceTenants {
		tenants = append(tenants, tenant)
	}
	slices.Sort(tenants)
	for _, t := range tenants {
		for _, m := range h.sourceTenants[t] {
			tenantStrings = append(tenantStrings, fmt.Sprintf("`%s`:   `%s` (%s)", t, m.regexp.String(), m.description))
		}
	}
	return fmt.Sprintf("rule group, the rule belongs to, has the required `source_tenants` configured, according to the mapping of metric names to tenants: \n        %s", strings.Join(tenantStrings, "\n        "))
}

func (h hasSourceTenantsForMetrics) Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	usedMetrics, err := getExpressionMetrics(rule.Expr)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, usedMetric := range usedMetrics {
		for tenant, metrics := range h.sourceTenants {
			for _, metric := range metrics {
				if !metric.regexp.MatchString(usedMetric.Name) {
					continue
				}
				if metric.negativeRegexp != nil && metric.negativeRegexp.MatchString(usedMetric.Name) {
					continue
				}
				if len(group.SourceTenants) == 0 && h.defaultTenant == tenant {
					continue
				}
				if !slices.Contains(group.SourceTenants, tenant) {
					errs = append(errs, fmt.Errorf("rule uses metric `%s` of the tenant `%s`, you should set the tenant in the group's source_tenants settings", usedMetric.Name, tenant))
				}
			}
		}
	}
	return errs
}

func newDoesNotUseEmoji(_ yaml.Node) (Validator, error) {
	return &doesNotUseEmoji{}, nil
}

type doesNotUseEmoji struct{}

func (h doesNotUseEmoji) String() string {
	return "fails if any rule uses an emoji in metric name or labels"
}

func (h doesNotUseEmoji) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for k, v := range rule.Labels {
		if gomoji.ContainsEmoji(k) {
			errs = append(errs, fmt.Errorf("rule uses label named `%s` with emoji", k))
		}
		if gomoji.ContainsEmoji(v) {
			errs = append(errs, fmt.Errorf("rule uses label with value `%s` containing emoji", v))
		}
	}
	usedMetrics, err := getExpressionMetrics(rule.Expr)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	if gomoji.ContainsEmoji(rule.Record) {
		errs = append(errs, fmt.Errorf("recording rule metric name contains emoji `%s`", rule.Record))
	}
	for _, m := range usedMetrics {
		if gomoji.ContainsEmoji(m.Name) {
			errs = append(errs, fmt.Errorf("rule uses metric `%s` with emoji", m.Name))
		}
		if m.VectorSelector != nil {
			for _, l := range m.VectorSelector.LabelMatchers {
				if gomoji.ContainsEmoji(l.Name) {
					errs = append(errs, fmt.Errorf("expression uses label `%s` with emoji", l.Value))
				}
				if gomoji.ContainsEmoji(l.Value) {
					errs = append(errs, fmt.Errorf("expression uses label with value `%s` containing emoji", l.Value))
				}
			}
		}
	}
	return errs
}

func newDoesNotUseUTF8(_ yaml.Node) (Validator, error) {
	return &doesNotUseEmoji{}, nil
}

type doesNotUseUTF8 struct{}

func (h doesNotUseUTF8) String() string {
	return "fails if any rule uses UTF-8 characters in metric name or labels"
}

func (h doesNotUseUTF8) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for k := range rule.Labels {
		if !model.LabelName(k).IsValidLegacy() {
			errs = append(errs, fmt.Errorf("rule uses label named `%s` with UTF-8 characters", k))
		}
	}
	usedMetrics, err := getExpressionMetrics(rule.Expr)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	if rule.Record != "" && !model.IsValidLegacyMetricName(rule.Record) {
		errs = append(errs, fmt.Errorf("recording rule metric name contains UTF-8 characters `%s`", rule.Record))
	}
	for _, m := range usedMetrics {
		if !model.IsValidLegacyMetricName(m.Name) {
			errs = append(errs, fmt.Errorf("rule uses metric `%s` with UTF-8 characters", m.Name))
		}
		if m.VectorSelector != nil {
			for _, l := range m.VectorSelector.LabelMatchers {
				if !model.LabelName(l.Name).IsValidLegacy() {
					errs = append(errs, fmt.Errorf("expression uses label `%s` with UTF-8 characters", l.Value))
				}
			}
		}
	}
	return errs
}
