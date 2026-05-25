package cpi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// Handler is the HTTP boundary for /api/cpi.
type Handler struct {
	store   *Store
	fetcher *GUSFetcher
	logger  *slog.Logger
}

// NewHandler wires the dependencies.
func NewHandler(store *Store, fetcher *GUSFetcher, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if fetcher == nil {
		fetcher = NewGUSFetcher()
	}
	return &Handler{store: store, fetcher: fetcher, logger: logger}
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
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	amount, vErr := requireFloat(raw, "amount")
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	from, vErr := requireDate(raw, "from_date")
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	to, vErr := requireDate(raw, "to_date")
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

// --- validation ---

func requireFloat(raw map[string]json.RawMessage, key string) (float64, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return f, nil
}

func requireDate(raw map[string]json.RawMessage, key string) (time.Time, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be YYYY-MM-DD"}
	}
	return t, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
