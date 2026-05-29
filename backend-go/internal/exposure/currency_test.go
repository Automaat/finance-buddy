package exposure

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

const epsilon = 1e-9

func eq(got, want float64) bool { return math.Abs(got-want) < epsilon }

func TestBuildReport_EmptyPortfolio(t *testing.T) {
	rep := BuildReport(nil, "", nil, 5)
	if !eq(rep.TotalPLN, 0) {
		t.Errorf("total = %v, want 0", rep.TotalPLN)
	}
	if !eq(rep.PLNPercent, 0) {
		t.Errorf("pln_pct = %v, want 0", rep.PLNPercent)
	}
	if !eq(rep.ForeignPct, 0) {
		t.Errorf("foreign_pct = %v, want 0", rep.ForeignPct)
	}
}

func TestBuildReport_PLNOnly(t *testing.T) {
	rep := BuildReport([]row{{currency: "PLN", valuePLN: d("100000")}}, "2026-05-01", nil, 5)
	if !eq(rep.PLNPercent, 100) {
		t.Errorf("pln_pct = %v, want 100", rep.PLNPercent)
	}
	if !eq(rep.ForeignPct, 0) {
		t.Errorf("foreign_pct = %v, want 0", rep.ForeignPct)
	}
}

func TestBuildReport_MixedCurrencies(t *testing.T) {
	rep := BuildReport([]row{
		{currency: "PLN", valuePLN: d("60000")},
		{currency: "USD", valuePLN: d("30000")},
		{currency: "EUR", valuePLN: d("10000")},
	}, "2026-05-01", nil, 5)
	if !eq(rep.PLNPercent, 60) {
		t.Errorf("pln_pct = %v, want 60", rep.PLNPercent)
	}
	if !eq(rep.ForeignPct, 40) {
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

// bucketByCurrency tests cover the new investment-account fan-out logic
// that replaces the old "GROUP BY accounts.currency" aggregation.

func TestBucketByCurrency_BankAccountUsesAccountCurrency(t *testing.T) {
	// Bank account (category=bank) — no holdings ratio applies.
	rows := bucketByCurrency([]accountValue{
		{accountID: 1, category: "bank", currency: "PLN", valuePLN: d("10000")},
	}, nil)
	if len(rows) != 1 || rows[0].currency != "PLN" || !rows[0].valuePLN.Equal(d("10000")) {
		t.Fatalf("got %+v, want one PLN/10000 row", rows)
	}
}

func TestBucketByCurrency_InvestmentAccountFansOutByHoldings(t *testing.T) {
	// IKE XTB (PLN-settled, category=etf) holds USD-denominated ISAC.UK.
	// Snapshot value 53 495 PLN should attribute entirely to USD.
	breakdown := map[int]map[string]decimal.Decimal{
		15: {"USD": d("53495.15")},
	}
	rows := bucketByCurrency([]accountValue{
		{accountID: 15, category: "etf", currency: "PLN", valuePLN: d("53495.15")},
	}, breakdown)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].currency != "USD" {
		t.Errorf("currency = %s, want USD", rows[0].currency)
	}
	if !rows[0].valuePLN.Equal(d("53495.15")) {
		t.Errorf("value = %s, want 53495.15", rows[0].valuePLN)
	}
}

func TestBucketByCurrency_MixedHoldingsSplitProportionally(t *testing.T) {
	// Hypothetical IKZE Mbank holding 60% USD ISAC + 40% EUR IUSQ.
	// Snapshot value 10 000 PLN → 6 000 USD bucket, 4 000 EUR bucket.
	breakdown := map[int]map[string]decimal.Decimal{
		14: {"USD": d("60"), "EUR": d("40")}, // ratios; magnitudes don't matter
	}
	rows := bucketByCurrency([]accountValue{
		{accountID: 14, category: "etf", currency: "PLN", valuePLN: d("10000")},
	}, breakdown)
	got := map[string]decimal.Decimal{}
	for _, r := range rows {
		got[r.currency] = r.valuePLN
	}
	if !got["USD"].Equal(d("6000")) {
		t.Errorf("USD = %s, want 6000", got["USD"])
	}
	if !got["EUR"].Equal(d("4000")) {
		t.Errorf("EUR = %s, want 4000", got["EUR"])
	}
}

func TestBucketByCurrency_InvestmentWithoutHoldingsFallsBackToAccountCurrency(t *testing.T) {
	// Empty holdings (no lots) — must fall back to accounts.currency, not
	// silently swallow the value.
	rows := bucketByCurrency([]accountValue{
		{accountID: 99, category: "stock", currency: "PLN", valuePLN: d("5000")},
	}, map[int]map[string]decimal.Decimal{})
	if len(rows) != 1 || rows[0].currency != "PLN" || !rows[0].valuePLN.Equal(d("5000")) {
		t.Fatalf("got %+v, want PLN/5000 fallback", rows)
	}
}

func TestBucketByCurrency_MixedAccountsTotalPreserved(t *testing.T) {
	// Realistic mix: PLN savings + USD-holding IKE XTB + EUR-holding IKZE
	// Bossa + a real-estate row. Totals must be preserved across buckets.
	breakdown := map[int]map[string]decimal.Decimal{
		15: {"USD": d("53495.15")},
		16: {"EUR": d("59462.54")},
	}
	rows := bucketByCurrency([]accountValue{
		{accountID: 1, category: "bank", currency: "PLN", valuePLN: d("20000")},
		{accountID: 15, category: "etf", currency: "PLN", valuePLN: d("53495.15")},
		{accountID: 16, category: "etf", currency: "PLN", valuePLN: d("59462.54")},
		{accountID: 7, category: "real_estate", currency: "PLN", valuePLN: d("450000")},
	}, breakdown)
	totals := map[string]decimal.Decimal{}
	for _, r := range rows {
		totals[r.currency] = r.valuePLN
	}
	if !totals["PLN"].Equal(d("470000")) {
		t.Errorf("PLN = %s, want 470000 (bank + real_estate)", totals["PLN"])
	}
	if !totals["USD"].Equal(d("53495.15")) {
		t.Errorf("USD = %s, want 53495.15", totals["USD"])
	}
	if !totals["EUR"].Equal(d("59462.54")) {
		t.Errorf("EUR = %s, want 59462.54", totals["EUR"])
	}
}

func TestBucketByCurrency_ZeroWeightsFallBackToPLN(t *testing.T) {
	// Edge case: holdings exist but valued at zero (e.g. all positions
	// closed). Fall back to a single PLN bucket so the value isn't lost.
	breakdown := map[int]map[string]decimal.Decimal{
		15: {"USD": decimal.Zero},
	}
	rows := bucketByCurrency([]accountValue{
		{accountID: 15, category: "etf", currency: "PLN", valuePLN: d("1000")},
	}, breakdown)
	if len(rows) != 1 || rows[0].currency != "PLN" || !rows[0].valuePLN.Equal(d("1000")) {
		t.Fatalf("got %+v, want PLN/1000 fallback on zero weights", rows)
	}
}
