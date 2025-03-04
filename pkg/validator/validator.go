package validator

import (
	"fmt"
	"regexp"

	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
)

type Validator interface {
	fmt.Stringer
	Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error
}

const (
	matchAnythingRegexp = ".*"
)

type RegexpForbidEmpty struct {
	Regexp *regexp.Regexp
}

func (r *RegexpForbidEmpty) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("regexp cannot be empty")
	}
	re, err := compileAnchoredRegexp(value)
	if err != nil {
		return err
	}
	*r = RegexpForbidEmpty{Regexp: re}
	return nil
}

type RegexpEmptyDefault struct {
	Regexp *regexp.Regexp
}

func (r *RegexpEmptyDefault) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	re, err := compileAnchoredRegexp(value)
	if err != nil {
		return err
	}
	*r = RegexpEmptyDefault{Regexp: re}
	return nil
}

type RegexpWildcardDefault struct {
	Regexp *regexp.Regexp
}

func (r *RegexpWildcardDefault) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	if value == "" {
		value = matchAnythingRegexp
	}
	re, err := compileAnchoredRegexp(value)
	if err != nil {
		return err
	}
	*r = RegexpWildcardDefault{Regexp: re}
	return nil
}

func anchorRegexp(regexpString string) string {
	return fmt.Sprintf("^%s$", regexpString)
}

func compileAnchoredRegexp(regexpString string) (*regexp.Regexp, error) {
	return regexp.Compile(anchorRegexp(regexpString))
}
