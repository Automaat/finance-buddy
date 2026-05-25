package pit38

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
)

type stubRates map[string]decimal.Decimal

func (s stubRates) GetRateToPLN(_ context.Context, currency string, _ time.Time) (fx.Result, error) {
	if currency == "PLN" {
		return fx.Result{Rate: decimal.NewFromInt(1), Found: true}, nil
	}
	if r, ok := s[currency]; ok {
		return fx.Result{Rate: r, Found: true}, nil
	}
	return fx.Result{Found: false}, nil
}

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func dateAt(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return v
}

func TestComputeReport_PLN_OneSale(t *testing.T) {
	rates := stubRates{}
	rep, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{{
		SecurityID: 1, Symbol: "CDR", Currency: "PLN",
		Lots: []holdings.Lot{
			{Side: holdings.SideBuy, Quantity: d("10"), Price: d("200"), Fee: d("5"), Date: dateAt(t, "2025-06-01")},
			{Side: holdings.SideSell, Quantity: d("10"), Price: d("250"), Fee: d("5"), Date: dateAt(t, "2026-03-15")},
		},
	}})
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if got := len(rep.Rows); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
	row := rep.Rows[0]
	// proceeds = 10 * 250 - 5 = 2495; cost_basis = 10 * (2000+5)/10 = 2005;
	// realized = 2495 - 2005 = 490
	if want := d("2495"); !row.Proceeds.Equal(want) {
		t.Errorf("proceeds = %s, want %s", row.Proceeds, want)
	}
	if want := d("2005"); !row.CostBasis.Equal(want) {
		t.Errorf("cost_basis = %s, want %s", row.CostBasis, want)
	}
	if want := d("490"); !row.RealizedGain.Equal(want) {
		t.Errorf("realized = %s, want %s", row.RealizedGain, want)
	}
	if row.HasFX {
		t.Errorf("has_fx = true; want false for PLN")
	}
	if want := d("490"); !rep.Totals.RealizedPLN.Equal(want) {
		t.Errorf("totals.realized_pln = %s, want %s", rep.Totals.RealizedPLN, want)
	}
}

func TestComputeReport_USD_AppliesNBPRates(t *testing.T) {
	// Buy at rate 4.00, sell at rate 5.00 — currency gain widens the PLN
	// realized vs. the USD realized.
	rates := stubRatesDated{
		"2025-06-01": d("4.0"),
		"2026-03-15": d("5.0"),
	}
	rep, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{{
		SecurityID: 2, Symbol: "AAPL", Currency: "USD",
		Lots: []holdings.Lot{
			{Side: holdings.SideBuy, Quantity: d("10"), Price: d("100"), Fee: d("0"), Date: dateAt(t, "2025-06-01")},
			{Side: holdings.SideSell, Quantity: d("10"), Price: d("110"), Fee: d("0"), Date: dateAt(t, "2026-03-15")},
		},
	}})
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if got := len(rep.Rows); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
	row := rep.Rows[0]
	// proceeds_pln = 10*110 * 5 = 5500; cost_basis_pln = 10 * 100 * 4 = 4000
	// realized_pln = 1500 (vs. 100 USD which would only be 500 PLN at sale rate)
	if want := d("5500"); !row.ProceedsPLN.Equal(want) {
		t.Errorf("proceeds_pln = %s, want %s", row.ProceedsPLN, want)
	}
	if want := d("4000"); !row.CostBasisPLN.Equal(want) {
		t.Errorf("cost_basis_pln = %s, want %s", row.CostBasisPLN, want)
	}
	if want := d("1500"); !row.RealizedPLN.Equal(want) {
		t.Errorf("realized_pln = %s, want %s", row.RealizedPLN, want)
	}
	if !row.HasFX {
		t.Errorf("has_fx = false; want true for USD")
	}
}

func TestComputeReport_SkipsSalesOutsideYear(t *testing.T) {
	rates := stubRates{}
	rep, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{{
		SecurityID: 1, Symbol: "CDR", Currency: "PLN",
		Lots: []holdings.Lot{
			{Side: holdings.SideBuy, Quantity: d("10"), Price: d("100"), Date: dateAt(t, "2024-01-15")},
			{Side: holdings.SideSell, Quantity: d("5"), Price: d("120"), Date: dateAt(t, "2025-12-31")},
			{Side: holdings.SideSell, Quantity: d("5"), Price: d("130"), Date: dateAt(t, "2026-02-01")},
		},
	}})
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if got := len(rep.Rows); got != 1 {
		t.Fatalf("rows = %d, want 1 (only the 2026 sale)", got)
	}
	if !rep.Rows[0].Date.Equal(dateAt(t, "2026-02-01")) {
		t.Errorf("date = %s, want 2026-02-01", rep.Rows[0].Date)
	}
}

func TestComputeReport_MissingRateErrors(t *testing.T) {
	// USD lot with no rate available — must surface ErrUnknownCurrency.
	rates := stubRates{}
	_, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{{
		SecurityID: 3, Symbol: "TSLA", Currency: "USD",
		Lots: []holdings.Lot{
			{Side: holdings.SideBuy, Quantity: d("1"), Price: d("200"), Date: dateAt(t, "2025-06-01")},
			{Side: holdings.SideSell, Quantity: d("1"), Price: d("250"), Date: dateAt(t, "2026-03-15")},
		},
	}})
	if err == nil {
		t.Fatal("ComputeReport: want error, got nil")
	}
}

func TestComputeReport_TotalsSumPLNFields(t *testing.T) {
	rates := stubRates{}
	rep, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{
		{
			SecurityID: 1, Symbol: "CDR", Currency: "PLN",
			Lots: []holdings.Lot{
				{Side: holdings.SideBuy, Quantity: d("10"), Price: d("100"), Date: dateAt(t, "2025-01-01")},
				{Side: holdings.SideSell, Quantity: d("10"), Price: d("120"), Date: dateAt(t, "2026-04-01")},
			},
		},
		{
			SecurityID: 2, Symbol: "ALG", Currency: "PLN",
			Lots: []holdings.Lot{
				{Side: holdings.SideBuy, Quantity: d("5"), Price: d("50"), Date: dateAt(t, "2024-01-01")},
				{Side: holdings.SideSell, Quantity: d("5"), Price: d("80"), Date: dateAt(t, "2026-09-01")},
			},
		},
	})
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	// CDR: (120-100)*10 = 200 PLN. ALG: (80-50)*5 = 150 PLN. Sum = 350 PLN.
	if want := d("350"); !rep.Totals.RealizedPLN.Equal(want) {
		t.Errorf("totals.realized_pln = %s, want %s", rep.Totals.RealizedPLN, want)
	}
}

func TestComputeReport_SortsRowsByDate(t *testing.T) {
	rates := stubRates{}
	rep, err := ComputeReport(t.Context(), 2026, rates, []SecurityLots{
		{
			SecurityID: 2, Symbol: "ALG", Currency: "PLN",
			Lots: []holdings.Lot{
				{Side: holdings.SideBuy, Quantity: d("5"), Price: d("50"), Date: dateAt(t, "2024-01-01")},
				{Side: holdings.SideSell, Quantity: d("5"), Price: d("80"), Date: dateAt(t, "2026-09-01")},
			},
		},
		{
			SecurityID: 1, Symbol: "CDR", Currency: "PLN",
			Lots: []holdings.Lot{
				{Side: holdings.SideBuy, Quantity: d("10"), Price: d("100"), Date: dateAt(t, "2025-01-01")},
				{Side: holdings.SideSell, Quantity: d("10"), Price: d("120"), Date: dateAt(t, "2026-04-01")},
			},
		},
	})
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if got := rep.Rows[0].Symbol; got != "CDR" {
		t.Errorf("first row symbol = %q, want CDR (April before September)", got)
	}
}

// stubRatesDated lets a test return a different rate per ISO date.
type stubRatesDated map[string]decimal.Decimal

func (s stubRatesDated) GetRateToPLN(_ context.Context, currency string, onDate time.Time) (fx.Result, error) {
	if currency == "PLN" {
		return fx.Result{Rate: decimal.NewFromInt(1), Found: true}, nil
	}
	key := onDate.Format("2006-01-02")
	if r, ok := s[key]; ok {
		return fx.Result{Rate: r, Found: true}, nil
	}
	return fx.Result{Found: false}, nil
}
