package validator

import (
	"fmt"
	"strings"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func newHasAllowedSourceTenants(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		AllowedSourceTenants []string `yaml:"allowedSourceTenants"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &hasAllowedSourceTenants{allowedSourceTenants: params.AllowedSourceTenants}, nil
}

type hasAllowedSourceTenants struct {
	allowedSourceTenants []string
}

func (h hasAllowedSourceTenants) String() string {
	return fmt.Sprintf("does not have other `source_tenants` than: `%s`", strings.Join(h.allowedSourceTenants, "`, `"))
}

func (h hasAllowedSourceTenants) Validate(group unmarshaler.RuleGroup, _ rulefmt.Rule, _ *prometheus.Client) []error {
	var invalidTenants []string
	for _, tenant := range group.SourceTenants {
		if !slices.Contains(h.allowedSourceTenants, tenant) {
			invalidTenants = append(invalidTenants, tenant)
		}
	}
	if len(invalidTenants) == 0 {
		return []error{}
	}
	return []error{fmt.Errorf("group has invalid source_tenants: `%s`", strings.Join(invalidTenants, "`,`"))}
}
