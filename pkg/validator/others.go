package validator

import (
	"fmt"
	"iter"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/prometheus/prometheus/model/rulefmt"
)

type SourceTenantMetrics struct {
	Regexp         RegexpWildcardDefault `yaml:"regexp"`
	NegativeRegexp RegexpEmptyDefault    `yaml:"negativeRegexp"`
	Description    string                `yaml:"description"`
}

func newHasSourceTenantsForMetrics(unmarshal unmarshalParamsFunc) (Validator, error) {
	params := struct {
		SourceTenants map[string][]SourceTenantMetrics `yaml:"sourceTenants"`
		DefaultTenant string                           `yaml:"defaultTenant"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	if len(params.SourceTenants) == 0 {
		return nil, fmt.Errorf("sourceTenants metrics mapping needs to be set")
	}
	validator := hasSourceTenantsForMetrics{sourceTenants: map[string][]tenantMetrics{}, defaultTenant: params.DefaultTenant}
	for tenant, metrics := range params.SourceTenants {
		m := make([]tenantMetrics, len(metrics))
		for i, metric := range metrics {
			m[i] = tenantMetrics{
				regexp:         metric.Regexp.Regexp,
				negativeRegexp: metric.NegativeRegexp.Regexp,
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

func newDoesNotContainTypos(unmarshal unmarshalParamsFunc) (Validator, error) {
	params := struct {
		MaxLevenshteinDistance int      `yaml:"maxLevenshteinDistance"`
		MaxDifferenceRatio     float64  `yaml:"maxDifferenceRatio"`
		WellKnownAnnotations   []string `yaml:"wellKnownAnnotations"`
		WellKnownRuleLabels    []string `yaml:"wellKnownRuleLabels"`
		WellKnownSeriesLabels  []string `yaml:"wellKnownSeriesLabels"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	if params.MaxLevenshteinDistance > 0 && params.MaxDifferenceRatio > 0 {
		return nil, fmt.Errorf("you can only set one of `maxLevenshteinDistance` or `MaxDifferenceRatio`, not both")
	}
	if params.MaxLevenshteinDistance == 0 && params.MaxDifferenceRatio == 0 {
		return nil, fmt.Errorf("you must set either `maxLevenshteinDistance` or `MaxDifferenceRatio` to a value greater than 0")
	}
	if params.MaxLevenshteinDistance < 0 {
		return nil, fmt.Errorf("`maxLevenshteinDistance` must be greater than or equal to 0")
	}
	if params.MaxDifferenceRatio < 0 || params.MaxDifferenceRatio > 1 {
		return nil, fmt.Errorf("`MaxDifferenceRatio` must be between 0 and 1")
	}

	validator := doesNotContainTypos{
		maxLevenshteinDistance: params.MaxLevenshteinDistance,
		maxDifferenceRatio:     params.MaxDifferenceRatio,
		wellKnownAnnotations:   params.WellKnownAnnotations,
		wellKnownRuleLabels:    params.WellKnownRuleLabels,
		wellKnownSeriesLabels:  params.WellKnownSeriesLabels,
	}
	return &validator, nil
}

type doesNotContainTypos struct {
	maxLevenshteinDistance int
	maxDifferenceRatio     float64
	wellKnownAnnotations   []string
	wellKnownRuleLabels    []string
	wellKnownSeriesLabels  []string
}

func (h doesNotContainTypos) String() string {
	out := "rule does not contain typos in typos in well known:"
	sprintSlice := func(name string, values []string) string {
		return fmt.Sprintf("\n        %s: `%s`", name, strings.Join(values, "`, `"))
	}
	if len(h.wellKnownAnnotations) > 0 {
		out += sprintSlice("Annotations", h.wellKnownAnnotations)
	}
	if len(h.wellKnownRuleLabels) > 0 {
		out += sprintSlice("Rule labels", h.wellKnownRuleLabels)
	}
	if len(h.wellKnownSeriesLabels) > 0 {
		out += sprintSlice("Series labels", h.wellKnownSeriesLabels)
	}
	return out
}

func (h doesNotContainTypos) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	isTypo := func(value, wellKnown string) bool {
		dst := fuzzy.LevenshteinDistance(value, wellKnown)
		if dst == 0 {
			return false
		}
		if h.maxLevenshteinDistance > 0 {
			return dst <= h.maxLevenshteinDistance
		}
		if h.maxDifferenceRatio > 0 {
			ratio := float64(dst) / float64(len(wellKnown))
			return ratio <= h.maxDifferenceRatio
		}
		return false
	}
	findTyposInSlice := func(valueType string, values iter.Seq[string], wellKnownValues []string) []error {
		var errs []error
		for value := range values {
			for _, wellKnownValue := range wellKnownValues {
				if isTypo(value, wellKnownValue) {
					errs = append(errs, fmt.Errorf("%s `%s` has a typo, did you mean : %s?", valueType, value, wellKnownValue))
				}
			}
		}
		return errs
	}
	if len(h.wellKnownAnnotations) > 0 {
		errs = append(errs, findTyposInSlice("annotation", maps.Keys(rule.Annotations), h.wellKnownAnnotations)...)
	}
	if len(h.wellKnownRuleLabels) > 0 {
		errs = append(errs, findTyposInSlice("rule label", maps.Keys(rule.Labels), h.wellKnownRuleLabels)...)
	}
	if len(h.wellKnownSeriesLabels) > 0 {
		usedLabels, err := getExpressionUsedLabels(rule.Expr)
		if err != nil {
			return append(errs, fmt.Errorf("failed to get label matchers used in expression `%s`: %w", rule.Expr, err))
		}
		errs = append(errs, findTyposInSlice("series label", slices.Values(usedLabels), h.wellKnownSeriesLabels)...)
	}
	return errs
}
