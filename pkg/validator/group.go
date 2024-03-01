package validator

import (
	"fmt"
	"strings"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/common/model"
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

func newHasAllowedEvaluationInterval(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Minimum   model.Duration `yaml:"minimum"`
		Maximum   model.Duration `yaml:"maximum"`
		MustBeSet bool           `yaml:"intervalMustBeSet"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Maximum == 0 {
		params.Maximum = model.Duration(1<<63 - 1)
	}
	if params.Minimum > params.Maximum {
		return nil, fmt.Errorf("minimum is greater than maximum")
	}
	if params.Maximum == 0 && params.Minimum == 0 {
		return nil, fmt.Errorf("at least one of the `minimum` or `maximum` must be set")
	}
	return &hasAllowedEvaluationInterval{minimum: params.Minimum, maximum: params.Maximum, mustBeSet: params.MustBeSet}, nil
}

type hasAllowedEvaluationInterval struct {
	minimum   model.Duration
	maximum   model.Duration
	mustBeSet bool
}

func (h hasAllowedEvaluationInterval) String() string {
	text := fmt.Sprintf("evaluation interval is between `%s` and `%s`", h.minimum, h.maximum)
	if h.mustBeSet {
		text += " and must be set"
	} else {
		text += " if set"
	}
	return text
}

func (h hasAllowedEvaluationInterval) Validate(group unmarshaler.RuleGroup, _ rulefmt.Rule, _ *prometheus.Client) []error {
	if group.Interval == 0 {
		if h.mustBeSet {
			return []error{fmt.Errorf("evaluation interval must be set")}
		}
		return []error{}
	}
	if h.minimum != 0 && group.Interval < h.minimum {
		return []error{fmt.Errorf("evaluation interval %s is less than `%s`", group.Interval, h.minimum)}
	}
	if h.maximum != 0 && group.Interval > h.maximum {
		return []error{fmt.Errorf("evaluation interval %s is greater than `%s`", group.Interval, h.maximum)}
	}
	return []error{}
}

func newHasValidPartialStrategy(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		MustBeSet bool `yaml:"mustBeSet"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &hasValidPartialStrategy{mustBeSet: params.MustBeSet}, nil
}

type hasValidPartialStrategy struct {
	mustBeSet bool
}

func (h hasValidPartialStrategy) String() string {
	text := "has valid partial_response_strategy (one of `warn` or `abort`)"
	if h.mustBeSet {
		text += " and must be set"
	} else {
		text += " if set"
	}
	return text
}

func (h hasValidPartialStrategy) Validate(group unmarshaler.RuleGroup, _ rulefmt.Rule, _ *prometheus.Client) []error {
	if group.PartialResponseStrategy == "" {
		if h.mustBeSet {
			return []error{fmt.Errorf("partial_response_strategy must be set")}
		}
		return []error{}
	}
	if group.PartialResponseStrategy != "warn" && group.PartialResponseStrategy != "abort" {
		return []error{fmt.Errorf("invalid partial_response_strategy `%s`, valid options are `warn` and `abort`", group.PartialResponseStrategy)}
	}
	return []error{}
}

func newMaxRulesPerGroup(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit int `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &maxRulesPerGroup{limit: params.Limit}, nil
}

type maxRulesPerGroup struct {
	limit int
}

func (h maxRulesPerGroup) String() string {
	return fmt.Sprintf("has at most %d rules", h.limit)
}

func (h maxRulesPerGroup) Validate(group unmarshaler.RuleGroup, _ rulefmt.Rule, _ *prometheus.Client) []error {
	if len(group.Rules) > h.limit {
		return []error{fmt.Errorf("group has %d rules, maximum is %d", len(group.Rules), h.limit)}
	}
	return []error{}
}

func newHasAllowedLimit(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit int `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &hasAllowedLimit{limit: params.Limit}, nil
}

type hasAllowedLimit struct {
	limit     int
	mustBeSet bool
}

func (h hasAllowedLimit) String() string {
	return fmt.Sprintf("does not have higher `limit` configured then %d", h.limit)
}

func (h hasAllowedLimit) Validate(group unmarshaler.RuleGroup, _ rulefmt.Rule, _ *prometheus.Client) []error {
	if group.Limit > h.limit {
		return []error{fmt.Errorf("group has limit %d, allowed maximum is %d", group.Limit, h.limit)}
	} else if group.Limit == 0 {
		return []error{fmt.Errorf("limit must be set, the default value 0 means it is unlimited and maximum allowed limit is %d", h.limit)}
	}
	return []error{}
}
