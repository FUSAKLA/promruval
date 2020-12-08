package validator

import (
	"fmt"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func newHasAnnotations(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotations []string `yam:"annotations"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Annotations) == 0 {
		return nil, fmt.Errorf("missing annotations")
	}
	return &hasAnnotations{annotations: params.Annotations}, nil
}

type hasAnnotations struct {
	annotations []string
}

func (h hasAnnotations) String() string {
	return fmt.Sprintf("has all of these annotations: `%s`", strings.Join(h.annotations, "`,`"))
}

func (h hasAnnotations) Validate(rule rulefmt.Rule) []error {
	var errs []error
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; !ok {
			errs = append(errs, fmt.Errorf("missing annotation `%s`", annotation))
		}
	}
	return errs
}

func newDoesNotHaveAnnotations(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotations []string `yam:"annotations"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Annotations) == 0 {
		return nil, fmt.Errorf("missing annotations")
	}
	return &doesNotHaveAnnotations{annotations: params.Annotations}, nil
}

type doesNotHaveAnnotations struct {
	annotations []string
}

func (h doesNotHaveAnnotations) String() string {
	return fmt.Sprintf("does not have annotations: `%s`", strings.Join(h.annotations, "`,`"))
}

func (h doesNotHaveAnnotations) Validate(rule rulefmt.Rule) []error {
	var errs []error
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; ok {
			errs = append(errs, fmt.Errorf("has forbidden annotation `%s`", annotation))
		}
	}
	return errs
}

func newHasAnyOfAnnotations(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotations []string `yaml:"annotations"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if len(params.Annotations) == 0 {
		return nil, fmt.Errorf("missing annotations")
	}
	return &hasAnyOfAnnotations{annotations: params.Annotations}, nil
}

type hasAnyOfAnnotations struct {
	annotations []string
}

func (h hasAnyOfAnnotations) String() string {
	return fmt.Sprintf("has any of these annotations: `%s`", strings.Join(h.annotations, "`,`"))
}

func (h hasAnyOfAnnotations) Validate(rule rulefmt.Rule) []error {
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; ok {
			return []error{}
		}
	}
	return []error{fmt.Errorf("missing any of these annotatios `%s`", strings.Join(h.annotations, "`,`"))}
}

func newAnnotationMatchesRegexp(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotation string         `yaml:"annotation"`
		Regexp     *regexp.Regexp `yaml:"regexp"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation")
	}
	return &annotationMatchesRegexp{annotation: params.Annotation, regexp: params.Regexp}, nil
}

type annotationMatchesRegexp struct {
	annotation string
	regexp     *regexp.Regexp
}

func (h annotationMatchesRegexp) String() string {
	return fmt.Sprintf("annotation `%s` matches regexp `%s`", h.annotation, h.regexp)
}

func (h annotationMatchesRegexp) Validate(rule rulefmt.Rule) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if !h.regexp.MatchString(value) {
		return []error{fmt.Errorf("annotation `%s` does not match the regular expression `%s`", h.annotation, h.regexp)}
	}
	return []error{}
}

func newAnnotationHasAllowedValue(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotation          string   `yaml:"annotation"`
		AllowedValues       []string `yaml:"allowedValues"`
		CommaSeparatedValue bool     `yaml:"commaSeparatedValue"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation")
	}
	if len(params.AllowedValues) == 0 {
		return nil, fmt.Errorf("missing allowed values")
	}
	return &annotationHasAllowedValue{annotation: params.Annotation, allowedValues: params.AllowedValues, commaSeparatedValue: params.CommaSeparatedValue}, nil
}

type annotationHasAllowedValue struct {
	annotation          string
	allowedValues       []string
	commaSeparatedValue bool
}

func (h annotationHasAllowedValue) String() string {
	text := fmt.Sprintf("has one of the allowed values: `%s`", strings.Join(h.allowedValues, "`,`"))
	if h.commaSeparatedValue {
		text = "split by comma " + text
	}
	return fmt.Sprintf("annotation `%s` %s", h.annotation, text)
}

func (h annotationHasAllowedValue) Validate(rule rulefmt.Rule) []error {
	ruleValue, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	valuesToCheck := []string{ruleValue}
	if h.commaSeparatedValue {
		valuesToCheck = strings.Split(ruleValue, ",")
	}
	for _, annotationValue := range valuesToCheck {
		for _, allowedValue := range h.allowedValues {
			if allowedValue == annotationValue {
				return []error{}
			}
		}
	}
	return []error{fmt.Errorf("annotation `%s` value `%s` is not one of the allowed values: `%s`", h.annotation, ruleValue, strings.Join(h.allowedValues, "`,`"))}
}

func newAnnotationIsValidURL(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotation string `yaml:"annotation"`
		ResolveURL bool   `yaml:"resolveUrl"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation name")
	}
	return &annotationIsValidURL{annotation: params.Annotation, resolveURL: params.ResolveURL}, nil
}

type annotationIsValidURL struct {
	annotation string
	resolveURL bool
}

func (h annotationIsValidURL) String() string {
	text := fmt.Sprintf("Annotation `%s` is a valid URL", h.annotation)
	if h.resolveURL {
		text += " and does not return HTTP status 404"
	}
	return text
}

func (h annotationIsValidURL) Validate(rule rulefmt.Rule) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if _, err := url.Parse(value); err != nil {
		return []error{fmt.Errorf("annotation `%s` is not valid URL", h.annotation)}
	}
	resp, err := http.Get(value)
	if err != nil {
		return []error{fmt.Errorf("failed to resolve URL `%s` in the `%s` Annotation", value, h.annotation)}
	}
	if resp.StatusCode == http.StatusNotFound {
		return []error{fmt.Errorf("URL `%s` in the `%s` Annotation returns HTTP status code 404 NotFound", value, h.annotation)}
	}
	return []error{}
}

func newAnnotationIsValidPromQL(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Annotation string `yaml:"annotation"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation name")
	}
	return &annotationIsValidPromQL{annotation: params.Annotation}, nil
}

type annotationIsValidPromQL struct {
	annotation string
	resolveURL bool
}

func (h annotationIsValidPromQL) String() string {
	return fmt.Sprintf("Annotation `%s` is a valid PromQL expression", h.annotation)
}

func (h annotationIsValidPromQL) Validate(rule rulefmt.Rule) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if _, err := parser.ParseExpr(value); err != nil {
		return []error{fmt.Errorf("annotation `%s` is not valid PromQL: %s", h.annotation, err)}
	}
	return []error{}
}
