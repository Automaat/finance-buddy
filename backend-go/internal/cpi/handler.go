package cpi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	Year            int     `json:"year"`
	YoYRate         pyFloat `json:"yoy_rate"`
	CumulativeIndex pyFloat `json:"cumulative_index"`
}

type seriesResponse struct {
	Points     []cpiPoint `json:"points"`
	BaseYear   *int       `json:"base_year"`
	LatestYear *int       `json:"latest_year"`
	Source     string     `json:"source"`
}

type adjustResponse struct {
	OriginalAmount pyFloat `json:"original_amount"`
	AdjustedAmount pyFloat `json:"adjusted_amount"`
	Factor         pyFloat `json:"factor"`
	FromDate       isoDate `json:"from_date"`
	ToDate         isoDate `json:"to_date"`
	AsOfYear       int     `json:"as_of_year"`
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
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
			YoYRate:         pyFloat(yoy),
			CumulativeIndex: pyFloat(idx),
		})
	}
	out := seriesResponse{Points: points, Source: GUSSourceTag}
	if len(points) > 0 {
		base := points[0].Year
		latest := points[len(points)-1].Year
		out.BaseYear = &base
		out.LatestYear = &latest
	}
	writeJSON(w, http.StatusOK, out)
}

// Adjust serves POST /api/cpi/adjust.
func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	amount, vErr := requireFloat(raw, "amount")
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	from, vErr := requireDate(raw, "from_date")
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	to, vErr := requireDate(raw, "to_date")
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}

	yoyMap, err := h.store.LoadYoYMap(r.Context())
	if err != nil {
		h.logger.Error("load cpi", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	indexMap := CumulativeIndex(yoyMap)
	adjusted, err := AdjustWithIndex(indexMap, amount, from, to)
	if err != nil {
		if errors.Is(err, ErrInflationDataMissing) {
			writeDetailError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		h.logger.Error("adjust", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	latest, ok, err := h.store.LatestKnownYear(r.Context())
	if err != nil {
		h.logger.Error("latest year", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !ok {
		writeDetailError(w, http.StatusServiceUnavailable, "CPI table is empty")
		return
	}
	factor := 0.0
	if amount != 0 {
		factor = adjusted / amount
	}
	writeJSON(w, http.StatusOK, adjustResponse{
		OriginalAmount: pyFloat(amount),
		AdjustedAmount: pyFloat(adjusted),
		Factor:         pyFloat(factor),
		FromDate:       isoDate(from),
		ToDate:         isoDate(to),
		AsOfYear:       latest,
	})
}

// Refresh serves POST /api/cpi/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	rows, err := h.fetcher.Fetch(r.Context())
	if err != nil {
		writeDetailError(w, http.StatusBadGateway, "GUS BDL fetch failed: "+err.Error())
		return
	}
	written, err := h.store.Upsert(r.Context(), GUSSourceTag, rows)
	if err != nil {
		h.logger.Error("upsert cpi", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	latest, ok, err := h.store.LatestKnownYear(r.Context())
	if err != nil {
		h.logger.Error("latest year", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := refreshResponse{RowsWritten: written}
	if ok {
		y := latest
		out.LatestYear = &y
	}
	writeJSON(w, http.StatusOK, out)
}

// --- validation ---

func requireFloat(raw map[string]json.RawMessage, key string) (float64, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	return f, nil
}

func requireDate(raw map[string]json.RawMessage, key string) (time.Time, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return time.Time{}, &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be a string"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be YYYY-MM-DD"}
	}
	return t, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

// --- wire types (copies — promote when shared helper lands) ---

type isoDate time.Time

const isoDateLayout = "2006-01-02"

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

func (d *isoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return err
	}
	*d = isoDate(t)
	return nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
