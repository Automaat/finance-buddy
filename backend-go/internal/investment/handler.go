package investment

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

type response struct {
	Category         string       `json:"category"`
	TotalValue       wire.PyFloat `json:"total_value"`
	TotalContributed wire.PyFloat `json:"total_contributed"`
	Returns          wire.PyFloat `json:"returns"`
	ROIPercentage    wire.PyFloat `json:"roi_percentage"`
}

// Handler is the HTTP boundary for /api/investment.
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

// --- Contribution-adjusted returns (#401) ---

type returnsResponse struct {
	Scope             scopeWire     `json:"scope"`
	Period            string        `json:"period"`
	Since             *string       `json:"since"`
	AsOf              string        `json:"as_of"`
	Deposits          wire.PyFloat  `json:"deposits"`
	Withdrawals       wire.PyFloat  `json:"withdrawals"`
	NetContributed    wire.PyFloat  `json:"net_contributed"`
	CurrentValue      wire.PyFloat  `json:"current_value"`
	ValuationChange   wire.PyFloat  `json:"valuation_change"`
	SimpleROIPct      wire.PyFloat  `json:"simple_roi_pct"`
	MoneyWeightedPct  *wire.PyFloat `json:"money_weighted_pct"`
	HasSnapshot       bool          `json:"has_snapshot"`
	ConvergenceFailed bool          `json:"convergence_failed"`
}

type scopeWire struct {
	Type    string `json:"type"`
	Value   string `json:"value,omitempty"`
	Account *int   `json:"account_id,omitempty"`
}

// Returns serves GET /api/investment/returns. Query params:
//
//	scope=all|account|category|wrapper (default all)
//	id=N           required when scope=account
//	value=...      required when scope=category|wrapper
//	period=1m|3m|ytd|1y|all (default all)
//
// Computes contribution-adjusted (XIRR / money-weighted) return in addition
// to the naive simple ROI so the user can see whether their investments grew
// or whether they only added more money.
func (h *Handler) Returns(w http.ResponseWriter, r *http.Request) {
	scope, scopeWire, vErr := parseScope(r)
	if vErr != "" {
		writeDetailError(w, http.StatusBadRequest, vErr)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}
	if !validPeriod(period) {
		writeDetailError(w, http.StatusBadRequest, "period must be one of 1m|3m|ytd|1y|all")
		return
	}
	now := time.Now().UTC()
	window := windowFor(period, now)

	value, snapDate, hasSnap, err := h.store.LatestValueInScope(r.Context(), scope, now)
	if err != nil {
		h.logger.Error("returns: snapshot lookup", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// Cash flows must align with the valuation date. When a snapshot exists,
	// upper-bound the flow window at snapDate so transactions added after the
	// snapshot don't pollute the return calculation.
	flowEnd := now
	if hasSnap {
		flowEnd = snapDate
	}
	flows, err := h.store.ScopedFlows(r.Context(), scope, PeriodWindow{Since: window.Since, AsOf: flowEnd})
	if err != nil {
		h.logger.Error("returns: flows", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	deposits, withdrawals := 0.0, 0.0
	xirrFlows := make([]CashFlow, 0, len(flows)+2)
	// Windowed periods need an opening balance so XIRR reflects the return
	// over the period rather than treating pre-window value as free money.
	// Inject the snapshot-as-of-since as a positive cash flow at `since`.
	var openingValue float64
	if window.Since != nil {
		openValDec, _, hasOpen, err := h.store.LatestValueInScope(r.Context(), scope, *window.Since)
		if err != nil {
			h.logger.Error("returns: opening snapshot", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if hasOpen {
			openingValue, _ = openValDec.Float64()
			if openingValue > 0 {
				xirrFlows = append(xirrFlows, CashFlow{Date: *window.Since, Amount: openingValue})
			}
		}
	}
	for _, f := range flows {
		amt, _ := f.Amount.Float64()
		if amt >= 0 {
			deposits += amt
		} else {
			withdrawals += -amt
		}
		xirrFlows = append(xirrFlows, CashFlow{Date: f.Date, Amount: amt})
	}
	netContrib := deposits - withdrawals
	currentValue, _ := value.Float64()
	// Valuation change is the difference between current value and what was
	// put in over the window — current value minus (opening + net flows).
	valuationChange := currentValue - openingValue - netContrib

	asOf := now
	if hasSnap {
		asOf = snapDate
	}
	resp := returnsResponse{
		Scope:           scopeWire,
		Period:          period,
		AsOf:            asOf.Format("2006-01-02"),
		Deposits:        wire.PyFloat(deposits),
		Withdrawals:     wire.PyFloat(withdrawals),
		NetContributed:  wire.PyFloat(netContrib),
		CurrentValue:    wire.PyFloat(currentValue),
		ValuationChange: wire.PyFloat(valuationChange),
		SimpleROIPct:    wire.PyFloat(math.Round(SimpleROI(openingValue+netContrib, currentValue)*10000) / 100),
		HasSnapshot:     hasSnap,
	}
	if window.Since != nil {
		s := window.Since.Format("2006-01-02")
		resp.Since = &s
	}
	if hasSnap {
		// Closing terminal flow = liquidate at current value at the snapshot date.
		xirrFlows = append(xirrFlows, CashFlow{Date: snapDate, Amount: -currentValue})
		mwr, xerr := XIRR(xirrFlows)
		if xerr != nil {
			resp.ConvergenceFailed = true
		} else {
			pct := wire.PyFloat(math.Round(mwr*10000) / 100)
			resp.MoneyWeightedPct = &pct
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func validPeriod(p string) bool {
	switch p {
	case "1m", "3m", "ytd", "1y", "all":
		return true
	}
	return false
}

func parseScope(r *http.Request) (ScopeFilter, scopeWire, string) {
	q := r.URL.Query()
	kind := q.Get("scope")
	if kind == "" {
		kind = "all"
	}
	switch kind {
	case "all":
		return ScopeFilter{All: true}, scopeWire{Type: "all"}, ""
	case "account":
		raw := q.Get("id")
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return ScopeFilter{}, scopeWire{}, "scope=account requires a positive integer id"
		}
		return ScopeFilter{AccountID: &id}, scopeWire{Type: "account", Account: &id}, ""
	case "category":
		v := q.Get("value")
		if v == "" {
			return ScopeFilter{}, scopeWire{}, "scope=category requires value"
		}
		return ScopeFilter{Category: &v}, scopeWire{Type: "category", Value: v}, ""
	case "wrapper":
		v := q.Get("value")
		if v == "" {
			return ScopeFilter{}, scopeWire{}, "scope=wrapper requires value"
		}
		return ScopeFilter{Wrapper: &v}, scopeWire{Type: "wrapper", Value: v}, ""
	}
	return ScopeFilter{}, scopeWire{}, "scope must be one of all|account|category|wrapper"
}

func windowFor(period string, asOf time.Time) PeriodWindow {
	switch period {
	case "1m":
		s := asOf.AddDate(0, -1, 0)
		return PeriodWindow{Since: &s, AsOf: asOf}
	case "3m":
		s := asOf.AddDate(0, -3, 0)
		return PeriodWindow{Since: &s, AsOf: asOf}
	case "ytd":
		s := time.Date(asOf.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return PeriodWindow{Since: &s, AsOf: asOf}
	case "1y":
		s := asOf.AddDate(-1, 0, 0)
		return PeriodWindow{Since: &s, AsOf: asOf}
	}
	return PeriodWindow{AsOf: asOf}
}

// StockStats serves GET /api/investment/stock-stats.
func (h *Handler) StockStats(w http.ResponseWriter, r *http.Request) {
	h.categoryStats(w, r, "stock")
}

// BondStats serves GET /api/investment/bond-stats.
func (h *Handler) BondStats(w http.ResponseWriter, r *http.Request) {
	h.categoryStats(w, r, "bond")
}

func (h *Handler) categoryStats(w http.ResponseWriter, r *http.Request, category string) {
	stats, err := h.store.StatsForCategory(r.Context(), category)
	if err != nil {
		h.logger.Error("category stats", "category", category, "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !stats.HasAccounts {
		writeJSON(w, http.StatusOK, response{Category: category})
		return
	}
	value, _ := stats.TotalValue.Float64()
	contributed, _ := stats.TotalContributed.Float64()
	returns := value - contributed
	roi := 0.0
	if contributed > 0 {
		roi = returns / contributed * 100
	}
	writeJSON(w, http.StatusOK, response{
		Category:         category,
		TotalValue:       wire.PyFloat(value),
		TotalContributed: wire.PyFloat(contributed),
		Returns:          wire.PyFloat(returns),
		ROIPercentage:    wire.PyFloat(math.Round(roi*100) / 100),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeDetailError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}
