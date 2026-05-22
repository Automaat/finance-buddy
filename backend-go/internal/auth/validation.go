package auth

import (
	"strings"

	"github.com/shopspring/decimal"
)

const minPasswordLen = 8

// PPK rate defaults + bounds, mirroring the persona schema.
var (
	defaultPPKEmployeeRate = decimal.NewFromFloat(2.0)
	defaultPPKEmployerRate = decimal.NewFromFloat(1.5)
	ppkMin                 = decimal.NewFromFloat(0.5)
	ppkMax                 = decimal.NewFromFloat(4.0)
)

// validateUsername trims and bounds the username. The second return value is
// a non-empty message when validation fails.
func validateUsername(raw string) (string, string) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", "Username cannot be empty"
	}
	if len(u) > 100 {
		return "", "Username too long (max 100 characters)"
	}
	return u, ""
}

// validatePassword returns a non-empty message when the password is too short.
func validatePassword(raw string) string {
	if len(raw) < minPasswordLen {
		return "Password must be at least 8 characters"
	}
	return ""
}

// optionalName trims an optional name/surname; an empty string becomes nil.
// field names the value for the error message.
func optionalName(raw *string, field string) (*string, string) {
	if raw == nil {
		return nil, ""
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, ""
	}
	if len(trimmed) > 100 {
		return nil, field + " too long (max 100 characters)"
	}
	return &trimmed, ""
}

// validatePPK enforces 0.5 <= rate <= 4.0 when a rate is supplied.
func validatePPK(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	if d.LessThan(ppkMin) || d.GreaterThan(ppkMax) {
		return "PPK rate must be between 0.5 and 4.0"
	}
	return ""
}

// ppkOrDefault returns the supplied rate, or fallback when none was given.
func ppkOrDefault(d *decimal.Decimal, fallback decimal.Decimal) *decimal.Decimal {
	if d != nil {
		return d
	}
	return &fallback
}
