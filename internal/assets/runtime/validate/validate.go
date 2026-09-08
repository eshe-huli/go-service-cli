// Package validate contains representation checks, not business invariants.
package validate

import "regexp"

var decimal = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// Decimal checks decimal-string syntax only. It does not perform arithmetic,
// enforce currency precision, choose rounding, or decide valid business amounts.
func Decimal(s string) bool { return decimal.MatchString(s) }
