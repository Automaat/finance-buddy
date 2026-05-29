// Package exposure derives portfolio currency exposure from the most recent
// snapshot. Group totals are reported in PLN (snapshot values are already
// PLN-converted), bucketed by the source account's quoted currency so users
// can see how much of their net worth is denominated in PLN vs. foreign
// currencies. Optional target bands report drift against a user-supplied
// PLN target percentage.
package exposure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// CurrencyBucket is one currency's slice of the portfolio.
type CurrencyBucket struct {
	Currency string  `json:"currency"`
	ValuePLN float64 `json:"value_pln"`
	Percent  float64 `json:"percent"`
}

// DriftBand is the optional drift summary against a user-set PLN target.
type DriftBand struct {
	TargetPLNPct float64 `json:"target_pln_pct"`
	ActualPLNPct float64 `json:"actual_pln_pct"`
	DriftPLNPct  float64 `json:"drift_pln_pct"`
	WithinTol    bool    `json:"within_tolerance"`
	TolerancePct float64 `json:"tolerance_pct"`
}

// Report is the full currency-exposure payload.
type Report struct {
	Currencies   []CurrencyBucket `json:"currencies"`
	TotalPLN     float64          `json:"total_pln"`
	PLNPercent   float64          `json:"pln_pct"`
	ForeignPct   float64          `json:"foreign_pct"`
	SnapshotDate string           `json:"snapshot_date"`
	Drift        *DriftBand       `json:"drift,omitempty"`
}

// Store loads currency-bucketed account values from the latest snapshot.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// row mirrors one (currency, sum(value)) pair. Used internally by the
// handler after the per-account snapshot values have been allocated across
// security currencies via the holdings breakdown.
type row struct {
	currency string
	valuePLN decimal.Decimal
}

// accountValue is one row's slice of the latest snapshot: which account,
// what category, what nominal currency, what PLN value. The handler walks
// these and decides per row whether to bucket as accounts.currency (cash
// accounts, real estate, …) or to fan out across the security currencies
// the account actually holds (stock/etf/bond/fund accounts).
type accountValue struct {
	accountID int
	category  string
	currency  string
	valuePLN  decimal.Decimal
}

// LatestByAccount returns one row per (active account, latest snapshot
// value). Used by the handler to decide per-row whether the value should
// bucket as accounts.currency or fan out by underlying holdings currency.
// Returns ("") snapshotDate when there are no snapshots yet.
func (s *Store) LatestByAccount(ctx context.Context) ([]accountValue, string, error) {
	var (
		latestID   int
		latestDate string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, to_char(date, 'YYYY-MM-DD')
		FROM snapshots ORDER BY date DESC, id DESC LIMIT 1`).Scan(&latestID, &latestDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return []accountValue{}, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("latest snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT sv.account_id, a.category, UPPER(a.currency), sv.value
		FROM snapshot_values sv
		JOIN accounts a ON a.id = sv.account_id
		WHERE sv.snapshot_id = $1 AND a.is_active = true
	`, latestID)
	if err != nil {
		return nil, "", fmt.Errorf("query per-account values: %w", err)
	}
	defer rows.Close()
	out := []accountValue{}
	for rows.Next() {
		var v accountValue
		if err := rows.Scan(&v.accountID, &v.category, &v.currency, &v.valuePLN); err != nil {
			return nil, "", fmt.Errorf("scan per-account value: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate per-account values: %w", err)
	}
	return out, latestDate, nil
}

// investmentCategories names the account categories whose snapshot value
// should fan out across underlying holdings currencies. Bank/cash/real-estate
// accounts keep the accounts.currency attribution.
var investmentCategories = map[string]bool{
	"stock": true,
	"etf":   true,
	"bond":  true,
	"fund":  true,
}

// HoldingsBreakdown returns per-account per-currency live MV PLN. Mirrors
// holdings.Valuator.AccountCurrencyBreakdownPLN; declared here as an
// interface so tests can stub the holdings dep without spinning up a real
// Postgres-backed Valuator.
type HoldingsBreakdown interface {
	AccountCurrencyBreakdownPLN(ctx context.Context) (map[int]map[string]decimal.Decimal, error)
}

// bucketByCurrency walks the latest snapshot values and bucketizes them
// into per-currency PLN totals. For accounts in investmentCategories that
// have a live holdings breakdown, the snapshot value is split across
// currencies in proportion to the live MV ratio — so an IKE XTB account
// whose snapshot value is 53 495 PLN and whose holdings are 100% USD
// attributes the full 53 495 PLN to USD instead of PLN. Falls back to
// accounts.currency when no breakdown exists (empty holdings, bank/cash
// accounts, etc.).
func bucketByCurrency(values []accountValue, breakdown map[int]map[string]decimal.Decimal) []row {
	totals := map[string]decimal.Decimal{}
	for i := range values {
		v := &values[i]
		buckets, ok := breakdown[v.accountID]
		if ok && investmentCategories[v.category] && len(buckets) > 0 {
			addProportional(totals, v.valuePLN, buckets)
			continue
		}
		totals[v.currency] = totals[v.currency].Add(v.valuePLN)
	}
	out := make([]row, 0, len(totals))
	for ccy, val := range totals {
		out = append(out, row{currency: ccy, valuePLN: val})
	}
	return out
}

// addProportional splits `total` across the keys of `weights` in proportion
// to the weight values and adds each share into `dst`. When weights sum to
// zero (all positions valued at zero), `total` falls through to a single
// "PLN" bucket — same fallback as a cash account, since a zero-value
// holdings ratio carries no currency signal.
func addProportional(dst map[string]decimal.Decimal, total decimal.Decimal, weights map[string]decimal.Decimal) {
	sum := decimal.Zero
	for _, w := range weights {
		sum = sum.Add(w)
	}
	if sum.IsZero() {
		dst["PLN"] = dst["PLN"].Add(total)
		return
	}
	for ccy, w := range weights {
		share := total.Mul(w).Div(sum)
		dst[ccy] = dst[ccy].Add(share)
	}
}

// BuildReport collapses raw per-currency values into percentages and the
// optional drift band. Pure function on top of the store result.
func BuildReport(rows []row, snapshotDate string, targetPLN *float64, tolerance float64) Report {
	rep := Report{Currencies: []CurrencyBucket{}, SnapshotDate: snapshotDate}
	totalAssets := decimal.Zero
	for i := range rows {
		if rows[i].valuePLN.IsPositive() {
			totalAssets = totalAssets.Add(rows[i].valuePLN)
		}
	}
	rep.TotalPLN, _ = totalAssets.Float64()
	for i := range rows {
		v, _ := rows[i].valuePLN.Float64()
		if v <= 0 {
			continue
		}
		pct := 0.0
		if rep.TotalPLN > 0 {
			pct = v / rep.TotalPLN * 100
		}
		rep.Currencies = append(rep.Currencies, CurrencyBucket{
			Currency: rows[i].currency,
			ValuePLN: v,
			Percent:  pct,
		})
		if rows[i].currency == "PLN" {
			rep.PLNPercent = pct
		}
	}
	rep.ForeignPct = 100 - rep.PLNPercent
	if rep.TotalPLN == 0 {
		rep.PLNPercent = 0
		rep.ForeignPct = 0
	}
	sort.SliceStable(rep.Currencies, func(i, j int) bool {
		return rep.Currencies[i].Percent > rep.Currencies[j].Percent
	})
	if targetPLN != nil {
		band := DriftBand{
			TargetPLNPct: *targetPLN,
			ActualPLNPct: rep.PLNPercent,
			DriftPLNPct:  rep.PLNPercent - *targetPLN,
			TolerancePct: tolerance,
		}
		band.WithinTol = abs(band.DriftPLNPct) <= tolerance
		rep.Drift = &band
	}
	return rep
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Handler is the HTTP boundary for /api/exposure.
type Handler struct {
	store    *Store
	holdings HoldingsBreakdown
	logger   *slog.Logger
}

// NewHandler wires the store + holdings breakdown + logger. holdings may
// be nil in tests; the report then attributes every investment account by
// accounts.currency (the pre-#662 behavior).
func NewHandler(store *Store, holdings HoldingsBreakdown, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, holdings: holdings, logger: logger}
}

// Currency serves GET /api/exposure/currency[?target_pln_pct=X&tolerance=Y].
// Both query params are optional; omitting target_pln_pct skips the drift
// block entirely.
func (h *Handler) Currency(w http.ResponseWriter, r *http.Request) {
	target, tolerance, vErr := parseDriftParams(r)
	if vErr != nil {
		httputil.WriteQueryValidationError(w, vErr.field, vErr.msg)
		return
	}
	values, snapshotDate, err := h.store.LatestByAccount(r.Context())
	if err != nil {
		h.logger.Error("exposure currency", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	breakdown := map[int]map[string]decimal.Decimal{}
	if h.holdings != nil {
		b, err := h.holdings.AccountCurrencyBreakdownPLN(r.Context())
		if err != nil {
			h.logger.Warn("exposure: holdings breakdown failed, falling back to account.currency",
				"err", err)
		} else {
			breakdown = b
		}
	}
	rows := bucketByCurrency(values, breakdown)
	rep := BuildReport(rows, snapshotDate, target, tolerance)
	httputil.WriteJSON(w, http.StatusOK, rep)
}

type validationErr struct {
	field, msg string
}

func parseDriftParams(r *http.Request) (*float64, float64, *validationErr) {
	tolerance := 5.0
	if v := strings.TrimSpace(r.URL.Query().Get("tolerance")); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || n < 0 {
			return nil, 0, &validationErr{field: "tolerance", msg: "must be a non-negative number"}
		}
		tolerance = n
	}
	v := strings.TrimSpace(r.URL.Query().Get("target_pln_pct"))
	if v == "" {
		return nil, tolerance, nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n < 0 || n > 100 {
		return nil, 0, &validationErr{field: "target_pln_pct", msg: "must be a number between 0 and 100"}
	}
	return &n, tolerance, nil
}
