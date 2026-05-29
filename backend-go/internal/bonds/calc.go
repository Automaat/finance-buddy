package bonds

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
)

// CurrentValue projects the bond's worth at `now`. Accepts both an annual
// CPI YoY map (legacy fallback) and a monthly map; the monthly map wins
// when populated because it matches the Ministry's per-period rate-setting
// rule exactly.
//
// Rule per Ministry "list emisyjny" for EDO/COI/ROS/ROD:
//
//   - Year 1 accrues at FirstYearRate (fixed at emission).
//   - Year N (N >= 2) accrues at (CPI YoY published in the month preceding
//     period N's first month) - 100 + Margin. For a bond with period N
//     starting at date D, the reference month is D.month-2 (with year wrap)
//     — GUS publishes month X's reading mid-month X+1, so the YoY available
//     when the Ministry sets the rate ~14 days before period start is the
//     month-two-ago value.
//   - When the monthly map lacks the exact reference month, fall back to
//     the closest available month; when both maps are empty, fall back to
//     FirstYearRate so a missing CPI refresh can't zero out portfolios.
//   - Capitalize: each completed year multiplies the running value;
//     non-capitalized bonds add `face_value * year_rate` per completed year.
//   - Partial current year applies pro-rata at the same year's rate.
//
// Returns FaceValue when now is on/before PurchaseDate.
func CurrentValue(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, now time.Time) decimal.Decimal {
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
		rate := yearRate(b, yoyByYear, monthly, yearIdx)
		if b.Capitalize {
			value = value.Mul(decimal.NewFromInt(1).Add(rate.Div(decimal.NewFromInt(100))))
		} else {
			accruedPayout = accruedPayout.Add(b.FaceValue.Mul(rate).Div(decimal.NewFromInt(100)))
		}
	}
	if fraction.IsPositive() {
		rate := yearRate(b, yoyByYear, monthly, completedYears+1)
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
func YearRate(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, yearIdx int) decimal.Decimal {
	return yearRate(b, yoyByYear, monthly, yearIdx)
}

func yearRate(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal, yearIdx int) decimal.Decimal {
	if yearIdx <= 1 {
		return b.FirstYearRate
	}
	// Monthly map (Eurostat HICP) is the canonical source — matches the
	// Ministry's per-period rate-setting rule. Annual map kept as fallback
	// for environments where the monthly scheduler hasn't run yet.
	if yoy, ok := lookupMonthlyYoY(b, monthly, yearIdx); ok {
		inflation := yoy.Sub(decimal.NewFromInt(100))
		return inflation.Add(b.Margin)
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

// lookupMonthlyYoY returns the CPI YoY value the Ministry would use to
// set rate for bond year `yearIdx`. Period N starts at purchase + (N-1)
// years; reference month is two months before that period's first month
// (GUS releases month X's reading mid-month X+1, so the most recent value
// available 14 days before period start is the month-two-ago YoY).
//
// Falls back to the closest earlier month present in `monthly` (drift
// during transient publishing gaps). Returns (_, false) when the map is
// empty so the caller can chain through to the annual fallback.
func lookupMonthlyYoY(b *TreasuryBond, monthly map[cpi.YearMonth]decimal.Decimal, yearIdx int) (decimal.Decimal, bool) {
	if len(monthly) == 0 {
		return decimal.Decimal{}, false
	}
	periodStart := b.PurchaseDate.AddDate(0, 12*(yearIdx-1), 0)
	refYear, refMonth := periodStart.Year(), int(periodStart.Month())-2
	for refMonth < 1 {
		refYear--
		refMonth += 12
	}
	if v, ok := monthly[cpi.YearMonth{Year: refYear, Month: refMonth}]; ok {
		return v, true
	}
	// Walk back month-by-month for up to 12 months to bridge a publishing
	// gap. Beyond 12 we treat the data as missing and let the annual
	// fallback take over.
	for step := 1; step <= 12; step++ {
		y, m := refYear, refMonth-step
		for m < 1 {
			y--
			m += 12
		}
		if v, ok := monthly[cpi.YearMonth{Year: y, Month: m}]; ok {
			return v, true
		}
	}
	return decimal.Decimal{}, false
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
func YieldToMaturity(b *TreasuryBond, yoyByYear map[int]decimal.Decimal, monthly map[cpi.YearMonth]decimal.Decimal) []YTMPoint {
	if b.Type.MaturityMonths() <= 0 {
		return []YTMPoint{}
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
		rate := yearRate(b, yoyByYear, monthly, y)
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
