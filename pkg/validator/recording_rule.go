package validator

import (
	"fmt"
	"regexp"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

func newRecordedMetricNameMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Regexp string `yaml:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Regexp == "" {
		return nil, fmt.Errorf("missing regexp")
	}
	r, err := regexp.Compile(params.Regexp)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp %s: %w", params.Regexp, err)
	}
	return &recordedMetricNameMatchesRegexp{
		pattern: r,
	}, nil
}

type recordedMetricNameMatchesRegexp struct {
	pattern *regexp.Regexp
}

func (h recordedMetricNameMatchesRegexp) String() string {
	return fmt.Sprintf("Recorded metric name matches regexp: %s", h.pattern.String())
}

func (h recordedMetricNameMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	if !h.pattern.MatchString(rule.Record) {
		errs = append(errs, fmt.Errorf("recorded metric name %s does not match pattern %s", rule.Alert, h.pattern.String()))
	}
	return errs
}
