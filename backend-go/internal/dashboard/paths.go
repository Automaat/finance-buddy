package dashboard

import (
	"context"
	"sort"
	"time"
)

var allocationGroupMap = map[string]string{
	"stock": "stocks", "fund": "stocks", "etf": "stocks",
	"bond": "bonds", "gold": "gold",
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
		res.MetricCards = buildMetricCards(latestRows, shared, res.RetirementAccountValue, latestDate, nwForSavings, res.CurrentNetWorth)
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
			res.MetricCards = buildMetricCards(latestRows, shared, res.RetirementAccountValue, latestDate, nwForSavings, res.CurrentNetWorth)
			res.AllocationAnalysis = buildAllocationAnalysis(latestRows, shared.config)
		}
	}
	res.InvestmentTimeSeries = buildInvestmentTimeSeries(rows, shared.snapshots, shared.txns)
	res.WrapperTimeSeries = buildWrapperTimeSeries(rows, shared.snapshots, shared.txns)
	res.CategoryTimeSeries = buildCategoryTimeSeries(rows, shared.snapshots, shared.txns)
	res.TileDeltas = computeTileDeltas(perSnapshotTotals(rows))
	return res, nil
}

// buildMetricCards is the shared metric-card computation for both paths.
func buildMetricCards(
	latestRows []mergedRow,
	shared sharedInputs,
	retirementValue float64,
	latestDate time.Time,
	netWorthsForSavings []float64,
	currentNetWorth float64,
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

	swr, _ := cfg.WithdrawalRate.Float64()
	if swr <= 0 {
		swr = 0.04
	}
	mc := metricCards{
		PropertySQM:             propertySQM,
		EmergencyFundMonths:     emergencyMonths,
		RetirementIncomeMonthly: retirementValue * swr / 12,
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

	assignFIREMetrics(&mc, computeFIRE(latestRows, cfg, currentNetWorth))
	return mc
}

// assignFIREMetrics copies the FIRE/Coast-FIRE bundle into the metric-card
// struct. Pulled out of buildMetricCards to keep that function under the
// funlen threshold without obscuring the wiring.
func assignFIREMetrics(mc *metricCards, fire fireMetrics) {
	mc.AnnualExpenses = fire.AnnualExpenses
	mc.FIRENumber = fire.FIRENumber
	mc.FIProgress = fire.FIProgress
	mc.RunwayMonths = fire.RunwayMonths
	mc.WithdrawalRate = fire.WithdrawalRate
	mc.CoastFIRENumber = fire.CoastFIRENumber
	mc.CoastFIREGap = fire.CoastFIREGap
	mc.CoastFIRETargetAge = fire.CoastFIRETargetAge
	mc.ExpectedReturnRate = fire.ExpectedReturnRate
	mc.BaristaMonthlyIncome = fire.BaristaMonthlyIncome
	mc.BaristaAnnualGap = fire.BaristaAnnualGap
	mc.BaristaFIRENumber = fire.BaristaFIRENumber
	mc.BaristaFIProgress = fire.BaristaFIProgress
	mc.BaristaYearsToFI = fire.BaristaYearsToFI
	mc.BridgeYears = fire.BridgeYears
	mc.BridgeCapitalNeeded = fire.BridgeCapitalNeeded
	mc.BridgeLiquidCapital = fire.BridgeLiquidCapital
	mc.BridgeCapitalGap = fire.BridgeCapitalGap
	mc.LeanFIRENumber = fire.LeanFIRENumber
	mc.LeanFIProgress = fire.LeanFIProgress
	mc.FatFIRENumber = fire.FatFIRENumber
	mc.FatFIProgress = fire.FatFIProgress
	mc.MonthlySavings = fire.MonthlySavings
	mc.FIYearsRemaining = fire.FIYearsRemaining
	mc.FIProjectedDate = fire.FIProjectedDate
	mc.FireNetWorth = fire.FireNetWorth
	mc.FireExcludedValue = fire.FireExcludedValue
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
