package exposure

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestBuildReport_EmptyPortfolio(t *testing.T) {
	rep := BuildReport(nil, "", nil, 5)
	if rep.TotalPLN != 0 {
		t.Errorf("total = %v, want 0", rep.TotalPLN)
	}
	if rep.PLNPercent != 0 {
		t.Errorf("pln_pct = %v, want 0", rep.PLNPercent)
	}
	if rep.ForeignPct != 0 {
		t.Errorf("foreign_pct = %v, want 0", rep.ForeignPct)
	}
}

func TestBuildReport_PLNOnly(t *testing.T) {
	rep := BuildReport([]row{{currency: "PLN", valuePLN: d("100000")}}, "2026-05-01", nil, 5)
	if rep.PLNPercent != 100 {
		t.Errorf("pln_pct = %v, want 100", rep.PLNPercent)
	}
	if rep.ForeignPct != 0 {
		t.Errorf("foreign_pct = %v, want 0", rep.ForeignPct)
	}
}

func TestBuildReport_MixedCurrencies(t *testing.T) {
	rep := BuildReport([]row{
		{currency: "PLN", valuePLN: d("60000")},
		{currency: "USD", valuePLN: d("30000")},
		{currency: "EUR", valuePLN: d("10000")},
	}, "2026-05-01", nil, 5)
	if rep.PLNPercent != 60 {
		t.Errorf("pln_pct = %v, want 60", rep.PLNPercent)
	}
	if rep.ForeignPct != 40 {
		t.Errorf("foreign_pct = %v, want 40", rep.ForeignPct)
	}
	if got := rep.Currencies[0].Currency; got != "PLN" {
		t.Errorf("first currency = %q, want PLN (largest slice)", got)
	}
	if rep.Drift != nil {
		t.Errorf("drift = %+v, want nil when target unset", rep.Drift)
	}
}

func TestBuildReport_DropsZeroAndNegativeBuckets(t *testing.T) {
	rep := BuildReport([]row{
		{currency: "PLN", valuePLN: d("50000")},
		{currency: "JPY", valuePLN: d("0")},
		{currency: "GBP", valuePLN: d("-1000")},
	}, "", nil, 5)
	if got := len(rep.Currencies); got != 1 {
		t.Fatalf("currency rows = %d, want 1 (zero + negative dropped)", got)
	}
	if rep.Currencies[0].Currency != "PLN" {
		t.Errorf("kept currency = %q, want PLN", rep.Currencies[0].Currency)
	}
}

func TestBuildReport_DriftBandTrue(t *testing.T) {
	target := 70.0
	rep := BuildReport([]row{
		{currency: "PLN", valuePLN: d("72000")},
		{currency: "USD", valuePLN: d("28000")},
	}, "2026-05-01", &target, 5)
	if rep.Drift == nil {
		t.Fatal("drift = nil, want populated band")
	}
	if math.Abs(rep.Drift.DriftPLNPct-2) > 0.001 {
		t.Errorf("drift = %v, want 2.0", rep.Drift.DriftPLNPct)
	}
	if !rep.Drift.WithinTol {
		t.Errorf("within_tolerance = false, want true (2 ≤ 5)")
	}
}

func TestBuildReport_DriftBandOutOfTolerance(t *testing.T) {
	target := 70.0
	rep := BuildReport([]row{
		{currency: "PLN", valuePLN: d("50000")},
		{currency: "USD", valuePLN: d("50000")},
	}, "2026-05-01", &target, 5)
	if rep.Drift.WithinTol {
		t.Errorf("within_tolerance = true, want false (drift -20 > tolerance 5)")
	}
}
