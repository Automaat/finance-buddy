package dashboard

import (
	"math"
	"time"
)

// liquidCategories is the set of account categories counted as liquid for
// the runway metric — bank checking and savings only. Investments are
// excluded: a true runway figure assumes assets that can cover next
// month's bills without market timing or tax-event drag.
var liquidCategories = map[string]struct{}{
	"bank":           {},
	"saving_account": {},
}

// fireMetrics is the FIRE-tile bundle computed from app_config and the
// latest snapshot. Pointers are nil when the inputs aren't enough to
// produce a meaningful number (monthly_expenses == 0, withdrawal_rate == 0,
// or net_worth <= 0 for the progress percentage).
type fireMetrics struct {
	AnnualExpenses       *float64
	FIRENumber           *float64
	FIProgress           *float64
	RunwayMonths         *float64
	WithdrawalRate       *float64
	CoastFIRENumber      *float64
	CoastFIREGap         *float64
	CoastFIRETargetAge   *int
	ExpectedReturnRate   *float64
	BaristaMonthlyIncome *float64
	BaristaAnnualGap     *float64
	BaristaFIRENumber    *float64
	BaristaFIProgress    *float64
	BaristaYearsToFI     *float64
}

// computeFIRE turns app_config + the latest snapshot's rows into the FIRE
// + runway pair. Math:
//
//	annual_expenses = monthly_expenses × 12
//	fire_number     = annual_expenses / withdrawal_rate
//	fi_progress     = net_worth / fire_number × 100
//	runway_months   = liquid_assets / monthly_expenses
//
// liquid_assets is the sum of active asset-account values whose category
// is in liquidCategories at the latest snapshot.
func computeFIRE(latestRows []mergedRow, cfg AppConfig, netWorth float64) fireMetrics {
	var out fireMetrics

	monthlyExpenses, _ := cfg.MonthlyExpenses.Float64()
	withdrawalRate, _ := cfg.WithdrawalRate.Float64()
	if monthlyExpenses <= 0 {
		return out
	}

	annual := monthlyExpenses * 12
	out.AnnualExpenses = &annual

	if withdrawalRate > 0 {
		wr := withdrawalRate
		out.WithdrawalRate = &wr
		fire := annual / withdrawalRate
		out.FIRENumber = &fire
		if fire > 0 && netWorth > 0 {
			progress := netWorth / fire * 100
			out.FIProgress = &progress
		}
	}

	liquid := 0.0
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil || r.Category == nil {
			continue
		}
		if *r.AccType != "asset" {
			continue
		}
		if _, ok := liquidCategories[*r.Category]; ok {
			liquid += r.Value
		}
	}
	runway := liquid / monthlyExpenses
	out.RunwayMonths = &runway

	// Surface the configured expected return rate once when it's positive, so
	// the wire field is independent of whether Coast or Barista FIRE actually
	// consumed it. The UI uses this rate to label both years-to-FI captions.
	if er, _ := cfg.ExpectedReturnRate.Float64(); er > 0 {
		rate := er
		out.ExpectedReturnRate = &rate
	}

	now := time.Now().UTC()
	addCoastFIRE(&out, cfg, netWorth, now)
	addBaristaFIRE(&out, cfg, netWorth)
	return out
}

// addBaristaFIRE fills the Barista FIRE fields when a part-time monthly
// income is configured and the existing FIRE inputs are valid. Math:
//
//	barista_annual_gap   = max(0, annual_expenses − barista_annual_income)
//	barista_fire_number  = barista_annual_gap ÷ withdrawal_rate
//	barista_fi_progress  = net_worth ÷ barista_fire_number × 100
//	barista_years_to_fi  = ln(barista_fire_number ÷ net_worth) ÷ ln(1 + r)
//
// The years-to-FI projection assumes no future contributions — compounding
// alone of today's net worth at the expected return rate. This is the most
// conservative projection that doesn't require yet-another configured
// "monthly savings" input, and it lines up with how Coast FIRE uses
// expected_return_rate.
func addBaristaFIRE(out *fireMetrics, cfg AppConfig, netWorth float64) {
	if out.AnnualExpenses == nil || out.WithdrawalRate == nil {
		return
	}
	if cfg.BaristaMonthlyIncome == nil {
		return
	}
	monthly, _ := cfg.BaristaMonthlyIncome.Float64()
	if monthly < 0 {
		return
	}
	baristaAnnual := monthly * 12
	bm := monthly
	out.BaristaMonthlyIncome = &bm

	gap := *out.AnnualExpenses - baristaAnnual
	if gap < 0 {
		gap = 0
	}
	out.BaristaAnnualGap = &gap

	fire := gap / *out.WithdrawalRate
	out.BaristaFIRENumber = &fire
	if fire > 0 && netWorth > 0 {
		progress := netWorth / fire * 100
		out.BaristaFIProgress = &progress
	}

	// Years-to-FI is only meaningful when (a) we still have ground to cover
	// (net_worth < fire), (b) net worth is positive (log undefined at 0/
	// negative), and (c) the projection has a positive growth rate.
	if fire <= 0 || netWorth <= 0 || netWorth >= fire {
		return
	}
	expectedReturn, _ := cfg.ExpectedReturnRate.Float64()
	if expectedReturn <= 0 {
		return
	}
	years := math.Log(fire/netWorth) / math.Log(1+expectedReturn)
	out.BaristaYearsToFI = &years
}

// addCoastFIRE fills the Coast FIRE fields when the inputs are sufficient:
// a configured target age, a positive expected return rate, a positive
// FIRE number, and a target age strictly greater than the current age.
//
// Coast FIRE math:
//
//	years_remaining     = target_age − current_age (whole years)
//	coast_fire_number   = fire_number ÷ (1 + expected_return)^years_remaining
//	coast_fire_gap      = coast_fire_number − net_worth
//
// coast_fire_number is "the capital needed today such that, with no further
// contributions, compounding alone reaches the FIRE number by the target
// age." Positive gap = shortfall; negative gap = surplus (already Coast-FI).
func addCoastFIRE(out *fireMetrics, cfg AppConfig, netWorth float64, now time.Time) {
	if out.FIRENumber == nil || *out.FIRENumber <= 0 {
		return
	}
	if cfg.CoastFIRETargetAge == nil {
		return
	}
	targetAge := *cfg.CoastFIRETargetAge
	if cfg.BirthDate.IsZero() {
		return
	}
	expectedReturn, _ := cfg.ExpectedReturnRate.Float64()
	if expectedReturn <= 0 {
		return
	}
	currentAge := yearsBetween(cfg.BirthDate, now)
	if targetAge <= currentAge {
		return
	}
	years := float64(targetAge - currentAge)
	coast := *out.FIRENumber / math.Pow(1+expectedReturn, years)
	gap := coast - netWorth
	out.CoastFIRENumber = &coast
	out.CoastFIREGap = &gap
	ta := targetAge
	out.CoastFIRETargetAge = &ta
}

// yearsBetween returns whole years from birth to now using the same
// 365.25-day approximation as the config validator (matches the frontend
// "current age" display).
func yearsBetween(birth, now time.Time) int {
	days := now.UTC().Truncate(24*time.Hour).Sub(birth.UTC().Truncate(24*time.Hour)) /
		(24 * time.Hour)
	if days < 0 {
		return 0
	}
	return int(float64(days) / 365.25)
}
