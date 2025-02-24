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

func anchorRegexp(regexpString string) string {
	return fmt.Sprintf("^%s$", regexpString)
}

func compileAnchoredRegexpWithDefault(regexpString, defaultValue string) (*regexp.Regexp, error) {
	if regexpString == "" {
		return regexp.Compile(anchorRegexp(defaultValue))
	}
	return regexp.Compile(anchorRegexp(regexpString))
}

func compileAnchoredRegexp(regexpString string) (*regexp.Regexp, error) {
	return compileAnchoredRegexpWithDefault(regexpString, "")
}
