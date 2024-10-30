package validator

import "fmt"

func anchorRegexp(regexp string) string {
	return fmt.Sprintf("^%s$", regexp)
}
