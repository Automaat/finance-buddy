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

// row mirrors one (currency, sum(value)) pair.
type row struct {
	currency string
	valuePLN decimal.Decimal
}

// LatestByCurrency returns total PLN value per account currency from the
// most recent snapshot. Active accounts only — soft-deleted ones are
// excluded so the chart matches what the user can actually invest from.
func (s *Store) LatestByCurrency(ctx context.Context) ([]row, string, error) {
	var (
		latestID   int
		latestDate string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, to_char(date, 'YYYY-MM-DD')
		FROM snapshots ORDER BY date DESC, id DESC LIMIT 1`).Scan(&latestID, &latestDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return []row{}, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("latest snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT UPPER(a.currency) AS currency, SUM(sv.value)
		FROM snapshot_values sv
		JOIN accounts a ON a.id = sv.account_id
		WHERE sv.snapshot_id = $1 AND a.is_active = true
		GROUP BY UPPER(a.currency)
	`, latestID)
	if err != nil {
		return nil, "", fmt.Errorf("query currency totals: %w", err)
	}
	defer rows.Close()
	out := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.currency, &r.valuePLN); err != nil {
			return nil, "", fmt.Errorf("scan currency total: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate currency totals: %w", err)
	}
	return out, latestDate, nil
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

// Currency serves GET /api/exposure/currency[?target_pln_pct=X&tolerance=Y].
// Both query params are optional; omitting target_pln_pct skips the drift
// block entirely.
func (h *Handler) Currency(w http.ResponseWriter, r *http.Request) {
	target, tolerance, vErr := parseDriftParams(r)
	if vErr != nil {
		httputil.WriteQueryValidationError(w, vErr.field, vErr.msg)
		return
	}
	rows, snapshotDate, err := h.store.LatestByCurrency(r.Context())
	if err != nil {
		h.logger.Error("exposure currency", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
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
