package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
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
		SourceTenants map[string]SourceTenantMetrics `yaml:"sourceTenants"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.SourceTenants == nil || len(params.SourceTenants) == 0 {
		return nil, fmt.Errorf("sourceTenants metrics mapping needs to be set")
	}
	validator := hasSourceTenantsForMetrics{sourceTenants: map[string]tenantMetrics{}}
	for tenant, metrics := range params.SourceTenants {
		compiledRegexp, err := regexp.Compile("^" + metrics.Regexp + "$")
		if err != nil {
			return nil, fmt.Errorf("invalid metric name regexp: %s", metrics.Regexp)
		}
		validator.sourceTenants[tenant] = tenantMetrics{
			regexp:      compiledRegexp,
			description: metrics.Description,
		}
	}
	return &validator, nil
}

type tenantMetrics struct {
	regexp      *regexp.Regexp
	description string
}

type hasSourceTenantsForMetrics struct {
	sourceTenants map[string]tenantMetrics
}

func (h hasSourceTenantsForMetrics) String() string {
	tenantStrings := []string{}
	for tenant, metricsRegexp := range h.sourceTenants {
		tenantStrings = append(tenantStrings, fmt.Sprintf("`%s`:   `%s` (%s)", tenant, metricsRegexp.regexp.String(), metricsRegexp.description))
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
		for tenant, metricsRegexp := range h.sourceTenants {
			if metricsRegexp.regexp.MatchString(usedMetric.Name) && !slices.Contains(group.SourceTenants, tenant) {
				errs = append(errs, fmt.Errorf("rule uses metric `%s` of the tenant `%s` tenant, you should set the tenant in the groups source_tenants settings", tenant, usedMetric.Name))
			}
		}
	}
	return errs
}
