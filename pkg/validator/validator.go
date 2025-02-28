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

type RegexpString interface {
	String() string
}

// Custom unmarhsalling for regexps.
type RegexpEmptyDefault string

func (r RegexpEmptyDefault) String() string { return string(r) }

type RegexpWildcardDefault string

func (r RegexpWildcardDefault) String() string { return string(r) }

func (r *RegexpWildcardDefault) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	if value == "" {
		value = matchAnythingRegexp
	}
	*r = RegexpWildcardDefault(value)
	return nil
}

func anchorRegexp(regexpString string) string {
	return fmt.Sprintf("^%s$", regexpString)
}

func compileAnchoredRegexp(regexpString RegexpString) (*regexp.Regexp, error) {
	return regexp.Compile(anchorRegexp(regexpString.String()))
}
