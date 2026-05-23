package config

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func validRequest() *request {
	birth := time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC)
	return &request{
		BirthDate:               isoDate(birth),
		RetirementAge:           65,
		RetirementMonthlySalary: decimal.NewFromInt(5000),
		AllocationRealEstate:    20,
		AllocationStocks:        50,
		AllocationBonds:         30,
		AllocationGold:          10,
		AllocationCommodities:   10,
		MonthlyExpenses:         decimal.NewFromInt(3000),
		MonthlyMortgagePayment:  decimal.NewFromInt(1500),
	}
}

func TestValidateOK(t *testing.T) {
	r := validRequest()
	if err := r.validate(); err != nil {
		t.Fatalf("expected valid, got %+v", err)
	}
}

func TestValidateFutureBirthDate(t *testing.T) {
	r := validRequest()
	r.BirthDate = isoDate(time.Now().UTC().AddDate(1, 0, 0))
	if err := r.validate(); err == nil || err.Field != "birth_date" {
		t.Fatalf("expected birth_date error, got %+v", err)
	}
}

func TestValidateAgeUnder18(t *testing.T) {
	r := validRequest()
	r.BirthDate = isoDate(time.Now().UTC().AddDate(-10, 0, 0))
	if err := r.validate(); err == nil || err.Field != "birth_date" {
		t.Fatalf("expected birth_date error, got %+v", err)
	}
}

func TestValidateAgeOver100(t *testing.T) {
	r := validRequest()
	r.BirthDate = isoDate(time.Now().UTC().AddDate(-110, 0, 0))
	if err := r.validate(); err == nil || err.Field != "birth_date" {
		t.Fatalf("expected birth_date error, got %+v", err)
	}
}

func TestValidateRetirementAgeOutOfRange(t *testing.T) {
	r := validRequest()
	r.RetirementAge = 10
	if err := r.validate(); err == nil || err.Field != "retirement_age" {
		t.Fatalf("expected retirement_age error, got %+v", err)
	}
}

func TestValidateRetirementSalaryNonPositive(t *testing.T) {
	r := validRequest()
	r.RetirementMonthlySalary = decimal.Zero
	if err := r.validate(); err == nil || err.Field != "retirement_monthly_salary" {
		t.Fatalf("expected retirement_monthly_salary error, got %+v", err)
	}
}

func TestValidateNegativeExpenses(t *testing.T) {
	r := validRequest()
	r.MonthlyExpenses = decimal.NewFromInt(-1)
	if err := r.validate(); err == nil || err.Field != "monthly_expenses" {
		t.Fatalf("expected monthly_expenses error, got %+v", err)
	}
}

func TestValidateNegativeMortgage(t *testing.T) {
	r := validRequest()
	r.MonthlyMortgagePayment = decimal.NewFromInt(-1)
	if err := r.validate(); err == nil || err.Field != "monthly_mortgage_payment" {
		t.Fatalf("expected monthly_mortgage_payment error, got %+v", err)
	}
}

func TestValidateAllocationOutOfRange(t *testing.T) {
	r := validRequest()
	r.AllocationStocks = 150
	if err := r.validate(); err == nil || err.Field != "allocation_stocks" {
		t.Fatalf("expected allocation_stocks error, got %+v", err)
	}
}

func TestValidateMarketAllocationDoesNotSumTo100(t *testing.T) {
	r := validRequest()
	r.AllocationStocks = 40
	r.AllocationBonds = 30
	r.AllocationGold = 10
	r.AllocationCommodities = 10
	if err := r.validate(); err == nil || err.Field != "allocation_stocks" {
		t.Fatalf("expected allocation sum error, got %+v", err)
	}
}

func TestValidateRetirementAgeBeforeCurrent(t *testing.T) {
	r := validRequest()
	now := time.Now().UTC()
	r.BirthDate = isoDate(now.AddDate(-50, 0, 0))
	r.RetirementAge = 30
	if err := r.validate(); err == nil || err.Field != "retirement_age" {
		t.Fatalf("expected retirement_age (vs current age) error, got %+v", err)
	}
}
