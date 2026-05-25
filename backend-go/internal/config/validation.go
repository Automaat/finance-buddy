package config

import (
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// validate replicates the field + model validators on
// backend/app/schemas/config.ConfigCreate.
//
// First-error-wins (matches FastAPI/Pydantic behavior for these handlers).
func (r *request) validate() *httputil.ValidationError {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	birth := time.Time(r.BirthDate).UTC().Truncate(24 * time.Hour)
	if !birth.Before(today) {
		return &httputil.ValidationError{Field: "birth_date", Msg: "Birth date must be in the past"}
	}
	age := int(float64(today.Sub(birth)/(24*time.Hour)) / 365.25)
	if age < 18 || age > 100 {
		return &httputil.ValidationError{Field: "birth_date", Msg: "Age must be between 18 and 100"}
	}

	if r.RetirementAge < 18 || r.RetirementAge > 100 {
		return &httputil.ValidationError{Field: "retirement_age", Msg: "Retirement age must be between 18 and 100"}
	}
	if !r.RetirementMonthlySalary.GreaterThan(decimal.Zero) {
		return &httputil.ValidationError{Field: "retirement_monthly_salary", Msg: "Retirement monthly salary must be greater than 0"}
	}
	if r.MonthlyExpenses.LessThan(decimal.Zero) {
		return &httputil.ValidationError{Field: "monthly_expenses", Msg: "Value must be non-negative"}
	}
	if r.MonthlyMortgagePayment.LessThan(decimal.Zero) {
		return &httputil.ValidationError{Field: "monthly_mortgage_payment", Msg: "Value must be non-negative"}
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
			return &httputil.ValidationError{Field: a.name, Msg: "Allocation must be between 0 and 100"}
		}
	}

	marketTotal := r.AllocationStocks + r.AllocationBonds + r.AllocationGold + r.AllocationCommodities
	if marketTotal != 100 {
		return &httputil.ValidationError{
			Field: "allocation_stocks",
			Msg:   fmt.Sprintf("Market allocations must sum to 100%%, got %d%%", marketTotal),
		}
	}

	if r.RetirementAge <= age {
		return &httputil.ValidationError{
			Field: "retirement_age",
			Msg: fmt.Sprintf(
				"Retirement age (%d) must be greater than current age (%d)",
				r.RetirementAge, age,
			),
		}
	}
	if r.WithdrawalRate != nil && !isAllowedWithdrawalRate(*r.WithdrawalRate) {
		return &httputil.ValidationError{Field: "withdrawal_rate", Msg: "Withdrawal rate must be 0.03, 0.035, or 0.04"}
	}
	if r.CoastFIRETargetAge != nil {
		ta := *r.CoastFIRETargetAge
		if ta < 18 || ta > 100 {
			return &httputil.ValidationError{Field: "coast_fire_target_age", Msg: "Coast FIRE target age must be between 18 and 100"}
		}
		if ta <= age {
			return &httputil.ValidationError{
				Field: "coast_fire_target_age",
				Msg: fmt.Sprintf(
					"Coast FIRE target age (%d) must be greater than current age (%d)",
					ta, age,
				),
			}
		}
	}
	if r.ExpectedReturnRate != nil {
		if r.ExpectedReturnRate.LessThan(minExpectedReturnRate) ||
			r.ExpectedReturnRate.GreaterThan(maxExpectedReturnRate) {
			return &httputil.ValidationError{Field: "expected_return_rate", Msg: "Expected return rate must be between 0.01 and 0.20"}
		}
	}
	if vErr := requireNonNegative(r); vErr != nil {
		return vErr
	}
	return nil
}

// requireNonNegative bundles the simple "field >= 0 when present" checks for
// the FIRE-band + Barista inputs so they don't push the main validator past
// the funlen threshold.
func requireNonNegative(r *request) *httputil.ValidationError {
	checks := []struct {
		name string
		val  *decimal.Decimal
	}{
		{"barista_monthly_income", r.BaristaMonthlyIncome},
		{"lean_monthly_expenses", r.LeanMonthlyExpenses},
		{"fat_monthly_expenses", r.FatMonthlyExpenses},
		{"monthly_savings", r.MonthlySavings},
	}
	for _, c := range checks {
		if c.val != nil && c.val.LessThan(decimal.Zero) {
			return &httputil.ValidationError{Field: c.name, Msg: "Value must be non-negative"}
		}
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
