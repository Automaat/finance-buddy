package bonds

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func bondEDO(purchase time.Time) *TreasuryBond {
	return &TreasuryBond{
		Type:          BondEDO,
		FaceValue:     decimal.NewFromInt(1000),
		PurchaseDate:  purchase,
		FirstYearRate: decimal.NewFromFloat(6.80),
		Margin:        decimal.NewFromFloat(2.0),
		Capitalize:    true,
	}
}

func bondCOI(purchase time.Time) *TreasuryBond {
	return &TreasuryBond{
		Type:          BondCOI,
		FaceValue:     decimal.NewFromInt(1000),
		PurchaseDate:  purchase,
		FirstYearRate: decimal.NewFromFloat(6.55),
		Margin:        decimal.NewFromFloat(1.25),
		Capitalize:    false,
	}
}

func TestCurrentValueBeforePurchaseIsFaceValue(t *testing.T) {
	b := bondEDO(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	now := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	got := CurrentValue(b, map[int]decimal.Decimal{2025: decimal.NewFromFloat(105)}, now)
	if !got.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("pre-purchase value should be face, got %s", got)
	}
}

func TestCurrentValueYear1PartialEDO(t *testing.T) {
	// 6 months at 6.80% — capitalized 1000 * (1 + 0.068 * 0.5) = 1034.0
	purchase := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) // ~half-year in
	b := bondEDO(purchase)
	got := CurrentValue(b, map[int]decimal.Decimal{}, now)
	want := decimal.NewFromFloat(1034.00)
	if got.Sub(want).Abs().GreaterThan(decimal.NewFromFloat(0.1)) {
		t.Fatalf("year1 partial = %s, want %s", got, want)
	}
}

func TestCurrentValueYear2EDOCompounding(t *testing.T) {
	purchase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // exactly 2y
	yoy := map[int]decimal.Decimal{
		2024: decimal.NewFromFloat(105.0), // not used for year 1
		2025: decimal.NewFromFloat(104.0), // applies to year 2 (purchase_year + 2 - 1)
	}
	b := bondEDO(purchase)
	// year 1: 1000 * 1.0680 = 1068.0
	// year 2: 1068.0 * (1 + (4.0 + 2.0)/100) = 1068.0 * 1.06 = 1132.08
	got := CurrentValue(b, yoy, now)
	want := decimal.NewFromFloat(1132.08)
	if !got.Equal(want) {
		t.Fatalf("year2 EDO compounded = %s, want %s", got, want)
	}
}

func TestCurrentValueCOIInterestPaidOut(t *testing.T) {
	purchase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	yoy := map[int]decimal.Decimal{
		2025: decimal.NewFromFloat(104.0),
	}
	b := bondCOI(purchase)
	// year 1: 1000 * 0.0655 = 65.50 paid out
	// year 2: 1000 * (4.0 + 1.25)/100 = 52.50 paid out
	// value = face + accrued = 1000 + 65.50 + 52.50 = 1118.00
	got := CurrentValue(b, yoy, now)
	want := decimal.NewFromFloat(1118.00)
	if !got.Equal(want) {
		t.Fatalf("COI accrued = %s, want %s", got, want)
	}
}

func TestYearRateFallsBackToFirstYearWhenCPIEmpty(t *testing.T) {
	b := bondEDO(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	r := YearRate(b, map[int]decimal.Decimal{}, 3)
	if !r.Equal(b.FirstYearRate) {
		t.Fatalf("empty CPI should fall back to first-year rate, got %s", r)
	}
}

func TestYearRateClampsFutureYearsToLatestKnown(t *testing.T) {
	purchase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := bondEDO(purchase)
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(103.5)}
	// Year 5 => purchase_year + 5 - 1 = 2028, not in map → clamp to latest = 2025.
	r := YearRate(b, yoy, 5)
	want := decimal.NewFromFloat(5.5) // (103.5 - 100) + 2.0
	if !r.Equal(want) {
		t.Fatalf("year 5 rate = %s, want %s", r, want)
	}
}

func TestYieldToMaturityEDOReturnsTenSamples(t *testing.T) {
	purchase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := bondEDO(purchase)
	yoy := map[int]decimal.Decimal{2025: decimal.NewFromFloat(104)}
	pts := YieldToMaturity(b, yoy)
	// 1 anchor (year 0) + 10 yearly samples
	if len(pts) != 11 {
		t.Fatalf("expected 11 YTM points, got %d", len(pts))
	}
	if !pts[0].Value.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("year 0 should be face value, got %s", pts[0].Value)
	}
	if pts[10].Year != 10 {
		t.Fatalf("last point should be year 10, got %d", pts[10].Year)
	}
	if pts[10].Value.LessThanOrEqual(pts[0].Value) {
		t.Fatalf("EDO should grow over 10y, got %s", pts[10].Value)
	}
}

func TestCurrentValueCapsAtMaturity(t *testing.T) {
	purchase := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // 16y past, EDO matures at 10y
	b := bondEDO(purchase)
	yoy := map[int]decimal.Decimal{2020: decimal.NewFromFloat(103)}
	got := CurrentValue(b, yoy, now)
	// Bond should equal its end-of-year-10 value, not continue compounding.
	pts := YieldToMaturity(b, yoy)
	want := pts[10].Value
	if !got.Equal(want) {
		t.Fatalf("post-maturity should cap at year-10 value; got %s want %s", got, want)
	}
}

func TestMaturityDateMatchesBondTenor(t *testing.T) {
	purchase := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	b := bondEDO(purchase)
	want := time.Date(2034, 1, 15, 0, 0, 0, 0, time.UTC)
	if got := MaturityDate(b); !got.Equal(want) {
		t.Fatalf("EDO matures = %s, want %s", got, want)
	}
}
