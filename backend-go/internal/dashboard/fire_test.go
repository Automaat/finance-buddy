package dashboard

import (
	"testing"

	"github.com/shopspring/decimal"
)

func newAccountRow(category, accType string, value float64) mergedRow {
	c := category
	t := accType
	id := 1
	return mergedRow{
		AccountID: &id,
		AccType:   &t,
		Category:  &c,
		Value:     value,
	}
}

func TestComputeFIREHappyPath(t *testing.T) {
	t.Parallel()
	rows := []mergedRow{
		newAccountRow("bank", "asset", 30000),
		newAccountRow("saving_account", "asset", 20000),
		newAccountRow("stock", "asset", 100000),
	}
	cfg := AppConfig{
		MonthlyExpenses: decimal.NewFromInt(5000),
		WithdrawalRate:  decimal.NewFromFloat(0.04),
	}

	got := computeFIRE(rows, cfg, 500_000)

	if got.AnnualExpenses == nil || *got.AnnualExpenses != 60_000 {
		t.Fatalf("annual_expenses = %v, want 60000", got.AnnualExpenses)
	}
	if got.FIRENumber == nil || *got.FIRENumber != 1_500_000 {
		t.Fatalf("fire_number = %v, want 1500000", got.FIRENumber)
	}
	if got.FIProgress == nil {
		t.Fatal("fi_progress should be set when net worth > 0")
	}
	// 500_000 / 1_500_000 * 100 ≈ 33.33%
	if !approxEqual(*got.FIProgress, 33.333333, 0.0001) {
		t.Errorf("fi_progress = %v, want ~33.33", *got.FIProgress)
	}
	// liquid = 30000 + 20000; stock excluded.
	// runway = 50000 / 5000 = 10 months
	if got.RunwayMonths == nil || *got.RunwayMonths != 10 {
		t.Errorf("runway_months = %v, want 10", got.RunwayMonths)
	}
	if got.WithdrawalRate == nil || *got.WithdrawalRate != 0.04 {
		t.Errorf("withdrawal_rate = %v, want 0.04", got.WithdrawalRate)
	}
}

func TestComputeFIREThreePercentRate(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		MonthlyExpenses: decimal.NewFromInt(4000),
		WithdrawalRate:  decimal.NewFromFloat(0.03),
	}
	got := computeFIRE(nil, cfg, 0)
	// annual = 48000; fire = 48000 / 0.03 = 1_600_000
	if got.FIRENumber == nil || *got.FIRENumber != 1_600_000 {
		t.Errorf("fire_number = %v, want 1.6M", got.FIRENumber)
	}
}

func TestComputeFIREZeroExpenses(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		MonthlyExpenses: decimal.Zero,
		WithdrawalRate:  decimal.NewFromFloat(0.04),
	}
	got := computeFIRE(nil, cfg, 100_000)
	// Zero monthly expenses → every FIRE-tile field is nil (the entire tile
	// is hidden on the FE), including withdrawal_rate.
	if got.AnnualExpenses != nil || got.FIRENumber != nil ||
		got.FIProgress != nil || got.RunwayMonths != nil ||
		got.WithdrawalRate != nil {
		t.Errorf("expected all-nil when monthly expenses == 0, got %+v", got)
	}
}

func TestComputeFIREZeroWithdrawalRate(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		MonthlyExpenses: decimal.NewFromInt(5000),
		WithdrawalRate:  decimal.Zero,
	}
	got := computeFIRE(nil, cfg, 100_000)
	if got.FIRENumber != nil || got.FIProgress != nil {
		t.Error("fire_number/fi_progress should be nil when withdrawal_rate == 0")
	}
	if got.AnnualExpenses == nil || got.RunwayMonths == nil {
		t.Error("annual_expenses + runway_months should still compute")
	}
}

func TestComputeFIRENegativeNetWorthSuppressesProgress(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		MonthlyExpenses: decimal.NewFromInt(5000),
		WithdrawalRate:  decimal.NewFromFloat(0.04),
	}
	got := computeFIRE(nil, cfg, -10_000)
	if got.FIProgress != nil {
		t.Errorf("fi_progress should be nil for non-positive net worth, got %v", *got.FIProgress)
	}
}

func TestComputeFIRELiquidExcludesNonBank(t *testing.T) {
	t.Parallel()
	rows := []mergedRow{
		newAccountRow("bank", "asset", 10_000),
		newAccountRow("stock", "asset", 50_000),
		newAccountRow("real_estate", "asset", 500_000),
		newAccountRow("bank", "liability", 5_000),
	}
	cfg := AppConfig{
		MonthlyExpenses: decimal.NewFromInt(2_000),
		WithdrawalRate:  decimal.NewFromFloat(0.04),
	}
	got := computeFIRE(rows, cfg, 0)
	// Only the bank-asset row counts: 10000 / 2000 = 5.
	if got.RunwayMonths == nil || *got.RunwayMonths != 5 {
		t.Errorf("runway = %v, want 5", got.RunwayMonths)
	}
}

func approxEqual(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}
