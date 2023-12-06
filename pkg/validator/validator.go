package validator

import (
	"fmt"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/prometheus/model/rulefmt"
)

type Validator interface {
	fmt.Stringer
	Validate(group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error
}
