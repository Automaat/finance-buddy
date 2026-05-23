package dashboard

import (
	"context"
	"sort"
	"time"
)

// netWorthPoint mirrors NetWorthPoint.
type netWorthPoint struct {
	Date  time.Time
	Value float64
}

// allocationItem mirrors AllocationItem.
type allocationItem struct {
	Category    string
	OwnerUserID *int
	Value       float64
}

// metricCards mirrors MetricCards. The four ratio fields are nil-able.
type metricCards struct {
	PropertySQM             float64
	EmergencyFundMonths     float64
	RetirementIncomeMonthly float64
	MortgageRemaining       float64
	MortgageMonthsLeft      int
	MortgageYearsLeft       float64
	RetirementTotal         float64
	InvestmentContributions float64
	InvestmentReturns       float64
	SavingsRate             *float64
	DebtToIncomeRatio       *float64
	HourOfWorkCost          *float64
	HourOfLifeCost          *float64
}

type allocationBreakdown struct {
	Category          string
	CurrentValue      float64
	CurrentPercentage float64
	TargetPercentage  float64
	Difference        float64
}

type wrapperBreakdown struct {
	Wrapper    string
	Value      float64
	Percentage float64
}

type rebalancingSuggestion struct {
	Category string
	Action   string
	Amount   float64
}

type allocationAnalysis struct {
	ByCategory           []allocationBreakdown
	ByWrapper            []wrapperBreakdown
	Rebalancing          []rebalancingSuggestion
	TotalInvestmentValue float64
}

// result is the fully-computed dashboard, ready for the handler to serialize.
type result struct {
	NetWorthHistory        []netWorthPoint
	CurrentNetWorth        float64
	ChangeVsLastMonth      float64
	TotalAssets            float64
	TotalLiabilities       float64
	Allocation             []allocationItem
	RetirementAccountValue float64
	MetricCards            metricCards
	AllocationAnalysis     allocationAnalysis
	InvestmentTimeSeries   []timeSeriesPoint
	WrapperTimeSeries      map[string][]timeSeriesPoint
	CategoryTimeSeries     map[string][]timeSeriesPoint
	TileDeltas             tileDeltas
}

// Compute runs the dashboard. Aggregate-backed when snapshot_aggregates has
// rows; raw fallback otherwise. Mirrors get_dashboard_data.
func Compute(ctx context.Context, s *Store) (result, error) {
	aggRows, err := s.LoadAggregateRows(ctx)
	if err != nil {
		return result{}, err
	}
	if len(aggRows) == 0 {
		return computeRaw(ctx, s)
	}
	return computeFromAggregates(ctx, s, aggRows)
}

func emptyResult() result {
	return result{
		NetWorthHistory:      []netWorthPoint{},
		Allocation:           []allocationItem{},
		AllocationAnalysis:   allocationAnalysis{ByCategory: []allocationBreakdown{}, ByWrapper: []wrapperBreakdown{}, Rebalancing: []rebalancingSuggestion{}},
		InvestmentTimeSeries: []timeSeriesPoint{},
		WrapperTimeSeries:    map[string][]timeSeriesPoint{"IKE": {}, "IKZE": {}, "PPK": {}},
		CategoryTimeSeries:   map[string][]timeSeriesPoint{"stock": {}, "bond": {}},
	}
}

// sharedInputs bundles the tables both paths need.
//
// accounts/assetIDs intentionally include soft-deleted rows so historical
// snapshot values keep their metadata after delete (issue #394). accountList
// is the active-only slice for "current state" reads (real-estate property
// counts, hasInvestment, allocation totals).
type sharedInputs struct {
	accounts     map[int]Account
	accountList  []Account
	assetIDs     map[int]struct{}
	snapshots    []SnapshotMeta
	snapshotDate map[int]time.Time
	config       AppConfig
	hasConfig    bool
	txns         []txnWithAccount
	salaries     []float64
}

func loadShared(ctx context.Context, s *Store) (sharedInputs, error) {
	var in sharedInputs
	allAccs, err := s.AllAccounts(ctx)
	if err != nil {
		return in, err
	}
	in.accounts = map[int]Account{}
	in.accountList = make([]Account, 0, len(allAccs))
	for _, a := range allAccs {
		in.accounts[a.ID] = a
		if a.IsActive {
			in.accountList = append(in.accountList, a)
		}
	}
	in.assetIDs, err = s.AllAssetIDs(ctx)
	if err != nil {
		return in, err
	}
	in.snapshots, err = s.AllSnapshots(ctx)
	if err != nil {
		return in, err
	}
	in.snapshotDate = map[int]time.Time{}
	for _, m := range in.snapshots {
		in.snapshotDate[m.ID] = m.Date
	}
	cfg, hasCfg, err := s.LoadAppConfig(ctx)
	if err != nil {
		return in, err
	}
	in.config = cfg
	in.hasConfig = hasCfg
	rawTxns, err := s.ActiveTransactions(ctx)
	if err != nil {
		return in, err
	}
	in.txns = buildTxnsWithAccounts(rawTxns, in.accounts)
	salDec, err := s.RecentSalaries(ctx, 3)
	if err != nil {
		return in, err
	}
	for _, d := range salDec {
		f, _ := d.Float64()
		in.salaries = append(in.salaries, f)
	}
	return in, nil
}

// hasInvestment reports whether any active account is an investment category.
func hasInvestment(accounts []Account) bool {
	for _, a := range accounts {
		if _, ok := investmentCategories[a.Category]; ok {
			return true
		}
	}
	return false
}

// latestSnapshotByDate returns the snapshot with the highest (date, id) among
// those that have at least one merged row — the raw path's selection.
func latestSnapshotByDate(rows []mergedRow) (int, time.Time, bool) {
	found := false
	var bestID int
	var bestDate time.Time
	for _, r := range rows {
		if !found || r.Date.After(bestDate) ||
			(r.Date.Equal(bestDate) && r.SnapshotID > bestID) {
			found = true
			bestID = r.SnapshotID
			bestDate = r.Date
		}
	}
	return bestID, bestDate, found
}

// sortAllocation orders allocation items by (category, owner_user_id).
// A nil owner_user_id (jointly owned) sorts before any concrete id.
func sortAllocation(items []allocationItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		oi, oj := items[i].OwnerUserID, items[j].OwnerUserID
		if (oi == nil) != (oj == nil) {
			return oi == nil
		}
		if oi == nil {
			return false
		}
		return *oi < *oj
	})
}
