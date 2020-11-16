package validator

import (
	"fmt"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

func newHasLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels []string `yam:"labels"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &hasLabels{labels: params.Labels}, nil
}

type hasLabels struct {
	labels []string
}

func (h hasLabels) String() string {
	return fmt.Sprintf("Has labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h hasLabels) Validate(rule rulefmt.Rule) []error {
	var errs []error
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; !ok {
			errs = append(errs, fmt.Errorf("missing label `%s`", label))
		}
	}
	return errs
}

func newDoesNotHaveLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels []string `yam:"labels"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &doesNotHaveLabels{labels: params.Labels}, nil
}

type doesNotHaveLabels struct {
	labels []string
}

func (h doesNotHaveLabels) String() string {
	return fmt.Sprintf("Does not have labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h doesNotHaveLabels) Validate(rule rulefmt.Rule) []error {
	var errs []error
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; ok {
			errs = append(errs, fmt.Errorf("has forbidden label `%s`", label))
		}
	}
	return errs
}

func newHasAnyOfLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels []string `yam:"labels"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &hasAnyOfLabels{labels: params.Labels}, nil
}

type hasAnyOfLabels struct {
	labels []string
}

func (h hasAnyOfLabels) String() string {
	return fmt.Sprintf("Has any of these labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h hasAnyOfLabels) Validate(rule rulefmt.Rule) []error {
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; ok {
			return []error{}
		}
	}
	return []error{fmt.Errorf("missing any of these annotatios `%s`", strings.Join(h.labels, "`,`"))}
}

func newLabelHasAllowedValue(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label         string `yaml:"label"`
		AllowedValues []string `yaml:"allowedValues"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Label == "" {
		return nil, fmt.Errorf("missing labels")
	}
	if len(params.AllowedValues) == 0 {
		return nil, fmt.Errorf("missing allowedValues")
	}
	return &labelHasAllowedValue{label: params.Label, allowedValues: params.AllowedValues}, nil
}

type labelHasAllowedValue struct {
	label         string
	allowedValues []string
}

func (h labelHasAllowedValue) String() string {
	return fmt.Sprintf("Label `%s` has one of the allowed values: `%s`", h.label, strings.Join(h.allowedValues, "`,`"))
}

func (h labelHasAllowedValue) Validate(rule rulefmt.Rule) []error {
	ruleValue, ok := rule.Labels[h.label]
	if !ok {
		return []error{}
	}
	for _, value := range h.allowedValues {
		if value == ruleValue {
			return []error{}
		}
	}
	return []error{fmt.Errorf("label `%s` value `%s` is not one of the allowed values: `%s`", h.label, ruleValue, strings.Join(h.allowedValues, "`,`"))}
}

func newLabelMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label  string `yam:"label"`
		Regexp *regexp.Regexp `yam:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Label == "" {
		return nil, fmt.Errorf("missing lanel name")
	}
	return &labelMatchesRegexp{label: params.Label, regexp: params.Regexp}, nil
}

type labelMatchesRegexp struct {
	label  string
	regexp *regexp.Regexp
}

func (h labelMatchesRegexp) String() string {
	return fmt.Sprintf("Label `%s` matches Regexp `%s`", h.label, h.regexp)
}

func (h labelMatchesRegexp) Validate(rule rulefmt.Rule) []error {
	value, ok := rule.Labels[h.label]
	if !ok {
		return []error{}
	}
	if !h.regexp.MatchString(value) {
		return []error{fmt.Errorf("label `%s` does not match the regular expression `%s`", h.label, h.regexp)}
	}
	return []error{}
}
