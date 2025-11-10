package validator

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/template"
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

func newKeepFiringForIsNotLongerThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &keepFiringForIsNotLongerThan{limit: params.Limit}, nil
}

type keepFiringForIsNotLongerThan struct {
	limit model.Duration
}

func (h keepFiringForIsNotLongerThan) String() string {
	return fmt.Sprintf("`keep_firing_for` is not longer than `%s`", h.limit)
}

func (h keepFiringForIsNotLongerThan) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if rule.KeepFiringFor > h.limit {
		return []error{fmt.Errorf("alert has `keep_firing_for: %s` which is longer than the specified limit of %s", rule.KeepFiringFor, h.limit)}
	}
	return nil
}

func newValidateLabelTemplates(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &validateLabelTemplates{}, nil
}

type validateLabelTemplates struct{}

func (h validateLabelTemplates) String() string {
	return "labels are valid templates"
}

func (h validateLabelTemplates) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	data := template.AlertTemplateData(map[string]string{}, map[string]string{}, "", promql.Sample{})
	defs := []string{
		"{{$labels := .Labels}}",
		"{{$externalLabels := .ExternalLabels}}",
		"{{$externalURL := .ExternalURL}}",
		"{{$value := .Value}}",
	}
	for k, v := range rule.Labels {
		t := template.NewTemplateExpander(context.TODO(), strings.Join(append(defs, v), ""), k, data, model.Now(), func(_ context.Context, _ string, _ time.Time) (promql.Vector, error) { return nil, nil }, &url.URL{}, []string{})
		if _, err := t.Expand(); err != nil && !strings.Contains(err.Error(), "error executing template") {
			errs = append(errs, fmt.Errorf("invalid template of label %s: %w", k, err))
		}
	}
	return errs
}

func newAlertNameMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Regexp   RegexpForbidEmpty `yaml:"regexp"`
		Negative bool              `yaml:"negative"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &alertNameMatchesRegexp{
		pattern:  params.Regexp.Regexp,
		negative: params.Negative,
	}, nil
}

type alertNameMatchesRegexp struct {
	pattern  *regexp.Regexp
	negative bool
}

func (h alertNameMatchesRegexp) String() string {
	return fmt.Sprintf("Alert name %s regexp: `%s`", matches(h.negative), h.pattern.String())
}

func (h alertNameMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	if h.pattern.MatchString(rule.Alert) == h.negative {
		errs = append(errs, fmt.Errorf("alert name `%s` %s regexp `%s`", rule.Alert, matches(!h.negative), h.pattern.String()))
	}
	return errs
}
