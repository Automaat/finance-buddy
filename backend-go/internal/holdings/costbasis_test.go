package holdings

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
)

// stubRates returns rate r for every lookup, found=true. For tests.
type stubRates struct{ r decimal.Decimal }

func (s stubRates) GetRateToPLN(_ context.Context, _ string, _ time.Time) (fx.Result, error) {
	return fx.Result{Rate: s.r, Found: true}, nil
}

// dateRates returns per-date rates; lookups for missing dates return found=false.
type dateRates map[string]decimal.Decimal

func (m dateRates) GetRateToPLN(_ context.Context, _ string, on time.Time) (fx.Result, error) {
	key := on.UTC().Format("2006-01-02")
	if v, ok := m[key]; ok {
		return fx.Result{Rate: v, Found: true}, nil
	}
	return fx.Result{}, nil
}

func d(s string) decimal.Decimal {
	v, _ := decimal.NewFromString(s)
	return v
}

func date(t *testing.T, s string) time.Time {
	t.Helper()
	out, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return out
}

func TestComputeRunning_SingleBuy(t *testing.T) {
	r, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Fee: d("5"), Date: date(t, "2024-01-01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Quantity.Equal(d("10")) {
		t.Errorf("qty = %s, want 10", r.Quantity)
	}
	// avg = (10*100 + 5) / 10 = 100.5
	if !r.AverageCost.Equal(d("100.5")) {
		t.Errorf("avg cost = %s, want 100.5", r.AverageCost)
	}
	if !r.CostBasis.Equal(d("1005")) {
		t.Errorf("basis = %s, want 1005", r.CostBasis)
	}
	if !r.RealizedGain.IsZero() {
		t.Errorf("realized gain should be zero, got %s", r.RealizedGain)
	}
}

func TestComputeRunning_WeightedAverageOnRepeatBuys(t *testing.T) {
	// Buy 10 @ 100, then 10 @ 120. No fees. Avg should be 110.
	r, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
		{Side: SideBuy, Quantity: d("10"), Price: d("120"), Date: date(t, "2024-02-01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Quantity.Equal(d("20")) {
		t.Errorf("qty = %s, want 20", r.Quantity)
	}
	if !r.AverageCost.Equal(d("110")) {
		t.Errorf("avg cost = %s, want 110", r.AverageCost)
	}
}

func TestComputeRunning_PartialSaleRealizesGain(t *testing.T) {
	// Buy 10 @ 100, sell 4 @ 150. Realized gain = (150 - 100) * 4 = 200.
	// Remaining qty = 6, avg cost stays 100.
	r, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
		{Side: SideSell, Quantity: d("4"), Price: d("150"), Date: date(t, "2024-06-01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Quantity.Equal(d("6")) {
		t.Errorf("qty = %s, want 6", r.Quantity)
	}
	if !r.AverageCost.Equal(d("100")) {
		t.Errorf("avg cost should stay 100, got %s", r.AverageCost)
	}
	if !r.RealizedGain.Equal(d("200")) {
		t.Errorf("realized gain = %s, want 200", r.RealizedGain)
	}
}

func TestComputeRunning_FullSaleZerosAvgCost(t *testing.T) {
	r, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
		{Side: SideSell, Quantity: d("10"), Price: d("120"), Date: date(t, "2024-06-01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Quantity.IsZero() {
		t.Errorf("qty = %s, want 0", r.Quantity)
	}
	if !r.AverageCost.IsZero() {
		t.Errorf("avg cost = %s, want 0", r.AverageCost)
	}
	if !r.RealizedGain.Equal(d("200")) {
		t.Errorf("realized gain = %s, want 200", r.RealizedGain)
	}
}

func TestComputeRunning_SellWithFeeReducesGain(t *testing.T) {
	r, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
		{Side: SideSell, Quantity: d("10"), Price: d("120"), Fee: d("15"), Date: date(t, "2024-06-01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Gain = (120 - 100) * 10 - 15 = 185
	if !r.RealizedGain.Equal(d("185")) {
		t.Errorf("realized gain = %s, want 185", r.RealizedGain)
	}
}

func TestComputeRunning_OversellErrors(t *testing.T) {
	_, err := ComputeRunning([]Lot{
		{Side: SideBuy, Quantity: d("5"), Price: d("100"), Date: date(t, "2024-01-01")},
		{Side: SideSell, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-06-01")},
	})
	if !errors.Is(err, ErrOversell) {
		t.Errorf("expected ErrOversell, got %v", err)
	}
}

func TestUnrealizedGain(t *testing.T) {
	r := Running{Quantity: d("10"), AverageCost: d("100")}
	got := UnrealizedGain(r, d("120"))
	if !got.Equal(d("200")) {
		t.Errorf("unrealized = %s, want 200", got)
	}
}

func TestUnrealizedGain_NoQuote(t *testing.T) {
	r := Running{Quantity: d("10"), AverageCost: d("100")}
	if !UnrealizedGain(r, decimal.Zero).IsZero() {
		t.Errorf("no quote should yield zero unrealized")
	}
}

func TestMarketValue_NoQuoteUsesCostBasis(t *testing.T) {
	r := Running{Quantity: d("10"), AverageCost: d("100"), CostBasis: d("1000")}
	if !MarketValue(r, decimal.Zero).Equal(d("1000")) {
		t.Errorf("no quote should fall back to cost basis")
	}
}

func TestMarketValue_QuotedUsesQuote(t *testing.T) {
	r := Running{Quantity: d("10"), AverageCost: d("100"), CostBasis: d("1000")}
	if !MarketValue(r, d("150")).Equal(d("1500")) {
		t.Errorf("quote should give 1500")
	}
}

func TestComputeRunningPLN_WeightedPLNAverage(t *testing.T) {
	rates := dateRates{
		"2025-05-02": d("3.7881"),
		"2025-05-09": d("3.7988"),
		"2026-05-26": d("3.6576"),
	}
	lots := []Lot{
		{Side: SideBuy, Quantity: d("75.3421"), Price: d("89.310"), Date: date(t, "2025-05-02")},
		{Side: SideBuy, Quantity: d("1.4583"), Price: d("90.013"), Date: date(t, "2025-05-09")},
		{Side: SideBuy, Quantity: d("44.3344"), Price: d("121.070"), Date: date(t, "2026-05-26")},
	}
	r, err := ComputeRunningPLN(context.Background(), lots, "USD", rates)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.HasPLN {
		t.Fatalf("expected HasPLN true")
	}
	// Per-lot opening PLN: 75.3421*89.310*3.7881 + 1.4583*90.013*3.7988 + 44.3344*121.070*3.6576 ≈ 45620.27.
	// Compare with 0.5 PLN tolerance to absorb decimal rounding.
	want := d("45620.27")
	diff := r.CostBasisPLN.Sub(want).Abs()
	if diff.GreaterThan(d("0.5")) {
		t.Errorf("CostBasisPLN = %s, want ≈ 45620.27 (diff %s)", r.CostBasisPLN.StringFixed(2), diff)
	}
	if r.RealizedGainPLN.Cmp(decimal.Zero) != 0 {
		t.Errorf("RealizedGainPLN should be zero (no sells), got %s", r.RealizedGainPLN)
	}
}

func TestComputeRunningPLN_MissingRateLeavesPLNZero(t *testing.T) {
	// One lot date has no rate — function should still return successfully
	// but with HasPLN=false so the caller falls back to native rendering.
	rates := dateRates{"2025-05-02": d("3.7881")}
	lots := []Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2025-05-02")},
		{Side: SideBuy, Quantity: d("10"), Price: d("110"), Date: date(t, "2025-06-01")}, // no rate
	}
	r, err := ComputeRunningPLN(context.Background(), lots, "USD", rates)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.HasPLN {
		t.Errorf("HasPLN should be false on partial coverage")
	}
	if !r.AverageCost.Equal(d("105")) {
		t.Errorf("native AverageCost = %s, want 105", r.AverageCost)
	}
}

func TestComputeRunningPLN_PLNCurrencyPassesThrough(t *testing.T) {
	r, err := ComputeRunningPLN(context.Background(), []Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
	}, "PLN", stubRates{r: d("1")})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !r.HasPLN {
		t.Errorf("HasPLN should be true for PLN")
	}
	if !r.CostBasisPLN.Equal(d("1000")) {
		t.Errorf("CostBasisPLN = %s, want 1000", r.CostBasisPLN)
	}
}

func TestComputeRunningPLN_NilRatesActsLikeComputeRunning(t *testing.T) {
	r, err := ComputeRunningPLN(context.Background(), []Lot{
		{Side: SideBuy, Quantity: d("10"), Price: d("100"), Date: date(t, "2024-01-01")},
	}, "USD", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.HasPLN {
		t.Errorf("HasPLN should stay false with nil rates")
	}
	if !r.AverageCost.Equal(d("100")) {
		t.Errorf("AverageCost = %s, want 100", r.AverageCost)
	}
}

func TestIsValidSide(t *testing.T) {
	if !IsValidSide("buy") || !IsValidSide("sell") {
		t.Errorf("buy/sell should be valid")
	}
	if IsValidSide("short") {
		t.Errorf("short should be invalid")
	}
}
