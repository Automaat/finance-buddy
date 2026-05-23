package bonds

import (
	"time"

	"github.com/shopspring/decimal"
)

// CurrentValue projects the bond's worth at `now`, given a CPI YoY map from
// the cpi_index table (yoy_rate as published by GUS — 114.4 = +14.4%).
//
// Rule per acceptance criteria:
//   - Year 1 accrues at FirstYearRate.
//   - Year N (N >= 2) accrues at (cpi_yoy_for_purchase_year+N-1) - 100 + Margin.
//     Missing CPI years clamp to the closest known year so freshly issued
//     bonds (current calendar year) don't read zero.
//   - Capitalize: each completed year multiplies the running value;
//     non-capitalized bonds add `face_value * year_rate` per completed year
//     (interest paid out rather than reinvested).
//   - Partial current year applies pro-rata at the same year's rate.
//
// Returns FaceValue when now is on/before PurchaseDate.
func CurrentValue(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, now time.Time) decimal.Decimal {
	if !now.After(b.PurchaseDate) {
		return b.FaceValue
	}
	completedYears, fraction := elapsedYearsAndFraction(b.PurchaseDate, now)
	// Past the tenor, the bond is redeemed at its final value; further years
	// don't accrue. Cap completedYears at the tenor and zero the partial
	// fraction so a long-stale row doesn't compound to infinity.
	tenorYears := bondTenorYears(b.Type)
	if completedYears >= tenorYears {
		completedYears = tenorYears
		fraction = decimal.Zero
	}

	value := b.FaceValue
	accruedPayout := decimal.Zero
	for yearIdx := 1; yearIdx <= completedYears; yearIdx++ {
		rate := yearRate(b, yoyByYear, yearIdx)
		if b.Capitalize {
			value = value.Mul(decimal.NewFromInt(1).Add(rate.Div(decimal.NewFromInt(100))))
		} else {
			accruedPayout = accruedPayout.Add(b.FaceValue.Mul(rate).Div(decimal.NewFromInt(100)))
		}
	}
	if fraction.IsPositive() {
		rate := yearRate(b, yoyByYear, completedYears+1)
		if b.Capitalize {
			factor := decimal.NewFromInt(1).Add(rate.Mul(fraction).Div(decimal.NewFromInt(100)))
			value = value.Mul(factor)
		} else {
			value = value.Add(b.FaceValue.Mul(rate).Mul(fraction).Div(decimal.NewFromInt(100)))
		}
	}
	if !b.Capitalize {
		value = value.Add(accruedPayout)
	}
	return value.Round(2)
}

// YearRate is the annual % applied to bond year `yearIdx` (1-based).
//
// Exposed for the YTM projection so the frontend can render the
// per-year coupon line; the calculator inside CurrentValue calls the same
// path.
func YearRate(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, yearIdx int) decimal.Decimal {
	return yearRate(b, yoyByYear, yearIdx)
}

func yearRate(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, yearIdx int) decimal.Decimal {
	if yearIdx <= 1 {
		return b.FirstYearRate
	}
	cpiYear := b.PurchaseDate.Year() + yearIdx - 1
	yoy := lookupYoY(yoyByYear, cpiYear)
	if yoy.IsZero() {
		// No CPI data at all — fall back to FirstYearRate so the bond at
		// least keeps accruing. Calling code may surface a warning to the
		// user; we don't want a missing GUS refresh to zero out portfolios.
		return b.FirstYearRate
	}
	inflation := yoy.Sub(decimal.NewFromInt(100))
	return inflation.Add(b.Margin)
}

// lookupYoY returns the YoY for the requested year, clamping to the
// nearest known year if absent (latest known for future years, earliest for
// pre-history years). Returns zero only when the map is empty.
func lookupYoY(yoyByYear map[int]decimal.Decimal, year int) decimal.Decimal {
	if len(yoyByYear) == 0 {
		return decimal.Zero
	}
	if v, ok := yoyByYear[year]; ok {
		return v
	}
	earliest, latest := 0, 0
	for y := range yoyByYear {
		if earliest == 0 || y < earliest {
			earliest = y
		}
		if y > latest {
			latest = y
		}
	}
	if year > latest {
		return yoyByYear[latest]
	}
	return yoyByYear[earliest]
}

// bondTenorYears rounds tenor months up to whole years so a (hypothetical)
// non-12-month bond still has a sensible accrual cap.
func bondTenorYears(t BondType) int {
	m := t.MaturityMonths()
	y := m / 12
	if m%12 != 0 {
		y++
	}
	return y
}

// elapsedYearsAndFraction splits time since `start` into (completed
// anniversaries, partial-year fraction). The fraction is days elapsed
// since the last anniversary, divided by the length of the next
// anniversary year (handles leap years cleanly).
func elapsedYearsAndFraction(start, now time.Time) (int, decimal.Decimal) {
	if !now.After(start) {
		return 0, decimal.Zero
	}
	completed := 0
	for {
		next := start.AddDate(0, 12*(completed+1), 0)
		if next.After(now) {
			break
		}
		completed++
	}
	lastAnniversary := start.AddDate(0, 12*completed, 0)
	nextAnniversary := start.AddDate(0, 12*(completed+1), 0)
	span := nextAnniversary.Sub(lastAnniversary).Hours() / 24
	elapsed := now.Sub(lastAnniversary).Hours() / 24
	if span <= 0 {
		return completed, decimal.Zero
	}
	return completed, decimal.NewFromFloat(elapsed / span)
}

// MaturityDate is PurchaseDate + bond tenor.
func MaturityDate(b *TreasuryBond) time.Time {
	return b.PurchaseDate.AddDate(0, b.Type.MaturityMonths(), 0)
}

// YTMPoint is one (date, value, year_rate) sample in the bond's projected
// yield-to-maturity series. `Year` is 1-based; the sample is taken at the
// end of that year.
type YTMPoint struct {
	Year     int
	Date     time.Time
	Value    decimal.Decimal
	YearRate decimal.Decimal
}

// YieldToMaturity projects the bond's value at the end of each year until
// MaturityDate, using the same rate rules CurrentValue applies. Used by
// the /bonds page chart.
func YieldToMaturity(b *TreasuryBond, yoyByYear map[int]decimal.Decimal) []YTMPoint {
	if b.Type.MaturityMonths() <= 0 {
		return nil
	}
	years := bondTenorYears(b.Type)
	out := make([]YTMPoint, 0, years+1)
	value := b.FaceValue
	accruedPayout := decimal.Zero
	out = append(out, YTMPoint{
		Year:     0,
		Date:     b.PurchaseDate,
		Value:    b.FaceValue,
		YearRate: decimal.Zero,
	})
	for y := 1; y <= years; y++ {
		rate := yearRate(b, yoyByYear, y)
		if b.Capitalize {
			value = value.Mul(decimal.NewFromInt(1).Add(rate.Div(decimal.NewFromInt(100))))
		} else {
			accruedPayout = accruedPayout.Add(b.FaceValue.Mul(rate).Div(decimal.NewFromInt(100)))
		}
		sampleDate := b.PurchaseDate.AddDate(0, y*12, 0)
		display := value
		if !b.Capitalize {
			display = b.FaceValue.Add(accruedPayout)
		}
		out = append(out, YTMPoint{
			Year:     y,
			Date:     sampleDate,
			Value:    display.Round(2),
			YearRate: rate.Round(4),
		})
	}
	return out
}
