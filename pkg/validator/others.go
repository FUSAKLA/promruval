package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type SourceTenantMetrics struct {
	Regexp      string `yaml:"regexp"`
	Description string `yaml:"description"`
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
			compiledRegexp, err := regexp.Compile("^" + metric.Regexp + "$")
			if err != nil {
				return nil, fmt.Errorf("invalid metric name regexp: %s", metric.Regexp)
			}
			m[i] = tenantMetrics{
				regexp:      compiledRegexp,
				description: metric.Description,
			}
		}
		validator.sourceTenants[tenant] = m
	}
	return &validator, nil
}

type tenantMetrics struct {
	regexp      *regexp.Regexp
	description string
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
