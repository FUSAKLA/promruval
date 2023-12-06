package validator

import (
	"fmt"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

func newForIsNotLongerThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Limit == model.Duration(0) {
		return nil, fmt.Errorf("missing limit")
	}
	return &forIsNotLongerThan{limit: params.Limit}, nil
}

type forIsNotLongerThan struct {
	limit model.Duration
}

func (h forIsNotLongerThan) String() string {
	return fmt.Sprintf("`for` is not longer than `%s`", h.limit)
}

func (h forIsNotLongerThan) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if rule.For != 0 && rule.For > h.limit {
		return []error{fmt.Errorf("alert has `for: %s` which is longer than the specified limit of %s", rule.For, h.limit)}
	}
	return nil
}
