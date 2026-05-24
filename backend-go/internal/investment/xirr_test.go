package investment

import (
	"math"
	"testing"
	"time"
)

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

func TestXIRR_FlatBuyAndHoldDoubles(t *testing.T) {
	// Put 1000 in on day 1, value 2000 one year later -> ~100% XIRR.
	flows := []CashFlow{
		{Date: date(t, "2024-01-01"), Amount: 1000},
		{Date: date(t, "2025-01-01"), Amount: -2000},
	}
	r, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(r-1.0) > 0.01 {
		t.Errorf("expected ~1.0, got %f", r)
	}
}

func TestXIRR_StaircaseContributions(t *testing.T) {
	// Deposit 1000/month for a year, end at 13_000 — XIRR is small positive.
	flows := []CashFlow{}
	d := date(t, "2024-01-01")
	for i := range 12 {
		flows = append(flows, CashFlow{Date: d.AddDate(0, i, 0), Amount: 1000})
	}
	flows = append(flows, CashFlow{Date: date(t, "2024-12-31"), Amount: -13000})
	r, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r <= 0 || r > 1.0 {
		t.Errorf("expected modest positive return, got %f", r)
	}
}

func TestXIRR_DepositsAlone_NotConverge(t *testing.T) {
	// Only positive flows — XIRR is undefined.
	_, err := XIRR([]CashFlow{
		{Date: date(t, "2024-01-01"), Amount: 1000},
		{Date: date(t, "2024-06-01"), Amount: 500},
	})
	if err == nil {
		t.Errorf("expected convergence error for all-positive flows")
	}
}

func TestXIRR_EmptyFlows(t *testing.T) {
	_, err := XIRR(nil)
	if err == nil {
		t.Errorf("expected convergence error for empty flows")
	}
}

func TestXIRR_DepositsAreNotGains(t *testing.T) {
	// Deposit 1000 then deposit another 1000, value stays at 2000 — no growth.
	flows := []CashFlow{
		{Date: date(t, "2024-01-01"), Amount: 1000},
		{Date: date(t, "2024-06-01"), Amount: 1000},
		{Date: date(t, "2024-12-31"), Amount: -2000},
	}
	r, err := XIRR(flows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(r) > 0.01 {
		t.Errorf("flat value should give ~0 XIRR, got %f", r)
	}
}

func TestSimpleROI(t *testing.T) {
	if SimpleROI(1000, 1200) != 0.2 {
		t.Errorf("expected 20%% ROI, got %f", SimpleROI(1000, 1200))
	}
	if SimpleROI(0, 1000) != 0 {
		t.Errorf("zero contributed should yield 0 ROI, got %f", SimpleROI(0, 1000))
	}
	if SimpleROI(1000, 800) != -0.2 {
		t.Errorf("expected -20%% ROI, got %f", SimpleROI(1000, 800))
	}
}
