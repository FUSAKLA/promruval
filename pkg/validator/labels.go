package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/prometheus/pkg/rulefmt"
	"gopkg.in/yaml.v3"
)

func newHasLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels       []string `yaml:"labels"`
		SearchInExpr bool     `yaml:"searchInExpr"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &hasLabels{labels: params.Labels, searchInExpr: params.SearchInExpr}, nil
}

type hasLabels struct {
	labels       []string
	searchInExpr bool
}

func (h hasLabels) String() string {
	return fmt.Sprintf("has labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h hasLabels) Validate(rule rulefmt.Rule) []error {
	var (
		errs       []error
		err        error
		exprLabels []string
	)

	if h.searchInExpr {
		exprLabels, err = getExpressionUsedLabels(rule.Expr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; !ok {
			foundInExpr := false
			for _, exprLabel := range exprLabels {
				if label == exprLabel {
					foundInExpr = true
				}
			}
			if foundInExpr {
				continue
			}
			errs = append(errs, fmt.Errorf("missing label `%s`", label))
		}
	}
	return errs
}

func newDoesNotHaveLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels       []string `yaml:"labels"`
		searchInExpr bool     `yaml:"searchInExpr"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Labels) == 0 {
		return nil, fmt.Errorf("missing labels")
	}
	return &doesNotHaveLabels{labels: params.Labels, searchInExpr: params.searchInExpr}, nil
}

type doesNotHaveLabels struct {
	labels       []string
	searchInExpr bool
}

func (h doesNotHaveLabels) String() string {
	return fmt.Sprintf("does not have labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h doesNotHaveLabels) Validate(rule rulefmt.Rule) []error {
	var errs []error
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; ok {
			errs = append(errs, fmt.Errorf("has forbidden label `%s`", label))
		}
	}
	if h.searchInExpr {
		usedLabels, err := getExpressionUsedLabels(rule.Expr)
		if err != nil {
			return []error{err}
		}
		for _, l := range usedLabels {
			for _, n := range h.labels {
				if l == n {
					errs = append(errs, fmt.Errorf("forbidden label `%s` used in expression", l))
				}
			}
		}
	}
	return errs
}

func newHasAnyOfLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Labels []string `yaml:"labels"`
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
	return fmt.Sprintf("has any of these labels: `%s`", strings.Join(h.labels, "`,`"))
}

func (h hasAnyOfLabels) Validate(rule rulefmt.Rule) []error {
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; ok {
			return []error{}
		}
	}
	return []error{fmt.Errorf("missing any of these labels `%s`", strings.Join(h.labels, "`,`"))}
}

func newLabelHasAllowedValue(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label               string   `yaml:"label"`
		AllowedValues       []string `yaml:"allowedValues"`
		CommaSeparatedValue bool     `yaml:"commaSeparatedValue"`
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
	return &labelHasAllowedValue{label: params.Label, allowedValues: params.AllowedValues, commaSeparatedValue: params.CommaSeparatedValue}, nil
}

type labelHasAllowedValue struct {
	label               string
	allowedValues       []string
	commaSeparatedValue bool
}

func (h labelHasAllowedValue) String() string {
	text := fmt.Sprintf("has one of the allowed values: `%s`", strings.Join(h.allowedValues, "`,`"))
	if h.commaSeparatedValue {
		text = "split by comma " + text
	}
	return fmt.Sprintf("label `%s` %s", h.label, text)
}

func (h labelHasAllowedValue) Validate(rule rulefmt.Rule) []error {
	ruleValue, ok := rule.Labels[h.label]
	if !ok {
		return []error{}
	}
	valuesToCheck := []string{ruleValue}
	if h.commaSeparatedValue {
		valuesToCheck = strings.Split(ruleValue, ",")
	}
	for _, labelValue := range valuesToCheck {
		for _, allowedValue := range h.allowedValues {
			if allowedValue == labelValue {
				return []error{}
			}
		}
	}
	return []error{fmt.Errorf("label `%s` value `%s` is not one of the allowed values: `%s`", h.label, ruleValue, strings.Join(h.allowedValues, "`,`"))}
}

func newLabelMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label  string `yaml:"label"`
		Regexp string `yaml:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Label == "" {
		return nil, fmt.Errorf("missing lanel name")
	}
	expr, err := regexp.Compile(params.Regexp)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp %s", params.Regexp)
	}
	return &labelMatchesRegexp{label: params.Label, regexp: expr}, nil
}

type labelMatchesRegexp struct {
	label  string
	regexp *regexp.Regexp
}

func (h labelMatchesRegexp) String() string {
	return fmt.Sprintf("label `%s` matches Regexp `%s`", h.label, h.regexp)
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
