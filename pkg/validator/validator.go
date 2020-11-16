package validator

import (
	"fmt"
	"github.com/prometheus/prometheus/pkg/rulefmt"
)

type Validator interface {
	fmt.Stringer
	Validate(rule rulefmt.Rule) []error
}
