package config

import (
	"fmt"
	"slices"
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
	if r.WithdrawalRate != nil && !isAllowedWithdrawalRate(*r.WithdrawalRate) {
		return &validationError{
			"withdrawal_rate",
			"Withdrawal rate must be 0.03, 0.035, or 0.04",
		}
	}
	if r.CoastFIRETargetAge != nil {
		ta := *r.CoastFIRETargetAge
		if ta < 18 || ta > 100 {
			return &validationError{
				"coast_fire_target_age",
				"Coast FIRE target age must be between 18 and 100",
			}
		}
		if ta <= age {
			return &validationError{
				"coast_fire_target_age",
				fmt.Sprintf(
					"Coast FIRE target age (%d) must be greater than current age (%d)",
					ta, age,
				),
			}
		}
	}
	if r.ExpectedReturnRate != nil {
		if r.ExpectedReturnRate.LessThan(minExpectedReturnRate) ||
			r.ExpectedReturnRate.GreaterThan(maxExpectedReturnRate) {
			return &validationError{
				"expected_return_rate",
				"Expected return rate must be between 0.01 and 0.20",
			}
		}
	}
	if r.BaristaMonthlyIncome != nil && r.BaristaMonthlyIncome.LessThan(decimal.Zero) {
		return &validationError{"barista_monthly_income", "Value must be non-negative"}
	}
	return nil
}

// minExpectedReturnRate / maxExpectedReturnRate bracket the Coast FIRE
// projection's expected_return_rate input: 1%–20% covers everything from
// conservative bond-heavy portfolios to optimistic stock returns. Values
// outside this band almost always point at a unit error (e.g. typing 7
// instead of 0.07).
var (
	minExpectedReturnRate = decimal.RequireFromString("0.01")
	maxExpectedReturnRate = decimal.RequireFromString("0.20")
)

// allowedWithdrawalRates are the three FIRE-research-backed safe-withdrawal
// rates exposed in /settings. RequireFromString — not NewFromFloat — so the
// constants stay exact: NewFromFloat(0.035) round-trips through a binary
// float and can lose precision, which would make Decimal.Equal miss a
// JSON "0.035" parsed exactly via shopspring's string path.
var allowedWithdrawalRates = []decimal.Decimal{
	decimal.RequireFromString("0.03"),
	decimal.RequireFromString("0.035"),
	decimal.RequireFromString("0.04"),
}

func isAllowedWithdrawalRate(v decimal.Decimal) bool {
	return slices.ContainsFunc(allowedWithdrawalRates, v.Equal)
}
