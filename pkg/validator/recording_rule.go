package validator

import (
	"fmt"
	"regexp"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

func newRecordedMetricNameMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Regexp RegexpForbidEmpty `yaml:"regexp"`
		Negate bool              `yaml:"negate"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &recordedMetricNameMatchesRegexp{
		pattern: params.Regexp.Regexp,
		negate:  params.Negate,
	}, nil
}

type recordedMetricNameMatchesRegexp struct {
	pattern *regexp.Regexp
	negate  bool
}

func (h recordedMetricNameMatchesRegexp) String() string {
	return fmt.Sprintf("recorded metric name %s regexp: `%s`", matches(h.negate), h.pattern.String())
}

func (h recordedMetricNameMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	if h.pattern.MatchString(rule.Record) == h.negate {
		errs = append(errs, fmt.Errorf("recorded metric name `%s` %s regexp `%s`", rule.Record, matches(!h.negate), h.pattern.String()))
	}
	return errs
}

func newRecordedMetricNameDoesNotMatchRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Regexp RegexpForbidEmpty `yaml:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &recordedMetricNameMatchesRegexp{
		pattern: params.Regexp.Regexp,
		negate:  true,
	}, nil
}
