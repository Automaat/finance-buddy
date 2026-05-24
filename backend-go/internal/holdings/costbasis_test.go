package holdings

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

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

func TestIsValidSide(t *testing.T) {
	if !IsValidSide("buy") || !IsValidSide("sell") {
		t.Errorf("buy/sell should be valid")
	}
	if IsValidSide("short") {
		t.Errorf("short should be invalid")
	}
}
