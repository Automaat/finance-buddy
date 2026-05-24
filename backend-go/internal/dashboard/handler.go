package dashboard

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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

// --- wire types ---

type isoDate time.Time

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

type netWorthPointWire struct {
	Date        isoDate `json:"date"`
	Value       pyFloat `json:"value"`
	Assets      pyFloat `json:"assets"`
	Liabilities pyFloat `json:"liabilities"`
	SnapshotID  int     `json:"snapshot_id"`
}

type allocationItemWire struct {
	Category    string  `json:"category"`
	OwnerUserID *int    `json:"owner_user_id"`
	Value       pyFloat `json:"value"`
}

type deltaValueWire struct {
	Absolute   pyFloat  `json:"absolute"`
	Percentage *pyFloat `json:"percentage"`
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
	PropertySQM             pyFloat  `json:"property_sqm"`
	EmergencyFundMonths     pyFloat  `json:"emergency_fund_months"`
	RetirementIncomeMonthly pyFloat  `json:"retirement_income_monthly"`
	MortgageRemaining       pyFloat  `json:"mortgage_remaining"`
	MortgageMonthsLeft      int      `json:"mortgage_months_left"`
	MortgageYearsLeft       pyFloat  `json:"mortgage_years_left"`
	RetirementTotal         pyFloat  `json:"retirement_total"`
	InvestmentContributions pyFloat  `json:"investment_contributions"`
	InvestmentReturns       pyFloat  `json:"investment_returns"`
	SavingsRate             *pyFloat `json:"savings_rate"`
	DebtToIncomeRatio       *pyFloat `json:"debt_to_income_ratio"`
	HourOfWorkCost          *pyFloat `json:"hour_of_work_cost"`
	HourOfLifeCost          *pyFloat `json:"hour_of_life_cost"`
	FIRENumber              *pyFloat `json:"fire_number"`
	FIProgress              *pyFloat `json:"fi_progress"`
	RunwayMonths            *pyFloat `json:"runway_months"`
	AnnualExpenses          *pyFloat `json:"annual_expenses"`
	WithdrawalRate          *pyFloat `json:"withdrawal_rate"`
	CoastFIRENumber         *pyFloat `json:"coast_fire_number"`
	CoastFIREGap            *pyFloat `json:"coast_fire_gap"`
	CoastFIRETargetAge      *int     `json:"coast_fire_target_age"`
	ExpectedReturnRate      *pyFloat `json:"expected_return_rate"`
}

type allocationBreakdownWire struct {
	Category          string  `json:"category"`
	CurrentValue      pyFloat `json:"current_value"`
	CurrentPercentage pyFloat `json:"current_percentage"`
	TargetPercentage  pyFloat `json:"target_percentage"`
	Difference        pyFloat `json:"difference"`
}

type wrapperBreakdownWire struct {
	Wrapper    string  `json:"wrapper"`
	Value      pyFloat `json:"value"`
	Percentage pyFloat `json:"percentage"`
}

type rebalancingWire struct {
	Category string  `json:"category"`
	Action   string  `json:"action"`
	Amount   pyFloat `json:"amount"`
}

type allocationAnalysisWire struct {
	ByCategory           []allocationBreakdownWire `json:"by_category"`
	ByWrapper            []wrapperBreakdownWire    `json:"by_wrapper"`
	Rebalancing          []rebalancingWire         `json:"rebalancing"`
	TotalInvestmentValue pyFloat                   `json:"total_investment_value"`
}

type timeSeriesPointWire struct {
	Date          isoDate `json:"date"`
	Value         pyFloat `json:"value"`
	Contributions pyFloat `json:"contributions"`
	Returns       pyFloat `json:"returns"`
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
	CurrentNetWorth        pyFloat                `json:"current_net_worth"`
	ChangeVsLastMonth      pyFloat                `json:"change_vs_last_month"`
	TotalAssets            pyFloat                `json:"total_assets"`
	TotalLiabilities       pyFloat                `json:"total_liabilities"`
	Allocation             []allocationItemWire   `json:"allocation"`
	RetirementAccountValue pyFloat                `json:"retirement_account_value"`
	MetricCards            metricCardsWire        `json:"metric_cards"`
	AllocationAnalysis     allocationAnalysisWire `json:"allocation_analysis"`
	InvestmentTimeSeries   []timeSeriesPointWire  `json:"investment_time_series"`
	WrapperTimeSeries      wrapperTimeSeriesWire  `json:"wrapper_time_series"`
	CategoryTimeSeries     categoryTimeSeriesWire `json:"category_time_series"`
	TileDeltas             tileDeltasWire         `json:"tile_deltas"`
}

func toWire(res result) dashboardWire {
	w := dashboardWire{
		CurrentNetWorth:        pyFloat(res.CurrentNetWorth),
		ChangeVsLastMonth:      pyFloat(res.ChangeVsLastMonth),
		TotalAssets:            pyFloat(res.TotalAssets),
		TotalLiabilities:       pyFloat(res.TotalLiabilities),
		RetirementAccountValue: pyFloat(res.RetirementAccountValue),
		MetricCards:            metricCardsToWire(res.MetricCards),
		AllocationAnalysis:     allocationAnalysisToWire(res.AllocationAnalysis),
		TileDeltas:             tileDeltasToWire(res.TileDeltas),
	}
	w.NetWorthHistory = make([]netWorthPointWire, 0, len(res.NetWorthHistory))
	for _, p := range res.NetWorthHistory {
		w.NetWorthHistory = append(w.NetWorthHistory, netWorthPointWire{
			Date:        isoDate(p.Date),
			Value:       pyFloat(p.Value),
			Assets:      pyFloat(p.Assets),
			Liabilities: pyFloat(p.Liabilities),
			SnapshotID:  p.SnapshotID,
		})
	}
	w.Allocation = make([]allocationItemWire, 0, len(res.Allocation))
	for _, a := range res.Allocation {
		w.Allocation = append(w.Allocation, allocationItemWire{
			Category: a.Category, OwnerUserID: a.OwnerUserID, Value: pyFloat(a.Value),
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
	return w
}

func seriesToWire(pts []timeSeriesPoint) []timeSeriesPointWire {
	out := make([]timeSeriesPointWire, 0, len(pts))
	for _, p := range pts {
		out = append(out, timeSeriesPointWire{
			Date:          isoDate(p.Date),
			Value:         pyFloat(p.Value),
			Contributions: pyFloat(p.Contributions),
			Returns:       pyFloat(p.Returns),
		})
	}
	return out
}

func metricCardsToWire(m metricCards) metricCardsWire {
	return metricCardsWire{
		PropertySQM:             pyFloat(m.PropertySQM),
		EmergencyFundMonths:     pyFloat(m.EmergencyFundMonths),
		RetirementIncomeMonthly: pyFloat(m.RetirementIncomeMonthly),
		MortgageRemaining:       pyFloat(m.MortgageRemaining),
		MortgageMonthsLeft:      m.MortgageMonthsLeft,
		MortgageYearsLeft:       pyFloat(m.MortgageYearsLeft),
		RetirementTotal:         pyFloat(m.RetirementTotal),
		InvestmentContributions: pyFloat(m.InvestmentContributions),
		InvestmentReturns:       pyFloat(m.InvestmentReturns),
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
	}
}

func allocationAnalysisToWire(a allocationAnalysis) allocationAnalysisWire {
	w := allocationAnalysisWire{
		TotalInvestmentValue: pyFloat(a.TotalInvestmentValue),
		ByCategory:           make([]allocationBreakdownWire, 0, len(a.ByCategory)),
		ByWrapper:            make([]wrapperBreakdownWire, 0, len(a.ByWrapper)),
		Rebalancing:          make([]rebalancingWire, 0, len(a.Rebalancing)),
	}
	for _, b := range a.ByCategory {
		w.ByCategory = append(w.ByCategory, allocationBreakdownWire{
			Category:          b.Category,
			CurrentValue:      pyFloat(b.CurrentValue),
			CurrentPercentage: pyFloat(b.CurrentPercentage),
			TargetPercentage:  pyFloat(b.TargetPercentage),
			Difference:        pyFloat(b.Difference),
		})
	}
	for _, b := range a.ByWrapper {
		w.ByWrapper = append(w.ByWrapper, wrapperBreakdownWire{
			Wrapper: b.Wrapper, Value: pyFloat(b.Value), Percentage: pyFloat(b.Percentage),
		})
	}
	for _, b := range a.Rebalancing {
		w.Rebalancing = append(w.Rebalancing, rebalancingWire{
			Category: b.Category, Action: b.Action, Amount: pyFloat(b.Amount),
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
	return &deltaValueWire{Absolute: pyFloat(d.Absolute), Percentage: floatPtr(d.Percentage)}
}

func floatPtr(f *float64) *pyFloat {
	if f == nil {
		return nil
	}
	pf := pyFloat(*f)
	return &pf
}
