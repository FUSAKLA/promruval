package validator

import (
	"fmt"
	"strings"
	"time"

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

func newHasAllowedEvaluationInterval(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		MinimumEvaluationInterval time.Duration `yaml:"minimum"`
		MaximumEvaluationInterval time.Duration `yaml:"maximum"`
		MustBeSet                 bool          `yaml:"intervalMustBeSet"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.MinimumEvaluationInterval > params.MaximumEvaluationInterval {
		return nil, fmt.Errorf("minimum is greater than maximum")
	}
	if params.MaximumEvaluationInterval == 0 && params.MinimumEvaluationInterval == 0 {
		return nil, fmt.Errorf("at least one of the `minimum` or `maximum` must be set")
	}
	if params.MaximumEvaluationInterval == 0 {
		params.MaximumEvaluationInterval = time.Duration(1<<63 - 1)

	}
	return &hasAllowedEvaluationInterval{minimum: params.MinimumEvaluationInterval, maximum: params.MaximumEvaluationInterval, mustBeSet: params.MustBeSet}, nil
}

type hasAllowedEvaluationInterval struct {
	minimum   time.Duration
	maximum   time.Duration
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
	if h.minimum != 0 && time.Duration(group.Interval) < h.minimum {
		return []error{fmt.Errorf("evaluation interval %s is less than `%s`", group.Interval, h.minimum)}
	}
	if h.maximum != 0 && time.Duration(group.Interval) > h.maximum {
		return []error{fmt.Errorf("evaluation interval %s is greater than `%s`", group.Interval, h.maximum)}
	}
	return []error{}
}
