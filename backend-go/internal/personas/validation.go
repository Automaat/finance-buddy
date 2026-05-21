package personas

import (
	"strings"

	"github.com/shopspring/decimal"
)

// defaultEmployeeRate / defaultEmployerRate mirror the Python schema defaults.
var (
	defaultEmployeeRate = decimal.NewFromFloat(2.0)
	defaultEmployerRate = decimal.NewFromFloat(1.5)
	ppkMin              = decimal.NewFromFloat(0.5)
	ppkMax              = decimal.NewFromFloat(4.0)
)

// validateName trims whitespace and rejects empty strings — same as the
// Pydantic field_validator on backend/app/schemas/personas.PersonaCreate.name.
func validateName(raw string) (string, *validationError) {
	stripped := strings.TrimSpace(raw)
	if stripped == "" {
		return "", &validationError{Field: "name", Msg: "Name cannot be empty"}
	}
	return stripped, nil
}

// validatePPKRange enforces 0.5 ≤ rate ≤ 4.0 — matches the Pydantic
// field_validator on ppk_employee_rate / ppk_employer_rate.
func validatePPKRange(v decimal.Decimal, field string) *validationError {
	if v.LessThan(ppkMin) || v.GreaterThan(ppkMax) {
		return &validationError{Field: field, Msg: "PPK rate must be between 0.5 and 4.0"}
	}
	return nil
}

// resolveRate returns the supplied rate (validated) or the default when nil.
func resolveRate(v *decimal.Decimal, fallback decimal.Decimal, field string) (decimal.Decimal, *validationError) {
	if v == nil {
		return fallback, nil
	}
	if vErr := validatePPKRange(*v, field); vErr != nil {
		return decimal.Decimal{}, vErr
	}
	return *v, nil
}
