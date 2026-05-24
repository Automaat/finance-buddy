package dashboard

import (
	"math"
	"testing"
	"time"

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

func TestAddCoastFIREHappyPath(t *testing.T) {
	t.Parallel()
	// Born 1990-01-01, "now" = 2025-01-01 → currentAge ≈ 35; target 65 → 30 years.
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	targetAge := 65
	cfg := AppConfig{
		BirthDate:          birth,
		CoastFIRETargetAge: &targetAge,
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	fire := 1_500_000.0
	out := fireMetrics{FIRENumber: &fire}

	addCoastFIRE(&out, cfg, 500_000, now)

	// coast = 1_500_000 / 1.07^30 ≈ 197_028
	wantCoast := 1_500_000 / math.Pow(1.07, 30)
	if out.CoastFIRENumber == nil || !approxEqual(*out.CoastFIRENumber, wantCoast, 1) {
		t.Fatalf("coast_fire_number = %v, want ≈%v", out.CoastFIRENumber, wantCoast)
	}
	// gap = coast − net_worth ≈ 197_028 − 500_000 < 0 (surplus)
	wantGap := wantCoast - 500_000
	if out.CoastFIREGap == nil || !approxEqual(*out.CoastFIREGap, wantGap, 1) {
		t.Fatalf("coast_fire_gap = %v, want ≈%v", out.CoastFIREGap, wantGap)
	}
	if out.CoastFIRETargetAge == nil || *out.CoastFIRETargetAge != 65 {
		t.Errorf("coast_fire_target_age = %v, want 65", out.CoastFIRETargetAge)
	}
	if out.ExpectedReturnRate == nil || *out.ExpectedReturnRate != 0.07 {
		t.Errorf("expected_return_rate = %v, want 0.07", out.ExpectedReturnRate)
	}
}

func TestAddCoastFIREGapPositiveWhenNetWorthShort(t *testing.T) {
	t.Parallel()
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	targetAge := 65
	cfg := AppConfig{
		BirthDate:          birth,
		CoastFIRETargetAge: &targetAge,
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	fire := 1_500_000.0
	out := fireMetrics{FIRENumber: &fire}

	addCoastFIRE(&out, cfg, 50_000, now)
	if out.CoastFIREGap == nil || *out.CoastFIREGap <= 0 {
		t.Fatalf("expected positive gap, got %v", out.CoastFIREGap)
	}
}

func TestAddCoastFIRENoTargetAgeStaysNil(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		BirthDate:          time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	fire := 1_500_000.0
	out := fireMetrics{FIRENumber: &fire}
	addCoastFIRE(&out, cfg, 500_000, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if out.CoastFIRENumber != nil || out.CoastFIREGap != nil {
		t.Errorf("expected nil coast fields without target age, got %+v", out)
	}
}

func TestAddCoastFIRETargetAgeAtOrBelowCurrentStaysNil(t *testing.T) {
	t.Parallel()
	// currentAge ≈ 35, target 30 → skip.
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	targetAge := 30
	cfg := AppConfig{
		BirthDate:          birth,
		CoastFIRETargetAge: &targetAge,
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	fire := 1_500_000.0
	out := fireMetrics{FIRENumber: &fire}
	addCoastFIRE(&out, cfg, 500_000, now)
	if out.CoastFIRENumber != nil {
		t.Errorf("expected nil coast_fire_number when target ≤ current age, got %v", *out.CoastFIRENumber)
	}
}

func TestAddCoastFIREZeroReturnRateStaysNil(t *testing.T) {
	t.Parallel()
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	targetAge := 65
	cfg := AppConfig{
		BirthDate:          birth,
		CoastFIRETargetAge: &targetAge,
		ExpectedReturnRate: decimal.Zero,
	}
	fire := 1_500_000.0
	out := fireMetrics{FIRENumber: &fire}
	addCoastFIRE(&out, cfg, 500_000, now)
	if out.CoastFIRENumber != nil {
		t.Errorf("expected nil coast_fire_number when return rate is zero, got %v", *out.CoastFIRENumber)
	}
}

func TestAddBaristaFIREHappyPath(t *testing.T) {
	t.Parallel()
	// Monthly expenses 5000 → annual 60000. Barista income 2000/mo → annual
	// 24000. Gap = 36000. Withdrawal 0.04 → barista FIRE = 900000.
	monthly := decimal.NewFromInt(2000)
	cfg := AppConfig{
		MonthlyExpenses:      decimal.NewFromInt(5000),
		WithdrawalRate:       decimal.RequireFromString("0.04"),
		ExpectedReturnRate:   decimal.RequireFromString("0.07"),
		BaristaMonthlyIncome: &monthly,
	}
	got := computeFIRE(nil, cfg, 500_000)

	if got.BaristaMonthlyIncome == nil || *got.BaristaMonthlyIncome != 2000 {
		t.Fatalf("barista_monthly_income = %v, want 2000", got.BaristaMonthlyIncome)
	}
	if got.BaristaAnnualGap == nil || *got.BaristaAnnualGap != 36_000 {
		t.Fatalf("barista_annual_gap = %v, want 36000", got.BaristaAnnualGap)
	}
	if got.BaristaFIRENumber == nil || *got.BaristaFIRENumber != 900_000 {
		t.Fatalf("barista_fire_number = %v, want 900000", got.BaristaFIRENumber)
	}
	// progress = 500k / 900k * 100 ≈ 55.56%
	if got.BaristaFIProgress == nil || !approxEqual(*got.BaristaFIProgress, 55.5555, 0.01) {
		t.Errorf("barista_fi_progress = %v, want ≈55.56", got.BaristaFIProgress)
	}
	// years = log(900000/500000) / log(1.07) ≈ 8.69
	wantYears := math.Log(900_000.0/500_000.0) / math.Log(1.07)
	if got.BaristaYearsToFI == nil || !approxEqual(*got.BaristaYearsToFI, wantYears, 0.01) {
		t.Errorf("barista_years_to_fi = %v, want ≈%v", got.BaristaYearsToFI, wantYears)
	}
}

func TestAddBaristaFIREIncomeCoversAllExpenses(t *testing.T) {
	t.Parallel()
	// Barista income 6000/mo > expenses 5000/mo → gap clamps to 0,
	// FIRE number = 0, progress + years-to-FI stay nil.
	monthly := decimal.NewFromInt(6000)
	cfg := AppConfig{
		MonthlyExpenses:      decimal.NewFromInt(5000),
		WithdrawalRate:       decimal.RequireFromString("0.04"),
		ExpectedReturnRate:   decimal.RequireFromString("0.07"),
		BaristaMonthlyIncome: &monthly,
	}
	got := computeFIRE(nil, cfg, 500_000)

	if got.BaristaAnnualGap == nil || *got.BaristaAnnualGap != 0 {
		t.Fatalf("barista_annual_gap = %v, want 0", got.BaristaAnnualGap)
	}
	if got.BaristaFIRENumber == nil || *got.BaristaFIRENumber != 0 {
		t.Fatalf("barista_fire_number = %v, want 0", got.BaristaFIRENumber)
	}
	if got.BaristaFIProgress != nil {
		t.Errorf("expected nil barista_fi_progress when fire = 0, got %v", *got.BaristaFIProgress)
	}
	if got.BaristaYearsToFI != nil {
		t.Errorf("expected nil barista_years_to_fi when fire = 0, got %v", *got.BaristaYearsToFI)
	}
}

func TestAddBaristaFIRENetWorthAlreadyAtTarget(t *testing.T) {
	t.Parallel()
	// Net worth already exceeds barista FIRE → years-to-FI nil (already there).
	monthly := decimal.NewFromInt(2000)
	cfg := AppConfig{
		MonthlyExpenses:      decimal.NewFromInt(5000),
		WithdrawalRate:       decimal.RequireFromString("0.04"),
		ExpectedReturnRate:   decimal.RequireFromString("0.07"),
		BaristaMonthlyIncome: &monthly,
	}
	got := computeFIRE(nil, cfg, 1_500_000)
	if got.BaristaYearsToFI != nil {
		t.Errorf("expected nil barista_years_to_fi when net worth ≥ target, got %v", *got.BaristaYearsToFI)
	}
	if got.BaristaFIProgress == nil || *got.BaristaFIProgress <= 100 {
		t.Errorf("expected progress > 100%%, got %v", got.BaristaFIProgress)
	}
}

func TestAddBaristaFIRENoIncomeStaysNil(t *testing.T) {
	t.Parallel()
	cfg := AppConfig{
		MonthlyExpenses:    decimal.NewFromInt(5000),
		WithdrawalRate:     decimal.RequireFromString("0.04"),
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	got := computeFIRE(nil, cfg, 500_000)
	if got.BaristaFIRENumber != nil || got.BaristaAnnualGap != nil {
		t.Errorf("expected nil barista fields without configured income, got %+v", got)
	}
}

func TestAddBaristaFIREZeroReturnRateNoYears(t *testing.T) {
	t.Parallel()
	monthly := decimal.NewFromInt(2000)
	cfg := AppConfig{
		MonthlyExpenses:      decimal.NewFromInt(5000),
		WithdrawalRate:       decimal.RequireFromString("0.04"),
		ExpectedReturnRate:   decimal.Zero,
		BaristaMonthlyIncome: &monthly,
	}
	got := computeFIRE(nil, cfg, 500_000)
	if got.BaristaFIRENumber == nil {
		t.Fatal("expected barista_fire_number set even without return rate")
	}
	if got.BaristaYearsToFI != nil {
		t.Errorf("expected nil barista_years_to_fi when return rate ≤ 0, got %v", *got.BaristaYearsToFI)
	}
}

func TestAddCoastFIRENoFIRENumberStaysNil(t *testing.T) {
	t.Parallel()
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	targetAge := 65
	cfg := AppConfig{
		BirthDate:          birth,
		CoastFIRETargetAge: &targetAge,
		ExpectedReturnRate: decimal.RequireFromString("0.07"),
	}
	out := fireMetrics{}
	addCoastFIRE(&out, cfg, 500_000, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if out.CoastFIRENumber != nil {
		t.Errorf("expected nil coast_fire_number without FIRE number, got %v", *out.CoastFIRENumber)
	}
}
