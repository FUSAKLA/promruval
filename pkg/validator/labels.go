package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
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

func (h hasLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
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

func (h doesNotHaveLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
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

func (h hasAnyOfLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	for _, label := range h.labels {
		if _, ok := rule.Labels[label]; ok {
			return []error{}
		}
	}
	return []error{fmt.Errorf("missing any of these labels `%s`", strings.Join(h.labels, "`,`"))}
}

func newLabelHasAllowedValue(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label                 string   `yaml:"label"`
		AllowedValues         []string `yaml:"allowedValues"`
		CommaSeparatedValue   bool     `yaml:"commaSeparatedValue"`
		IgnoreTemplatedValues bool     `yaml:"ignoreTemplatedValues"`
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
	return &labelHasAllowedValue{label: params.Label, allowedValues: params.AllowedValues, commaSeparatedValue: params.CommaSeparatedValue, ignoreTemplatedValues: params.IgnoreTemplatedValues}, nil
}

type labelHasAllowedValue struct {
	label                 string
	allowedValues         []string
	commaSeparatedValue   bool
	ignoreTemplatedValues bool
}

func (h labelHasAllowedValue) String() string {
	text := fmt.Sprintf("has one of the allowed values: `%s`", strings.Join(h.allowedValues, "`,`"))
	if h.commaSeparatedValue {
		text = "split by comma " + text
	}
	text = fmt.Sprintf("label `%s` %s", h.label, text)
	if h.ignoreTemplatedValues {
		text += " (templated values are ignored)"
	}
	return text
}

func (h labelHasAllowedValue) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	ruleValue, ok := rule.Labels[h.label]
	if !ok {
		return []error{}
	}
	if h.ignoreTemplatedValues && strings.Contains(ruleValue, "{{") {
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
		Label  string             `yaml:"label"`
		Regexp RegexpEmptyDefault `yaml:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Label == "" {
		return nil, fmt.Errorf("missing label name")
	}
	expr, err := compileAnchoredRegexp(params.Regexp)
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
	return fmt.Sprintf("label `%s` matches regexp `%s`", h.label, h.regexp)
}

func (h labelMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	value, ok := rule.Labels[h.label]
	if !ok {
		return []error{}
	}
	if !h.regexp.MatchString(value) {
		return []error{fmt.Errorf("label `%s` does not match the regular expression `%s`", h.label, h.regexp)}
	}
	return []error{}
}

func newNonEmptyLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &nonEmptyLabels{}, nil
}

type nonEmptyLabels struct{}

func (h nonEmptyLabels) String() string {
	return "labels does not have empty values"
}

func (h nonEmptyLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for k, v := range rule.Labels {
		if v == "" {
			errs = append(errs, fmt.Errorf("label `%s` has empty value, has no effect", k))
		}
	}
	return errs
}

func newExclusiveLabels(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Label1      string `yaml:"firstLabel"`
		Label1Value string `yaml:"firstLabelValue"`
		Label2      string `yaml:"secondLabel"`
		Label2Value string `yaml:"secondLabelValue"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Label1 == "" {
		return nil, fmt.Errorf("missing label1 name")
	}
	if params.Label2 == "" {
		return nil, fmt.Errorf("missing label2 name")
	}
	return &exclusiveLabels{label1: params.Label1, label1Value: params.Label1Value, label2: params.Label2, label2Value: params.Label2Value}, nil
}

type exclusiveLabels struct {
	label1      string
	label1Value string
	label2      string
	label2Value string
}

func (h exclusiveLabels) String() string {
	text := fmt.Sprintf("if rule has label `%s` ", h.label1)
	if h.label1Value != "" {
		text += fmt.Sprintf("with value `%s` ", h.label1Value)
	}
	text += fmt.Sprintf(", it cannot have label `%s`", h.label2)
	if h.label2Value != "" {
		text += fmt.Sprintf("with value `%s` ", h.label2Value)
	}
	return text
}

func (h exclusiveLabels) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	label1Value, hasLabel1 := rule.Labels[h.label1]
	label2Value, hasLabel2 := rule.Labels[h.label2]
	if !hasLabel1 || (h.label1Value != "" && h.label1Value != label1Value) {
		return []error{}
	}
	errMsg := fmt.Sprintf("if the rule has label `%s`", h.label1)
	if h.label1Value != "" {
		errMsg += fmt.Sprintf(" with value `%s`", h.label1Value)
	}
	if !hasLabel2 {
		return []error{}
	}
	errMsg += fmt.Sprintf(", it cannot have label `%s`", h.label2)
	if h.label2Value == "" {
		return []error{errors.New(errMsg)}
	}
	if h.label2Value != "" && h.label2Value == label2Value {
		errMsg += fmt.Sprintf(" with value `%s`", h.label2Value)
		return []error{errors.New(errMsg)}
	}
	return []error{}
}
