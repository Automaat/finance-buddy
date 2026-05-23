package dashboard

import (
	"context"
	"fmt"
	"sort"
	"time"
)

var allocationGroupMap = map[string]string{
	"stock": "stocks", "fund": "stocks", "etf": "stocks",
	"bond": "bonds", "gold": "gold",
}

// aggregatePointsBySnapshot collapses snapshot_aggregates rows (which are
// per-(snapshot, owner)) into per-snapshot net-worth/assets/liabilities
// points sorted by date. Missing snapshot dates cause a fail-fast error.
func aggregatePointsBySnapshot(aggRows []AggregateRow, snapshotDate map[int]time.Time) ([]netWorthPoint, error) {
	type snapAgg struct {
		nw, assets, liabs float64
	}
	bySnap := map[int]snapAgg{}
	for _, r := range aggRows {
		nw, _ := r.NetWorth.Float64()
		ta, _ := r.TotalAssets.Float64()
		tl, _ := r.TotalLiabilities.Float64()
		agg := bySnap[r.SnapshotID]
		agg.nw += nw
		agg.assets += ta
		agg.liabs += tl
		bySnap[r.SnapshotID] = agg
	}
	points := make([]netWorthPoint, 0, len(bySnap))
	for sid, agg := range bySnap {
		date, ok := snapshotDate[sid]
		if !ok {
			// A snapshot_aggregates row references a snapshot that doesn't
			// exist — partial restore / corruption. Fail fast, as Python's
			// snap_date[sid] KeyError does, rather than skewing the history
			// with a zero date.
			return nil, fmt.Errorf("aggregate references missing snapshot %d", sid)
		}
		points = append(points, netWorthPoint{
			Date:        date,
			Value:       agg.nw,
			Assets:      agg.assets,
			Liabilities: agg.liabs,
			SnapshotID:  sid,
		})
	}
	// Date primary, SnapshotID tiebreaker — keeps order stable when two
	// snapshots share a calendar day.
	sort.Slice(points, func(i, j int) bool {
		if !points[i].Date.Equal(points[j].Date) {
			return points[i].Date.Before(points[j].Date)
		}
		return points[i].SnapshotID < points[j].SnapshotID
	})
	return points, nil
}

// computeFromAggregates ports _get_dashboard_data_from_aggregates.
func computeFromAggregates(ctx context.Context, s *Store, aggRows []AggregateRow) (result, error) {
	shared, err := loadShared(ctx, s)
	if err != nil {
		return result{}, err
	}

	points, err := aggregatePointsBySnapshot(aggRows, shared.snapshotDate)
	if err != nil {
		return result{}, err
	}

	res := emptyResult()
	res.NetWorthHistory = points
	if len(points) > 0 {
		res.CurrentNetWorth = points[len(points)-1].Value
	}
	lastMonthNW := 0.0
	if len(points) > 1 {
		lastMonthNW = points[len(points)-2].Value
	}
	res.ChangeVsLastMonth = res.CurrentNetWorth - lastMonthNW

	// latest snapshot = highest snapshot_id among aggregate rows.
	latestSID := 0
	for _, r := range aggRows {
		if r.SnapshotID > latestSID {
			latestSID = r.SnapshotID
		}
	}
	for _, r := range aggRows {
		if r.SnapshotID != latestSID {
			continue
		}
		ta, _ := r.TotalAssets.Float64()
		tl, _ := r.TotalLiabilities.Float64()
		res.TotalAssets += ta
		res.TotalLiabilities += tl
		for _, item := range r.Allocation {
			res.Allocation = append(res.Allocation, allocationItem{
				Category: item.Category, OwnerUserID: r.OwnerUserID, Value: item.Value,
			})
		}
	}
	sortAllocation(res.Allocation)

	// latest snapshot's merged rows drive metric cards + allocation analysis.
	latestValues, err := s.SnapshotValuesFor(ctx, latestSID)
	if err != nil {
		return result{}, err
	}
	latestRows := buildMergedRows(latestValues, shared.snapshotDate, shared.accounts, shared.assetIDs)
	res.RetirementAccountValue = retirementValueOf(latestRows)

	if shared.hasConfig && len(latestRows) > 0 {
		latestDate := shared.snapshotDate[latestSID]
		nwForSavings := aggregateMonthlyNetWorths(aggRows)
		res.MetricCards = buildMetricCards(latestRows, shared, res.RetirementAccountValue, latestDate, nwForSavings)
		res.AllocationAnalysis = buildAllocationAnalysis(latestRows, shared.config)
	}

	// time series + tile deltas.
	if hasInvestment(shared.accountList) {
		allValues, err := s.AllSnapshotValues(ctx)
		if err != nil {
			return result{}, err
		}
		allRows := buildMergedRows(allValues, shared.snapshotDate, shared.accounts, shared.assetIDs)
		res.InvestmentTimeSeries = buildInvestmentTimeSeries(allRows, shared.snapshots, shared.txns)
		res.WrapperTimeSeries = buildWrapperTimeSeries(allRows, shared.snapshots, shared.txns)
		res.CategoryTimeSeries = buildCategoryTimeSeries(allRows, shared.snapshots, shared.txns)
		res.TileDeltas = computeTileDeltas(perSnapshotTotals(allRows))
	} else {
		res.TileDeltas = computeTileDeltas(synthTotalsFromAggregates(aggRows, shared.snapshotDate))
	}
	return res, nil
}

// computeRaw ports _get_dashboard_data_raw.
func computeRaw(ctx context.Context, s *Store) (result, error) {
	shared, err := loadShared(ctx, s)
	if err != nil {
		return result{}, err
	}
	allValues, err := s.AllSnapshotValues(ctx)
	if err != nil {
		return result{}, err
	}
	rows := buildMergedRows(allValues, shared.snapshotDate, shared.accounts, shared.assetIDs)
	res := emptyResult()
	if len(rows) == 0 {
		return res, nil
	}

	// net worth grouped by date — track assets/liabilities separately so
	// the waterfall chart can show per-month contribution by side.
	type byDateAgg struct {
		nw, assets, liabs float64
		snapshotID        int
	}
	// Value-typed map so nilaway can verify every read is non-nil — the
	// aggregate is updated by reassignment after each row.
	byDate := map[time.Time]byDateAgg{}
	for _, r := range rows {
		agg := byDate[r.Date]
		if r.AccountID != nil && r.AccType != nil {
			if *r.AccType == "asset" {
				agg.assets += r.Value
			} else {
				agg.liabs += r.Value
			}
		} else if r.AssetID != nil && r.AssetName != nil {
			agg.assets += r.Value
		}
		agg.nw += r.signedValue()
		// Lowest snapshot_id for a given date wins — multiple snapshots on
		// one date is unusual; the click-through points at the earliest-
		// created of the duplicates.
		if agg.snapshotID == 0 || r.SnapshotID < agg.snapshotID {
			agg.snapshotID = r.SnapshotID
		}
		byDate[r.Date] = agg
	}
	dates := make([]time.Time, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	res.NetWorthHistory = make([]netWorthPoint, 0, len(dates))
	for _, d := range dates {
		agg := byDate[d]
		res.NetWorthHistory = append(res.NetWorthHistory, netWorthPoint{
			Date:        d,
			Value:       agg.nw,
			Assets:      agg.assets,
			Liabilities: agg.liabs,
			SnapshotID:  agg.snapshotID,
		})
	}
	// Convenience alias for the old single-value lookup below.
	nwByDate := map[time.Time]float64{}
	for d, agg := range byDate {
		nwByDate[d] = agg.nw
	}
	if len(dates) > 0 {
		res.CurrentNetWorth = nwByDate[dates[len(dates)-1]]
	}
	lastMonth := 0.0
	if len(dates) > 1 {
		lastMonth = nwByDate[dates[len(dates)-2]]
	}
	res.ChangeVsLastMonth = res.CurrentNetWorth - lastMonth

	latestSID, latestDate, found := latestSnapshotByDate(rows)
	if found {
		var latestRows []mergedRow
		for _, r := range rows {
			if r.SnapshotID == latestSID {
				latestRows = append(latestRows, r)
			}
		}
		res.TotalAssets, res.TotalLiabilities = rawTotals(latestRows)
		res.Allocation = rawAllocation(latestRows)
		res.RetirementAccountValue = retirementValueOf(latestRows)
		if shared.hasConfig {
			nwForSavings := rawMonthlyNetWorths(rows, shared.snapshots)
			res.MetricCards = buildMetricCards(latestRows, shared, res.RetirementAccountValue, latestDate, nwForSavings)
			res.AllocationAnalysis = buildAllocationAnalysis(latestRows, shared.config)
		}
	}
	res.InvestmentTimeSeries = buildInvestmentTimeSeries(rows, shared.snapshots, shared.txns)
	res.WrapperTimeSeries = buildWrapperTimeSeries(rows, shared.snapshots, shared.txns)
	res.CategoryTimeSeries = buildCategoryTimeSeries(rows, shared.snapshots, shared.txns)
	res.TileDeltas = computeTileDeltas(perSnapshotTotals(rows))
	return res, nil
}

// retirementValueOf sums latest-snapshot account-asset rows with purpose=retirement.
func retirementValueOf(latestRows []mergedRow) float64 {
	total := 0.0
	for _, r := range latestRows {
		if r.AccountID != nil && r.AccType != nil && *r.AccType == "asset" &&
			r.Purpose != nil && *r.Purpose == "retirement" {
			total += r.Value
		}
	}
	return total
}

func rawTotals(latestRows []mergedRow) (float64, float64) {
	assets := 0.0
	liabilities := 0.0
	for _, r := range latestRows {
		switch {
		case r.AssetID != nil:
			assets += r.Value
		case r.AccountID != nil && r.AccType != nil:
			switch *r.AccType {
			case "asset":
				assets += r.Value
			case "liability":
				liabilities += r.Value
			}
		}
	}
	return assets, liabilities
}

func rawAllocation(latestRows []mergedRow) []allocationItem {
	// owner is *int — pack it into a comparable key. shared==true marks a
	// nil owner_user_id (jointly owned).
	type key struct {
		cat    string
		owner  int
		shared bool
	}
	sums := map[key]float64{}
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil || *r.AccType != "asset" {
			continue
		}
		if r.Category == nil {
			continue
		}
		k := key{cat: *r.Category}
		if r.OwnerUserID == nil {
			k.shared = true
		} else {
			k.owner = *r.OwnerUserID
		}
		sums[k] += r.Value
	}
	out := make([]allocationItem, 0, len(sums))
	for k, v := range sums {
		item := allocationItem{Category: k.cat, Value: v}
		if !k.shared {
			id := k.owner
			item.OwnerUserID = &id
		}
		out = append(out, item)
	}
	sortAllocation(out)
	return out
}

// aggregateMonthlyNetWorths ports _compute_savings_rate_from_aggregates'
// month-bucketing: latest snapshot per month, sorted chronologically.
func aggregateMonthlyNetWorths(aggRows []AggregateRow) []float64 {
	snapNW := map[int]float64{}
	snapMonth := map[int]time.Time{}
	for _, r := range aggRows {
		nw, _ := r.NetWorth.Float64()
		snapNW[r.SnapshotID] += nw
		snapMonth[r.SnapshotID] = r.Month
	}
	monthLatestSID := map[time.Time]int{}
	for sid, month := range snapMonth {
		if cur, ok := monthLatestSID[month]; !ok || sid > cur {
			monthLatestSID[month] = sid
		}
	}
	months := make([]time.Time, 0, len(monthLatestSID))
	for m := range monthLatestSID {
		months = append(months, m)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Before(months[j]) })
	out := make([]float64, 0, len(months))
	for _, m := range months {
		out = append(out, snapNW[monthLatestSID[m]])
	}
	return out
}

// rawMonthlyNetWorths ports the raw savings-rate input: net worth of the
// last-4 snapshots (by the snapshots-ordered-by-date tail).
func rawMonthlyNetWorths(rows []mergedRow, snapshots []SnapshotMeta) []float64 {
	nwBySnap := map[int]float64{}
	for _, r := range rows {
		nwBySnap[r.SnapshotID] += r.signedValue()
	}
	out := make([]float64, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, nwBySnap[s.ID])
	}
	return out
}

// synthTotalsFromAggregates ports the no-investment tile-delta branch — one
// synthetic totals row per snapshot from the aggregate sums.
func synthTotalsFromAggregates(aggRows []AggregateRow, snapshotDate map[int]time.Time) []snapshotTotals {
	type acc struct{ assets, liabilities float64 }
	bySnap := map[int]*acc{}
	for _, r := range aggRows {
		a, ok := bySnap[r.SnapshotID]
		if !ok {
			a = &acc{}
			bySnap[r.SnapshotID] = a
		}
		ta, _ := r.TotalAssets.Float64()
		tl, _ := r.TotalLiabilities.Float64()
		a.assets += ta
		a.liabilities += tl
	}
	out := make([]snapshotTotals, 0, len(bySnap))
	for sid, a := range bySnap {
		out = append(out, snapshotTotals{
			SnapshotID:  sid,
			Date:        snapshotDate[sid],
			Assets:      a.assets,
			Liabilities: a.liabilities,
			NetWorth:    a.assets - a.liabilities,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		return out[i].SnapshotID < out[j].SnapshotID
	})
	return out
}

// buildMetricCards is the shared metric-card computation for both paths.
func buildMetricCards(
	latestRows []mergedRow,
	shared sharedInputs,
	retirementValue float64,
	latestDate time.Time,
	netWorthsForSavings []float64,
) metricCards {
	cfg := shared.config

	realEstateIDs := map[int]struct{}{}
	totalPropertySQM := 0.0
	for _, a := range shared.accountList {
		if a.Category == "real_estate" {
			realEstateIDs[a.ID] = struct{}{}
			if a.SquareMeters != nil {
				sqm, _ := a.SquareMeters.Float64()
				totalPropertySQM += sqm
			}
		}
	}
	propertyValue := 0.0
	mortgageRemaining := 0.0
	emergencyFund := 0.0
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil {
			continue
		}
		if *r.AccType == "asset" {
			if _, re := realEstateIDs[*r.AccountID]; re {
				propertyValue += r.Value
			}
			if r.Purpose != nil && *r.Purpose == "emergency_fund" {
				emergencyFund += r.Value
			}
		}
		if *r.AccType == "liability" && r.Category != nil &&
			(*r.Category == "housing" || *r.Category == "mortgage") {
			mortgageRemaining += r.Value
		}
	}
	propertySQM := 0.0
	if propertyValue > 0 {
		equity := (propertyValue - mortgageRemaining) / propertyValue
		if equity < 0 {
			equity = 0
		}
		propertySQM = totalPropertySQM * equity
	}
	monthlyExpenses, _ := cfg.MonthlyExpenses.Float64()
	emergencyMonths := 0.0
	if monthlyExpenses > 0 {
		emergencyMonths = emergencyFund / monthlyExpenses
	}
	monthlyMortgage, _ := cfg.MonthlyMortgagePayment.Float64()
	mortgageMonthsLeft := 0
	if monthlyMortgage > 0 {
		mortgageMonthsLeft = int(abs(mortgageRemaining) / monthlyMortgage)
	}
	investmentContributions := 0.0
	for _, t := range shared.txns {
		if t.Purpose != nil && *t.Purpose == "retirement" && !t.Date.After(latestDate) {
			investmentContributions += t.SignedAmount
		}
	}

	mc := metricCards{
		PropertySQM:             propertySQM,
		EmergencyFundMonths:     emergencyMonths,
		RetirementIncomeMonthly: retirementValue * 0.04 / 12,
		MortgageRemaining:       mortgageRemaining,
		MortgageMonthsLeft:      mortgageMonthsLeft,
		MortgageYearsLeft:       float64(mortgageMonthsLeft) / 12,
		RetirementTotal:         retirementValue,
		InvestmentContributions: investmentContributions,
		InvestmentReturns:       retirementValue - investmentContributions,
		SavingsRate:             savingsRate(netWorthsForSavings, shared.salaries),
	}
	if monthlyMortgage > 0 && len(shared.salaries) > 0 && shared.salaries[0] != 0 {
		dti := monthlyMortgage / shared.salaries[0] * 100
		mc.DebtToIncomeRatio = &dti
	}
	if len(shared.salaries) > 0 && shared.salaries[0] != 0 {
		work := shared.salaries[0] / monthlyWorkHours
		life := shared.salaries[0] / monthlyLifeHours
		mc.HourOfWorkCost = &work
		mc.HourOfLifeCost = &life
	}
	return mc
}

// buildAllocationAnalysis is the shared allocation-analysis computation.
func buildAllocationAnalysis(latestRows []mergedRow, cfg AppConfig) allocationAnalysis {
	out := allocationAnalysis{
		ByCategory:  []allocationBreakdown{},
		ByWrapper:   []wrapperBreakdown{},
		Rebalancing: []rebalancingSuggestion{},
	}
	allocCategories := map[string]struct{}{"stock": {}, "bond": {}, "fund": {}, "etf": {}, "gold": {}}

	totalInvestment := 0.0
	byGroup := map[string]float64{}
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil || *r.AccType != "asset" || r.Category == nil {
			continue
		}
		if _, ok := allocCategories[*r.Category]; !ok {
			continue
		}
		totalInvestment += r.Value
		group := allocationGroupMap[*r.Category]
		if group == "" {
			group = "other"
		}
		byGroup[group] += r.Value
	}
	out.TotalInvestmentValue = totalInvestment

	targets := []struct {
		name   string
		target float64
	}{
		{"stocks", float64(cfg.AllocationStocks)},
		{"bonds", float64(cfg.AllocationBonds)},
		{"gold", float64(cfg.AllocationGold)},
	}
	for _, t := range targets {
		current := byGroup[t.name]
		pct := 0.0
		if totalInvestment > 0 {
			pct = current / totalInvestment * 100
		}
		out.ByCategory = append(out.ByCategory, allocationBreakdown{
			Category:          t.name,
			CurrentValue:      current,
			CurrentPercentage: pct,
			TargetPercentage:  t.target,
			Difference:        pct - t.target,
		})
	}

	allInvCategories := map[string]struct{}{
		"stock": {}, "bond": {}, "fund": {}, "etf": {}, "gold": {}, "ppk": {},
	}
	wrapperSums := map[string]float64{}
	allInvTotal := 0.0
	for _, r := range latestRows {
		if r.AccountID == nil || r.AccType == nil || *r.AccType != "asset" || r.Category == nil {
			continue
		}
		if _, ok := allInvCategories[*r.Category]; !ok {
			continue
		}
		wrapper := "Regular"
		if r.Wrapper != nil && *r.Wrapper != "" {
			wrapper = *r.Wrapper
		}
		wrapperSums[wrapper] += r.Value
		allInvTotal += r.Value
	}
	wrapperNames := make([]string, 0, len(wrapperSums))
	for w := range wrapperSums {
		wrapperNames = append(wrapperNames, w)
	}
	sort.Strings(wrapperNames)
	for _, w := range wrapperNames {
		pct := 0.0
		if allInvTotal > 0 {
			pct = wrapperSums[w] / allInvTotal * 100
		}
		out.ByWrapper = append(out.ByWrapper, wrapperBreakdown{
			Wrapper: w, Value: wrapperSums[w], Percentage: pct,
		})
	}

	for _, b := range out.ByCategory {
		if b.Difference >= -1 {
			continue
		}
		targetValue := totalInvestment * b.TargetPercentage / 100
		targetPct := b.TargetPercentage / 100
		if targetPct >= 1 {
			continue
		}
		amountToAdd := (targetValue - b.CurrentValue) / (1 - targetPct)
		if amountToAdd > 100 {
			out.Rebalancing = append(out.Rebalancing, rebalancingSuggestion{
				Category: b.Category, Action: "buy", Amount: amountToAdd,
			})
		}
	}
	return out
}
