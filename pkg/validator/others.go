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

func newHasSourceTenantsForMetrics(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		SourceTenants map[string]string `yaml:"sourceTenants"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.SourceTenants == nil || len(params.SourceTenants) == 0 {
		return nil, fmt.Errorf("sourceTenants metrics mapping needs to be set")
	}
	validator := hasSourceTenantsForMetrics{sourceTenants: map[string]*regexp.Regexp{}}
	for tenant, metricsRegexp := range params.SourceTenants {
		compiledRegexp, err := regexp.Compile("^" + metricsRegexp + "$")
		if err != nil {
			return nil, fmt.Errorf("invalid metric name regexp: %s", metricsRegexp)
		}
		validator.sourceTenants[tenant] = compiledRegexp
	}
	return &validator, nil
}

type hasSourceTenantsForMetrics struct {
	sourceTenants map[string]*regexp.Regexp
}

func (h hasSourceTenantsForMetrics) String() string {
	tenantStrings := []string{}
	for tenant, metricsRegexp := range h.sourceTenants {
		tenantStrings = append(tenantStrings, fmt.Sprintf("`%s`:`%s`", tenant, metricsRegexp.String()))
	}
	return fmt.Sprintf("rule group, the rule belongs to, has the required `source_tenants` configured, according to the mapping of metric names to tenants: %s", strings.Join(tenantStrings, ", "))
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
			if metricsRegexp.MatchString(usedMetric.Name) && !slices.Contains(group.SourceTenants, tenant) {
				errs = append(errs, fmt.Errorf("missing source_tenant `%s` for metric `%s`", tenant, usedMetric))
			}
		}
	}
	return errs
}
