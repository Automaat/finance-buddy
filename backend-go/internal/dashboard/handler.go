package dashboard

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// Handler is the HTTP boundary for /api/dashboard.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

// Get serves GET /api/dashboard.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rng, err := parseDateRange(r.URL.Query())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": err.Error()})
		return
	}
	res, err := Compute(r.Context(), h.store)
	if err != nil {
		h.logger.Error("dashboard", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Internal Server Error"})
		return
	}
	applyDateRange(&res, rng)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toWire(res)); err != nil {
		h.logger.Error("encode dashboard", "err", err)
	}
}

// dateRange is the inclusive [from, to] filter from the query string. A zero
// time on either bound disables that side of the filter.
type dateRange struct {
	from time.Time
	to   time.Time
}

func parseDateRange(q url.Values) (dateRange, error) {
	var r dateRange
	if s := strings.TrimSpace(q.Get("date_from")); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return r, errors.New("invalid date_from — expected yyyy-mm-dd")
		}
		r.from = t
	}
	if s := strings.TrimSpace(q.Get("date_to")); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return r, errors.New("invalid date_to — expected yyyy-mm-dd")
		}
		r.to = t
	}
	if !r.from.IsZero() && !r.to.IsZero() && r.to.Before(r.from) {
		return r, errors.New("date_to must be on or after date_from")
	}
	return r, nil
}

func (r dateRange) includes(d time.Time) bool {
	if !r.from.IsZero() && d.Before(r.from) {
		return false
	}
	if !r.to.IsZero() && d.After(r.to) {
		return false
	}
	return true
}

// applyDateRange filters the time-series fields in place. Snapshot tiles
// (current net worth, totals, allocation, tile deltas) are not touched —
// they always reflect the latest state.
func applyDateRange(res *result, rng dateRange) {
	if rng.from.IsZero() && rng.to.IsZero() {
		return
	}
	res.NetWorthHistory = filterNetWorthHistory(res.NetWorthHistory, rng)
	res.InvestmentTimeSeries = filterTimeSeries(res.InvestmentTimeSeries, rng)
	for k, v := range res.WrapperTimeSeries {
		res.WrapperTimeSeries[k] = filterTimeSeries(v, rng)
	}
	for k, v := range res.CategoryTimeSeries {
		res.CategoryTimeSeries[k] = filterTimeSeries(v, rng)
	}
}

func filterNetWorthHistory(pts []netWorthPoint, rng dateRange) []netWorthPoint {
	out := make([]netWorthPoint, 0, len(pts))
	for _, p := range pts {
		if rng.includes(p.Date) {
			out = append(out, p)
		}
	}
	return out
}

func filterTimeSeries(pts []timeSeriesPoint, rng dateRange) []timeSeriesPoint {
	out := make([]timeSeriesPoint, 0, len(pts))
	for _, p := range pts {
		if rng.includes(p.Date) {
			out = append(out, p)
		}
	}
	return out
}

type netWorthPointWire struct {
	Date        wire.IsoDate `json:"date"`
	Value       wire.PyFloat `json:"value"`
	Assets      wire.PyFloat `json:"assets"`
	Liabilities wire.PyFloat `json:"liabilities"`
	SnapshotID  int          `json:"snapshot_id"`
}

type allocationItemWire struct {
	Category    string       `json:"category"`
	OwnerUserID *int         `json:"owner_user_id"`
	Value       wire.PyFloat `json:"value"`
}

type deltaValueWire struct {
	Absolute   wire.PyFloat  `json:"absolute"`
	Percentage *wire.PyFloat `json:"percentage"`
}

type tileDeltaWire struct {
	MoM *deltaValueWire `json:"mom"`
	YoY *deltaValueWire `json:"yoy"`
}

type tileDeltasWire struct {
	NetWorth    tileDeltaWire `json:"net_worth"`
	Assets      tileDeltaWire `json:"assets"`
	Liabilities tileDeltaWire `json:"liabilities"`
}

type metricCardsWire struct {
	PropertySQM             wire.PyFloat  `json:"property_sqm"`
	EmergencyFundMonths     wire.PyFloat  `json:"emergency_fund_months"`
	RetirementIncomeMonthly wire.PyFloat  `json:"retirement_income_monthly"`
	MortgageRemaining       wire.PyFloat  `json:"mortgage_remaining"`
	MortgageMonthsLeft      int           `json:"mortgage_months_left"`
	MortgageYearsLeft       wire.PyFloat  `json:"mortgage_years_left"`
	RetirementTotal         wire.PyFloat  `json:"retirement_total"`
	InvestmentContributions wire.PyFloat  `json:"investment_contributions"`
	InvestmentReturns       wire.PyFloat  `json:"investment_returns"`
	SavingsRate             *wire.PyFloat `json:"savings_rate"`
	DebtToIncomeRatio       *wire.PyFloat `json:"debt_to_income_ratio"`
	HourOfWorkCost          *wire.PyFloat `json:"hour_of_work_cost"`
	HourOfLifeCost          *wire.PyFloat `json:"hour_of_life_cost"`
	FIRENumber              *wire.PyFloat `json:"fire_number"`
	FIProgress              *wire.PyFloat `json:"fi_progress"`
	RunwayMonths            *wire.PyFloat `json:"runway_months"`
	AnnualExpenses          *wire.PyFloat `json:"annual_expenses"`
	WithdrawalRate          *wire.PyFloat `json:"withdrawal_rate"`
	CoastFIRENumber         *wire.PyFloat `json:"coast_fire_number"`
	CoastFIREGap            *wire.PyFloat `json:"coast_fire_gap"`
	CoastFIRETargetAge      *int          `json:"coast_fire_target_age"`
	ExpectedReturnRate      *wire.PyFloat `json:"expected_return_rate"`
	BaristaMonthlyIncome    *wire.PyFloat `json:"barista_monthly_income"`
	BaristaAnnualGap        *wire.PyFloat `json:"barista_annual_gap"`
	BaristaFIRENumber       *wire.PyFloat `json:"barista_fire_number"`
	BaristaFIProgress       *wire.PyFloat `json:"barista_fi_progress"`
	BaristaYearsToFI        *wire.PyFloat `json:"barista_years_to_fi"`
	BridgeYears             *int          `json:"bridge_years"`
	BridgeCapitalNeeded     *wire.PyFloat `json:"bridge_capital_needed"`
	BridgeLiquidCapital     *wire.PyFloat `json:"bridge_liquid_capital"`
	BridgeCapitalGap        *wire.PyFloat `json:"bridge_capital_gap"`
	LeanFIRENumber          *wire.PyFloat `json:"lean_fire_number"`
	LeanFIProgress          *wire.PyFloat `json:"lean_fi_progress"`
	FatFIRENumber           *wire.PyFloat `json:"fat_fire_number"`
	FatFIProgress           *wire.PyFloat `json:"fat_fi_progress"`
	MonthlySavings          *wire.PyFloat `json:"monthly_savings"`
	FIYearsRemaining        *wire.PyFloat `json:"fi_years_remaining"`
	FIProjectedDate         *string       `json:"fi_projected_date"`
	FireNetWorth            *wire.PyFloat `json:"fire_net_worth"`
	FireExcludedValue       *wire.PyFloat `json:"fire_excluded_value"`
}

type allocationBreakdownWire struct {
	Category          string       `json:"category"`
	CurrentValue      wire.PyFloat `json:"current_value"`
	CurrentPercentage wire.PyFloat `json:"current_percentage"`
	TargetPercentage  wire.PyFloat `json:"target_percentage"`
	Difference        wire.PyFloat `json:"difference"`
}

type wrapperBreakdownWire struct {
	Wrapper    string       `json:"wrapper"`
	Value      wire.PyFloat `json:"value"`
	Percentage wire.PyFloat `json:"percentage"`
}

type rebalancingWire struct {
	Category string       `json:"category"`
	Action   string       `json:"action"`
	Amount   wire.PyFloat `json:"amount"`
}

type allocationAnalysisWire struct {
	ByCategory           []allocationBreakdownWire `json:"by_category"`
	ByWrapper            []wrapperBreakdownWire    `json:"by_wrapper"`
	Rebalancing          []rebalancingWire         `json:"rebalancing"`
	TotalInvestmentValue wire.PyFloat              `json:"total_investment_value"`
}

type timeSeriesPointWire struct {
	Date          wire.IsoDate `json:"date"`
	Value         wire.PyFloat `json:"value"`
	Contributions wire.PyFloat `json:"contributions"`
	Returns       wire.PyFloat `json:"returns"`
}

type wrapperTimeSeriesWire struct {
	IKE  []timeSeriesPointWire `json:"ike"`
	IKZE []timeSeriesPointWire `json:"ikze"`
	PPK  []timeSeriesPointWire `json:"ppk"`
}

type categoryTimeSeriesWire struct {
	Stock []timeSeriesPointWire `json:"stock"`
	Bond  []timeSeriesPointWire `json:"bond"`
}

type dashboardWire struct {
	NetWorthHistory        []netWorthPointWire    `json:"net_worth_history"`
	CurrentNetWorth        wire.PyFloat           `json:"current_net_worth"`
	ChangeVsLastMonth      wire.PyFloat           `json:"change_vs_last_month"`
	TotalAssets            wire.PyFloat           `json:"total_assets"`
	TotalLiabilities       wire.PyFloat           `json:"total_liabilities"`
	Allocation             []allocationItemWire   `json:"allocation"`
	RetirementAccountValue wire.PyFloat           `json:"retirement_account_value"`
	MetricCards            metricCardsWire        `json:"metric_cards"`
	AllocationAnalysis     allocationAnalysisWire `json:"allocation_analysis"`
	InvestmentTimeSeries   []timeSeriesPointWire  `json:"investment_time_series"`
	WrapperTimeSeries      wrapperTimeSeriesWire  `json:"wrapper_time_series"`
	CategoryTimeSeries     categoryTimeSeriesWire `json:"category_time_series"`
	TileDeltas             tileDeltasWire         `json:"tile_deltas"`
	LatestSnapshotDate     *wire.IsoDate          `json:"latest_snapshot_date"`
}

func toWire(res result) dashboardWire {
	w := dashboardWire{
		CurrentNetWorth:        wire.PyFloat(res.CurrentNetWorth),
		ChangeVsLastMonth:      wire.PyFloat(res.ChangeVsLastMonth),
		TotalAssets:            wire.PyFloat(res.TotalAssets),
		TotalLiabilities:       wire.PyFloat(res.TotalLiabilities),
		RetirementAccountValue: wire.PyFloat(res.RetirementAccountValue),
		MetricCards:            metricCardsToWire(res.MetricCards),
		AllocationAnalysis:     allocationAnalysisToWire(res.AllocationAnalysis),
		TileDeltas:             tileDeltasToWire(res.TileDeltas),
	}
	w.NetWorthHistory = make([]netWorthPointWire, 0, len(res.NetWorthHistory))
	for _, p := range res.NetWorthHistory {
		w.NetWorthHistory = append(w.NetWorthHistory, netWorthPointWire{
			Date:        wire.IsoDate(p.Date),
			Value:       wire.PyFloat(p.Value),
			Assets:      wire.PyFloat(p.Assets),
			Liabilities: wire.PyFloat(p.Liabilities),
			SnapshotID:  p.SnapshotID,
		})
	}
	w.Allocation = make([]allocationItemWire, 0, len(res.Allocation))
	for _, a := range res.Allocation {
		w.Allocation = append(w.Allocation, allocationItemWire{
			Category: a.Category, OwnerUserID: a.OwnerUserID, Value: wire.PyFloat(a.Value),
		})
	}
	w.InvestmentTimeSeries = seriesToWire(res.InvestmentTimeSeries)
	w.WrapperTimeSeries = wrapperTimeSeriesWire{
		IKE:  seriesToWire(res.WrapperTimeSeries["IKE"]),
		IKZE: seriesToWire(res.WrapperTimeSeries["IKZE"]),
		PPK:  seriesToWire(res.WrapperTimeSeries["PPK"]),
	}
	w.CategoryTimeSeries = categoryTimeSeriesWire{
		Stock: seriesToWire(res.CategoryTimeSeries["stock"]),
		Bond:  seriesToWire(res.CategoryTimeSeries["bond"]),
	}
	if res.LatestSnapshotDate != nil {
		d := wire.IsoDate(*res.LatestSnapshotDate)
		w.LatestSnapshotDate = &d
	}
	return w
}

func seriesToWire(pts []timeSeriesPoint) []timeSeriesPointWire {
	out := make([]timeSeriesPointWire, 0, len(pts))
	for _, p := range pts {
		out = append(out, timeSeriesPointWire{
			Date:          wire.IsoDate(p.Date),
			Value:         wire.PyFloat(p.Value),
			Contributions: wire.PyFloat(p.Contributions),
			Returns:       wire.PyFloat(p.Returns),
		})
	}
	return out
}

func metricCardsToWire(m metricCards) metricCardsWire {
	return metricCardsWire{
		PropertySQM:             wire.PyFloat(m.PropertySQM),
		EmergencyFundMonths:     wire.PyFloat(m.EmergencyFundMonths),
		RetirementIncomeMonthly: wire.PyFloat(m.RetirementIncomeMonthly),
		MortgageRemaining:       wire.PyFloat(m.MortgageRemaining),
		MortgageMonthsLeft:      m.MortgageMonthsLeft,
		MortgageYearsLeft:       wire.PyFloat(m.MortgageYearsLeft),
		RetirementTotal:         wire.PyFloat(m.RetirementTotal),
		InvestmentContributions: wire.PyFloat(m.InvestmentContributions),
		InvestmentReturns:       wire.PyFloat(m.InvestmentReturns),
		SavingsRate:             floatPtr(m.SavingsRate),
		DebtToIncomeRatio:       floatPtr(m.DebtToIncomeRatio),
		HourOfWorkCost:          floatPtr(m.HourOfWorkCost),
		HourOfLifeCost:          floatPtr(m.HourOfLifeCost),
		FIRENumber:              floatPtr(m.FIRENumber),
		FIProgress:              floatPtr(m.FIProgress),
		RunwayMonths:            floatPtr(m.RunwayMonths),
		AnnualExpenses:          floatPtr(m.AnnualExpenses),
		WithdrawalRate:          floatPtr(m.WithdrawalRate),
		CoastFIRENumber:         floatPtr(m.CoastFIRENumber),
		CoastFIREGap:            floatPtr(m.CoastFIREGap),
		CoastFIRETargetAge:      m.CoastFIRETargetAge,
		ExpectedReturnRate:      floatPtr(m.ExpectedReturnRate),
		BaristaMonthlyIncome:    floatPtr(m.BaristaMonthlyIncome),
		BaristaAnnualGap:        floatPtr(m.BaristaAnnualGap),
		BaristaFIRENumber:       floatPtr(m.BaristaFIRENumber),
		BaristaFIProgress:       floatPtr(m.BaristaFIProgress),
		BaristaYearsToFI:        floatPtr(m.BaristaYearsToFI),
		BridgeYears:             m.BridgeYears,
		BridgeCapitalNeeded:     floatPtr(m.BridgeCapitalNeeded),
		BridgeLiquidCapital:     floatPtr(m.BridgeLiquidCapital),
		BridgeCapitalGap:        floatPtr(m.BridgeCapitalGap),
		LeanFIRENumber:          floatPtr(m.LeanFIRENumber),
		LeanFIProgress:          floatPtr(m.LeanFIProgress),
		FatFIRENumber:           floatPtr(m.FatFIRENumber),
		FatFIProgress:           floatPtr(m.FatFIProgress),
		MonthlySavings:          floatPtr(m.MonthlySavings),
		FIYearsRemaining:        floatPtr(m.FIYearsRemaining),
		FIProjectedDate:         m.FIProjectedDate,
		FireNetWorth:            floatPtr(m.FireNetWorth),
		FireExcludedValue:       floatPtr(m.FireExcludedValue),
	}
}

func allocationAnalysisToWire(a allocationAnalysis) allocationAnalysisWire {
	w := allocationAnalysisWire{
		TotalInvestmentValue: wire.PyFloat(a.TotalInvestmentValue),
		ByCategory:           make([]allocationBreakdownWire, 0, len(a.ByCategory)),
		ByWrapper:            make([]wrapperBreakdownWire, 0, len(a.ByWrapper)),
		Rebalancing:          make([]rebalancingWire, 0, len(a.Rebalancing)),
	}
	for _, b := range a.ByCategory {
		w.ByCategory = append(w.ByCategory, allocationBreakdownWire{
			Category:          b.Category,
			CurrentValue:      wire.PyFloat(b.CurrentValue),
			CurrentPercentage: wire.PyFloat(b.CurrentPercentage),
			TargetPercentage:  wire.PyFloat(b.TargetPercentage),
			Difference:        wire.PyFloat(b.Difference),
		})
	}
	for _, b := range a.ByWrapper {
		w.ByWrapper = append(w.ByWrapper, wrapperBreakdownWire{
			Wrapper: b.Wrapper, Value: wire.PyFloat(b.Value), Percentage: wire.PyFloat(b.Percentage),
		})
	}
	for _, b := range a.Rebalancing {
		w.Rebalancing = append(w.Rebalancing, rebalancingWire{
			Category: b.Category, Action: b.Action, Amount: wire.PyFloat(b.Amount),
		})
	}
	return w
}

func tileDeltasToWire(t tileDeltas) tileDeltasWire {
	return tileDeltasWire{
		NetWorth:    tileDeltaToWire(t.NetWorth),
		Assets:      tileDeltaToWire(t.Assets),
		Liabilities: tileDeltaToWire(t.Liabilities),
	}
}

func tileDeltaToWire(t tileDelta) tileDeltaWire {
	return tileDeltaWire{MoM: deltaToWire(t.MoM), YoY: deltaToWire(t.YoY)}
}

func deltaToWire(d *deltaValue) *deltaValueWire {
	if d == nil {
		return nil
	}
	return &deltaValueWire{Absolute: wire.PyFloat(d.Absolute), Percentage: floatPtr(d.Percentage)}
}

func floatPtr(f *float64) *wire.PyFloat {
	if f == nil {
		return nil
	}
	pf := wire.PyFloat(*f)
	return &pf
}
