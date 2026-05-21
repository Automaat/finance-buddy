package config

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// validationError captures a single Pydantic-shaped field error.
type validationError struct {
	Field string
	Msg   string
}

// validate replicates the field + model validators on
// backend/app/schemas/config.ConfigCreate.
//
// First-error-wins (matches FastAPI/Pydantic behavior for these handlers).
func (r *request) validate() *validationError {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	birth := time.Time(r.BirthDate).UTC().Truncate(24 * time.Hour)
	if !birth.Before(today) {
		return &validationError{"birth_date", "Birth date must be in the past"}
	}
	age := int(float64(today.Sub(birth)/(24*time.Hour)) / 365.25)
	if age < 18 || age > 100 {
		return &validationError{"birth_date", "Age must be between 18 and 100"}
	}

	if r.RetirementAge < 18 || r.RetirementAge > 100 {
		return &validationError{"retirement_age", "Retirement age must be between 18 and 100"}
	}
	if !r.RetirementMonthlySalary.GreaterThan(decimal.Zero) {
		return &validationError{
			"retirement_monthly_salary",
			"Retirement monthly salary must be greater than 0",
		}
	}
	if r.MonthlyExpenses.LessThan(decimal.Zero) {
		return &validationError{"monthly_expenses", "Value must be non-negative"}
	}
	if r.MonthlyMortgagePayment.LessThan(decimal.Zero) {
		return &validationError{"monthly_mortgage_payment", "Value must be non-negative"}
	}

	for _, a := range [...]struct {
		name  string
		value int
	}{
		{"allocation_real_estate", r.AllocationRealEstate},
		{"allocation_stocks", r.AllocationStocks},
		{"allocation_bonds", r.AllocationBonds},
		{"allocation_gold", r.AllocationGold},
		{"allocation_commodities", r.AllocationCommodities},
	} {
		if a.value < 0 || a.value > 100 {
			return &validationError{a.name, "Allocation must be between 0 and 100"}
		}
	}

	marketTotal := r.AllocationStocks + r.AllocationBonds + r.AllocationGold + r.AllocationCommodities
	if marketTotal != 100 {
		return &validationError{
			"allocation_stocks",
			fmt.Sprintf("Market allocations must sum to 100%%, got %d%%", marketTotal),
		}
	}

	if r.RetirementAge <= age {
		return &validationError{
			"retirement_age",
			fmt.Sprintf(
				"Retirement age (%d) must be greater than current age (%d)",
				r.RetirementAge, age,
			),
		}
	}
	return nil
}
