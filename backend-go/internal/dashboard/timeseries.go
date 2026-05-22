package dashboard

import (
	"sort"
	"time"
)

// timeSeriesPoint mirrors InvestmentTimeSeriesPoint.
type timeSeriesPoint struct {
	Date          time.Time
	Value         float64
	Contributions float64
	Returns       float64
}

// txnWithAccount is a transaction joined to its account — carries the
// signed amount + the joined account's category/wrapper/purpose.
type txnWithAccount struct {
	Date         time.Time
	SignedAmount float64
	Category     *string
	Wrapper      *string
	Purpose      *string
}

// buildTxnsWithAccounts joins active transactions to accounts and computes
// the signed amount (withdrawal -> negative). Mirrors the
// transactions_with_accounts_df construction.
func buildTxnsWithAccounts(txns []Transaction, accounts map[int]Account) []txnWithAccount {
	out := make([]txnWithAccount, 0, len(txns))
	for _, t := range txns {
		amt, _ := t.Amount.Float64()
		sign := 1.0
		if t.TransactionType != nil && *t.TransactionType == "withdrawal" {
			sign = -1.0
		}
		row := txnWithAccount{Date: t.Date, SignedAmount: amt * sign}
		if acc, ok := accounts[t.AccountID]; ok {
			c := acc.Category
			p := acc.Purpose
			row.Category = &c
			row.Purpose = &p
			row.Wrapper = acc.AccountWrapper
		}
		out = append(out, row)
	}
	return out
}

// cumContributions ports _cum_contributions_per_snapshot: for each snapshot
// date, the cumulative signed_amount of transactions on/before it.
func cumContributions(txns []txnWithAccount, snapDates []time.Time) []float64 {
	out := make([]float64, len(snapDates))
	if len(txns) == 0 {
		return out
	}
	sorted := make([]txnWithAccount, len(txns))
	copy(sorted, txns)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })
	cum := make([]float64, len(sorted))
	running := 0.0
	for i, t := range sorted {
		running += t.SignedAmount
		cum[i] = running
	}
	for i, d := range snapDates {
		// number of transactions with date <= d, minus 1 (searchsorted right)
		idx := sort.Search(len(sorted), func(k int) bool { return sorted[k].Date.After(d) }) - 1
		if idx >= 0 {
			out[i] = cum[idx]
		}
	}
	return out
}

// buildSeries ports _build_series: per-snapshot summed value + cumulative
// contributions for the rows/transactions matching the supplied predicates.
func buildSeries(
	rows []mergedRow,
	snapshots []SnapshotMeta,
	txns []txnWithAccount,
	valueMatch func(mergedRow) bool,
	txnMatch func(txnWithAccount) bool,
) []timeSeriesPoint {
	if len(rows) == 0 || len(snapshots) == 0 {
		return []timeSeriesPoint{}
	}
	snaps := make([]SnapshotMeta, len(snapshots))
	copy(snaps, snapshots)
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].Date.Before(snaps[j].Date) })

	valBySnap := map[int]float64{}
	for _, r := range rows {
		if valueMatch(r) {
			valBySnap[r.SnapshotID] += r.Value
		}
	}
	snapDates := make([]time.Time, len(snaps))
	for i, s := range snaps {
		snapDates[i] = s.Date
	}
	var contributions []float64
	if txnMatch != nil && len(txns) > 0 {
		filtered := []txnWithAccount{}
		for _, t := range txns {
			if txnMatch(t) {
				filtered = append(filtered, t)
			}
		}
		contributions = cumContributions(filtered, snapDates)
	} else {
		contributions = make([]float64, len(snaps))
	}
	out := make([]timeSeriesPoint, 0, len(snaps))
	for i, s := range snaps {
		value := valBySnap[s.ID]
		c := contributions[i]
		out = append(out, timeSeriesPoint{
			Date: s.Date, Value: value, Contributions: c, Returns: value - c,
		})
	}
	return out
}

func isInvestmentCategory(c *string) bool {
	if c == nil {
		return false
	}
	_, ok := investmentCategories[*c]
	return ok
}

// buildInvestmentTimeSeries ports build_investment_time_series.
func buildInvestmentTimeSeries(rows []mergedRow, snaps []SnapshotMeta, txns []txnWithAccount) []timeSeriesPoint {
	valueMatch := func(r mergedRow) bool {
		return r.AccountID != nil && r.AccType != nil && *r.AccType == "asset" &&
			isInvestmentCategory(r.Category)
	}
	var txnMatch func(txnWithAccount) bool
	if len(txns) > 0 {
		txnMatch = func(t txnWithAccount) bool { return isInvestmentCategory(t.Category) }
	}
	return buildSeries(rows, snaps, txns, valueMatch, txnMatch)
}

// buildWrapperTimeSeries ports build_wrapper_time_series — IKE/IKZE/PPK.
func buildWrapperTimeSeries(rows []mergedRow, snaps []SnapshotMeta, txns []txnWithAccount) map[string][]timeSeriesPoint {
	out := map[string][]timeSeriesPoint{}
	for _, wrapper := range []string{"IKE", "IKZE", "PPK"} {
		w := wrapper
		valueMatch := func(r mergedRow) bool {
			return r.AccountID != nil && r.AccType != nil && *r.AccType == "asset" &&
				isInvestmentCategory(r.Category) &&
				r.Wrapper != nil && *r.Wrapper == w
		}
		var txnMatch func(txnWithAccount) bool
		if len(txns) > 0 {
			txnMatch = func(t txnWithAccount) bool {
				return isInvestmentCategory(t.Category) && t.Wrapper != nil && *t.Wrapper == w
			}
		}
		out[wrapper] = buildSeries(rows, snaps, txns, valueMatch, txnMatch)
	}
	return out
}

// buildCategoryTimeSeries ports build_category_time_series — stock/bond groups.
func buildCategoryTimeSeries(rows []mergedRow, snaps []SnapshotMeta, txns []txnWithAccount) map[string][]timeSeriesPoint {
	groups := map[string]map[string]struct{}{
		"stock": {"stock": {}, "fund": {}, "etf": {}},
		"bond":  {"bond": {}},
	}
	out := map[string][]timeSeriesPoint{}
	for group, cats := range groups {
		c := cats
		inGroup := func(cat *string) bool {
			if cat == nil {
				return false
			}
			_, ok := c[*cat]
			return ok
		}
		valueMatch := func(r mergedRow) bool {
			return r.AccountID != nil && r.AccType != nil && *r.AccType == "asset" && inGroup(r.Category)
		}
		var txnMatch func(txnWithAccount) bool
		if len(txns) > 0 {
			txnMatch = func(t txnWithAccount) bool { return inGroup(t.Category) }
		}
		out[group] = buildSeries(rows, snaps, txns, valueMatch, txnMatch)
	}
	return out
}
