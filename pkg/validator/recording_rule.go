package validator

import (
	"fmt"
	"regexp"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
)

func newRecordedMetricNameMatchesRegexp(unmarshal unmarshalParamsFunc) (Validator, error) {
	params := struct {
		Regexp   RegexpForbidEmpty `yaml:"regexp"`
		Negative bool              `yaml:"negative"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	return &recordedMetricNameMatchesRegexp{
		pattern:  params.Regexp.Regexp,
		negative: params.Negative,
	}, nil
}

type recordedMetricNameMatchesRegexp struct {
	pattern  *regexp.Regexp
	negative bool
}

func (h recordedMetricNameMatchesRegexp) String() string {
	return fmt.Sprintf("recorded metric name %s regexp: `%s`", matches(h.negative), h.pattern.String())
}

func (h recordedMetricNameMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	if h.pattern.MatchString(rule.Record) == h.negative {
		errs = append(errs, fmt.Errorf("recorded metric name `%s` %s regexp `%s`", rule.Record, matches(!h.negative), h.pattern.String()))
	}
	return errs
}

func newRecordedMetricNameDoesNotMatchRegexp(unmarshal unmarshalParamsFunc) (Validator, error) {
	params := struct {
		Regexp RegexpForbidEmpty `yaml:"regexp"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	return &recordedMetricNameMatchesRegexp{
		pattern:  params.Regexp.Regexp,
		negative: true,
	}, nil
}
