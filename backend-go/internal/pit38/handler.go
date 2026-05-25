package pit38

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// Store loads the raw lot history grouped per security so the report can
// be computed without N+1 queries.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// LoadAllLots returns one SecurityLots entry per security that has any
// lot in the database, with the lots in chronological order.
func (s *Store) LoadAllLots(ctx context.Context) ([]SecurityLots, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sec.id, sec.symbol, sec.currency,
		       l.id, l.account_id, l.security_id, l.side, l.quantity, l.price,
		       l.fee, l.date, l.created_at
		FROM securities sec
		JOIN lots l ON l.security_id = sec.id
		ORDER BY sec.id, l.date, l.id`)
	if err != nil {
		return nil, fmt.Errorf("pit38 load lots: %w", err)
	}
	defer rows.Close()
	groups := map[int]*SecurityLots{}
	order := []int{}
	for rows.Next() {
		var (
			secID    int
			symbol   string
			currency string
			lot      holdings.Lot
			sideStr  string
		)
		if err := rows.Scan(&secID, &symbol, &currency,
			&lot.ID, &lot.AccountID, &lot.SecurityID, &sideStr, &lot.Quantity,
			&lot.Price, &lot.Fee, &lot.Date, &lot.CreatedAt); err != nil {
			return nil, fmt.Errorf("pit38 scan lot: %w", err)
		}
		lot.Side = holdings.Side(sideStr)
		g, ok := groups[secID]
		if !ok {
			g = &SecurityLots{SecurityID: secID, Symbol: symbol, Currency: currency}
			groups[secID] = g
			order = append(order, secID)
		}
		g.Lots = append(g.Lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pit38 iterate lots: %w", err)
	}
	out := make([]SecurityLots, 0, len(order))
	for _, id := range order {
		g := groups[id]
		if g == nil {
			continue
		}
		out = append(out, *g)
	}
	return out, nil
}

// defaultReportYear picks the calendar year the report should cover when
// the caller doesn't supply one. Polish PIT filings are due by April 30
// for the previous year, so before that date the previous year is the
// most useful default; afterwards the current year is.
func defaultReportYear(now time.Time) int {
	if now.Month() < time.May {
		return now.Year() - 1
	}
	return now.Year()
}

// Handler is the HTTP boundary for /api/pit38.
type Handler struct {
	store  *Store
	fx     *fx.Service
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store, FX service and logger.
func NewHandler(store *Store, fxSvc *fx.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, fx: fxSvc, logger: logger, now: time.Now}
}

// Realized serves GET /api/pit38/realized?year=YYYY[&format=csv].
// JSON is the default; format=csv switches to a downloadable spreadsheet.
//
// Default year follows the PIT filing window: before April 30, last
// calendar year; after, the current year. Mirrors the UI default so
// directly hitting the API yields the same report.
func (h *Handler) Realized(w http.ResponseWriter, r *http.Request) {
	now := h.now().UTC()
	year := defaultReportYear(now)
	if v := r.URL.Query().Get("year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httputil.WriteQueryValidationError(w, "year", "must be an integer")
			return
		}
		year = n
	}
	lots, err := h.store.LoadAllLots(r.Context())
	if err != nil {
		h.logger.Error("pit38 load", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	rep, err := ComputeReport(r.Context(), year, h.fx, lots)
	if err != nil {
		if errors.Is(err, ErrUnknownCurrency) {
			httputil.WriteDetailError(w, http.StatusUnprocessableEntity,
				"NBP rate not available for one of the traded currencies")
			return
		}
		if errors.Is(err, holdings.ErrOversell) {
			httputil.WriteDetailError(w, http.StatusUnprocessableEntity,
				"Lot history oversells a security — clean up the lot log first")
			return
		}
		h.logger.Error("pit38 compute", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		writeCSV(w, rep)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, rep)
}

func writeCSV(w http.ResponseWriter, rep Report) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="pit38-%d.csv"`, rep.Year))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"date", "symbol", "currency", "quantity", "proceeds", "cost_basis",
		"fees", "realized_gain", "fx_rate", "proceeds_pln", "cost_basis_pln",
		"fees_pln", "realized_pln",
	})
	for i := range rep.Rows {
		row := &rep.Rows[i]
		_ = cw.Write([]string{
			time.Time(row.Date).Format("2006-01-02"),
			row.Symbol,
			row.Currency,
			row.Quantity.String(),
			row.Proceeds.StringFixed(2),
			row.CostBasis.StringFixed(2),
			row.Fees.StringFixed(2),
			row.RealizedGain.StringFixed(2),
			row.FXRate.String(),
			row.ProceedsPLN.StringFixed(2),
			row.CostBasisPLN.StringFixed(2),
			row.FeesPLN.StringFixed(2),
			row.RealizedPLN.StringFixed(2),
		})
	}
	_ = cw.Write([]string{
		"TOTAL", "", "", "", "", "", "", "", "",
		rep.Totals.ProceedsPLN.StringFixed(2),
		rep.Totals.CostBasisPLN.StringFixed(2),
		rep.Totals.FeesPLN.StringFixed(2),
		rep.Totals.RealizedPLN.StringFixed(2),
	})
}
