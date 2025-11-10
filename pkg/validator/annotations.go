package validator

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/template"
)

func newHasAnnotations(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotations []string `yaml:"annotations"`
	}{}
	if err := unmarshal(&params); err != nil {
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

func (h hasAnnotations) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; !ok {
			errs = append(errs, fmt.Errorf("missing annotation `%s`", annotation))
		}
	}
	return errs
}

func newDoesNotHaveAnnotations(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotations []string `yaml:"annotations"`
	}{}
	if err := unmarshal(&params); err != nil {
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

func (h doesNotHaveAnnotations) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; ok {
			errs = append(errs, fmt.Errorf("has forbidden annotation `%s`", annotation))
		}
	}
	return errs
}

func newHasAnyOfAnnotations(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotations []string `yaml:"annotations"`
	}{}
	if err := unmarshal(&params); err != nil {
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

func (h hasAnyOfAnnotations) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	for _, annotation := range h.annotations {
		if _, ok := rule.Annotations[annotation]; ok {
			return []error{}
		}
	}
	return []error{fmt.Errorf("missing any of these annotations `%s`", strings.Join(h.annotations, "`,`"))}
}

func newAnnotationMatchesRegexp(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotation string             `yaml:"annotation"`
		Regexp     RegexpEmptyDefault `yaml:"regexp"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation")
	}
	return &annotationMatchesRegexp{annotation: params.Annotation, regexp: params.Regexp.Regexp}, nil
}

type annotationMatchesRegexp struct {
	annotation string
	regexp     *regexp.Regexp
}

func (h annotationMatchesRegexp) String() string {
	return fmt.Sprintf("annotation `%s` matches regexp `%s`", h.annotation, h.regexp)
}

func (h annotationMatchesRegexp) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if !h.regexp.MatchString(value) {
		return []error{fmt.Errorf("annotation `%s` does not match the regular expression `%s`", h.annotation, h.regexp)}
	}
	return []error{}
}

func newAnnotationHasAllowedValue(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotation          string   `yaml:"annotation"`
		AllowedValues       []string `yaml:"allowedValues"`
		CommaSeparatedValue bool     `yaml:"commaSeparatedValue"`
	}{}
	if err := unmarshal(&params); err != nil {
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

func (h annotationHasAllowedValue) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
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

func newAnnotationIsValidURL(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotation string `yaml:"annotation"`
		ResolveURL bool   `yaml:"resolveUrl"`
		AsTemplate bool   `yaml:"asTemplate"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation name")
	}
	return &annotationIsValidURL{annotation: params.Annotation, resolveURL: params.ResolveURL, asTemplate: params.AsTemplate}, nil
}

type annotationIsValidURL struct {
	annotation string
	resolveURL bool
	asTemplate bool
}

func (h annotationIsValidURL) String() string {
	text := fmt.Sprintf("Annotation `%s` is a valid URL", h.annotation)
	if h.asTemplate {
		text += " (when parsed as a Go text template)"
	}
	if h.resolveURL {
		text += " and does not return HTTP status 404"
	}
	return text
}

func (h annotationIsValidURL) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if h.asTemplate {
		var err error
		value, err = newTemplateExpander(value).Expand()
		if err != nil {
			return []error{fmt.Errorf("template expansion of annotation `%s` failed: %w", h.annotation, err)}
		}
	}
	if !govalidator.IsURL(value) {
		return []error{fmt.Errorf("annotation `%s` is not valid URL: `%s`", h.annotation, value)}
	}
	if !h.resolveURL {
		return []error{}
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

func newAnnotationIsValidPromQL(unmarshal func(interface{}) error) (Validator, error) {
	params := struct {
		Annotation string `yaml:"annotation"`
	}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	if params.Annotation == "" {
		return nil, fmt.Errorf("missing annotation name")
	}
	return &annotationIsValidPromQL{annotation: params.Annotation}, nil
}

type annotationIsValidPromQL struct {
	annotation string
}

func (h annotationIsValidPromQL) String() string {
	return fmt.Sprintf("annotation `%s` is a valid PromQL expression", h.annotation)
}

func (h annotationIsValidPromQL) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	value, ok := rule.Annotations[h.annotation]
	if !ok {
		return []error{}
	}
	if _, err := parser.ParseExpr(value); err != nil {
		return []error{fmt.Errorf("annotation `%s` is not valid PromQL: %w", h.annotation, err)}
	}
	return []error{}
}

func newValidateAnnotationTemplates(unmarshal func(interface{}) error) (Validator, error) {
	params := struct{}{}
	if err := unmarshal(&params); err != nil {
		return nil, err
	}
	return &validateAnnotationTemplates{}, nil
}

type validateAnnotationTemplates struct{}

func (h validateAnnotationTemplates) String() string {
	return "annotations are valid templates"
}

func (h validateAnnotationTemplates) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	for k, v := range rule.Annotations {
		if _, err := newTemplateExpander(v).Expand(); err != nil && !strings.Contains(err.Error(), "error executing template") {
			errs = append(errs, fmt.Errorf("invalid template of annotation %s: %w", k, err))
		}
	}
	return errs
}

func newTemplateExpander(text string) *template.Expander {
	data := template.AlertTemplateData(nil, nil, "", promql.Sample{})
	defs := []string{
		"{{$labels := .Labels}}",
		"{{$externalLabels := .ExternalLabels}}",
		"{{$externalURL := .ExternalURL}}",
		"{{$value := .Value}}",
	}
	return template.NewTemplateExpander(
		context.Background(),
		strings.Join(append(defs, text), ""),
		"",
		data,
		model.TimeFromUnix(0),
		func(context.Context, string, time.Time) (promql.Vector, error) { return nil, nil },
		&url.URL{},
		nil,
	)
}
