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
	AnnualExpenses     *float64
	FIRENumber         *float64
	FIProgress         *float64
	RunwayMonths       *float64
	WithdrawalRate     *float64
	CoastFIRENumber    *float64
	CoastFIREGap       *float64
	CoastFIRETargetAge *int
	ExpectedReturnRate *float64
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

	addCoastFIRE(&out, cfg, netWorth, time.Now().UTC())
	return out
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
	er := expectedReturn
	out.ExpectedReturnRate = &er
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
