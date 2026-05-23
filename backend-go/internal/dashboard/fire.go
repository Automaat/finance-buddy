package dashboard

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
	AnnualExpenses *float64
	FIRENumber     *float64
	FIProgress     *float64
	RunwayMonths   *float64
	WithdrawalRate *float64
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
	return out
}
