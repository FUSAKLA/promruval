package validator

import (
	"fmt"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/prometheus/prometheus/model/rulefmt"
)

type Validator interface {
	fmt.Stringer
	Validate(rule rulefmt.Rule, prometheusClient *prometheus.Client) []error
}
