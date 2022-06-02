package validator

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/rulefmt"
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
	return fmt.Sprintf("alert `for` is not longer than `%s`", h.limit)
}

func (h forIsNotLongerThan) Validate(rule rulefmt.Rule, _ *prometheus.Client) []error {
	if rule.For != 0 && rule.For > h.limit {
		return []error{fmt.Errorf("alert has `for: %s` which is longer than the specified limit of %s", rule.For, h.limit)}
	}
	return nil
}
