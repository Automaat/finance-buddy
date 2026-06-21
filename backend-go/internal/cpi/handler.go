package cpi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// MonthlyFetcher is the slice of cpi.EurostatHICPFetcher / cpi.FREDFetcher
// the manual refresh endpoint accepts. Declared as an interface so the
// handler doesn't care which monthly source the scheduler resolved.
type MonthlyFetcher interface {
	FetchSince(ctx context.Context, since time.Time) ([]MonthYearRate, error)
}

// monthlyBackfillStart mirrors scheduler.monthlyBackfillStart. Duplicated
// here rather than imported to avoid a handler→scheduler dependency
// inversion (scheduler imports cpi, not the other way round).
var monthlyBackfillStart = time.Date(2013, time.January, 1, 0, 0, 0, 0, time.UTC)

// Handler is the HTTP boundary for /api/cpi.
type Handler struct {
	store          *Store
	fetcher        *GUSFetcher
	monthlyFetcher MonthlyFetcher
	monthlySource  string
	logger         *slog.Logger
}

// NewHandler wires the dependencies. monthlyFetcher and monthlySource may
// be empty in setups without a monthly source — Refresh-monthly then
// returns 503.
func NewHandler(store *Store, fetcher *GUSFetcher, monthlyFetcher MonthlyFetcher, monthlySource string, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if fetcher == nil {
		fetcher = NewGUSFetcher()
	}
	return &Handler{
		store:          store,
		fetcher:        fetcher,
		monthlyFetcher: monthlyFetcher,
		monthlySource:  monthlySource,
		logger:         logger,
	}
}

type cpiPoint struct {
	Year            int          `json:"year"`
	YoYRate         wire.PyFloat `json:"yoy_rate"`
	CumulativeIndex wire.PyFloat `json:"cumulative_index"`
}

type seriesResponse struct {
	Points     []cpiPoint `json:"points"`
	BaseYear   *int       `json:"base_year"`
	LatestYear *int       `json:"latest_year"`
	Source     string     `json:"source"`
}

type adjustResponse struct {
	OriginalAmount wire.PyFloat `json:"original_amount"`
	AdjustedAmount wire.PyFloat `json:"adjusted_amount"`
	Factor         wire.PyFloat `json:"factor"`
	FromDate       wire.IsoDate `json:"from_date"`
	ToDate         wire.IsoDate `json:"to_date"`
	AsOfYear       int          `json:"as_of_year"`
}

type refreshResponse struct {
	RowsWritten int  `json:"rows_written"`
	LatestYear  *int `json:"latest_year"`
}

// GetSeries serves GET /api/cpi/series.
func (h *Handler) GetSeries(w http.ResponseWriter, r *http.Request) {
	yoyMap, err := h.store.LoadYoYMap(r.Context())
	if err != nil {
		h.logger.Error("load cpi", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	indexMap := CumulativeIndex(yoyMap)
	years := sortedYears(indexMap)
	points := make([]cpiPoint, 0, len(years))
	for _, y := range years {
		yoy, _ := yoyMap[y].Float64()
		idx, _ := indexMap[y].Float64()
		points = append(points, cpiPoint{
			Year:            y,
			YoYRate:         wire.PyFloat(yoy),
			CumulativeIndex: wire.PyFloat(idx),
		})
	}
	out := seriesResponse{Points: points, Source: GUSSourceTag}
	if len(points) > 0 {
		base := points[0].Year
		latest := points[len(points)-1].Year
		out.BaseYear = &base
		out.LatestYear = &latest
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Adjust serves POST /api/cpi/adjust.
func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	amount, vErr := requireFloat(raw, "amount")
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	from, vErr := validation.RequiredDate(raw, "from_date")
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	to, vErr := validation.RequiredDate(raw, "to_date")
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}

	yoyMap, err := h.store.LoadYoYMap(r.Context())
	if err != nil {
		h.logger.Error("load cpi", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	indexMap := CumulativeIndex(yoyMap)
	adjusted, err := AdjustWithIndex(indexMap, amount, from, to)
	if err != nil {
		if errors.Is(err, ErrInflationDataMissing) {
			httputil.WriteDetailError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		h.logger.Error("adjust", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	latest, ok, err := h.store.LatestKnownYear(r.Context())
	if err != nil {
		h.logger.Error("latest year", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !ok {
		httputil.WriteDetailError(w, http.StatusServiceUnavailable, "CPI table is empty")
		return
	}
	factor := 0.0
	if amount != 0 {
		factor = adjusted / amount
	}
	httputil.WriteJSON(w, http.StatusOK, adjustResponse{
		OriginalAmount: wire.PyFloat(amount),
		AdjustedAmount: wire.PyFloat(adjusted),
		Factor:         wire.PyFloat(factor),
		FromDate:       wire.IsoDate(from),
		ToDate:         wire.IsoDate(to),
		AsOfYear:       latest,
	})
}

// Refresh serves POST /api/cpi/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	rows, err := h.fetcher.Fetch(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusBadGateway, "GUS BDL fetch failed: "+err.Error())
		return
	}
	written, err := h.store.Upsert(r.Context(), GUSSourceTag, rows)
	if err != nil {
		h.logger.Error("upsert cpi", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	latest, ok, err := h.store.LatestKnownYear(r.Context())
	if err != nil {
		h.logger.Error("latest year", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := refreshResponse{RowsWritten: written}
	if ok {
		y := latest
		out.LatestYear = &y
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// refreshMonthlyResponse mirrors refreshResponse for the monthly path —
// LatestMonth is YYYY-MM ("" when the table is still empty).
type refreshMonthlyResponse struct {
	RowsWritten int    `json:"rows_written"`
	Source      string `json:"source"`
	LatestMonth string `json:"latest_month"`
}

// RefreshMonthly serves POST /api/cpi/refresh-monthly. Mirrors the
// existing annual /api/cpi/refresh: fetches from the wired monthly source
// (FRED or Eurostat) and upserts. Used to backfill after a source change
// or to short-circuit the scheduler's 7-day staleness gate.
func (h *Handler) RefreshMonthly(w http.ResponseWriter, r *http.Request) {
	if h.monthlyFetcher == nil {
		httputil.WriteDetailError(w, http.StatusServiceUnavailable, "Monthly CPI source not configured")
		return
	}
	rows, err := h.monthlyFetcher.FetchSince(r.Context(), monthlyBackfillStart)
	if err != nil {
		httputil.WriteDetailError(w, http.StatusBadGateway, "Monthly CPI fetch failed: "+err.Error())
		return
	}
	written, err := h.store.UpsertMonthly(r.Context(), h.monthlySource, rows)
	if err != nil {
		h.logger.Error("upsert monthly cpi", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := refreshMonthlyResponse{RowsWritten: written, Source: h.monthlySource}
	// Find the highest (year, month) in the fetched batch.
	for _, r := range rows {
		stamp := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		if stamp > out.LatestMonth {
			out.LatestMonth = stamp
		}
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// --- validation ---

func requireFloat(raw map[string]json.RawMessage, key string) (float64, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return f, nil
}
