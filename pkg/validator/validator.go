package validator

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/prometheus"
	"github.com/prometheus/prometheus/pkg/rulefmt"
)

type Validator interface {
	fmt.Stringer
	Validate(rule rulefmt.Rule, prometheusClient *prometheus.Client) []error
}
