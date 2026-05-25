package recurring

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler is the HTTP boundary for /api/recurring.
type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger, now: time.Now}
}

type response struct {
	ID              int      `json:"id"`
	AccountID       int      `json:"account_id"`
	Amount          string   `json:"amount"`
	OwnerUserID     *int     `json:"owner_user_id"`
	TransactionType *string  `json:"transaction_type"`
	Category        *string  `json:"category"`
	Description     string   `json:"description"`
	Frequency       string   `json:"frequency"`
	DayOfMonth      *int     `json:"day_of_month"`
	StartDate       string   `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	Active          bool     `json:"active"`
	SkippedDates    []string `json:"skipped_dates"`
	LastRunDate     *string  `json:"last_run_date"`
	NextOccurrence  *string  `json:"next_occurrence"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type listResponse struct {
	Recurring []response `json:"recurring"`
}

type runNowResponse struct {
	TransactionID *int   `json:"transaction_id"`
	Date          string `json:"date"`
	AlreadyMinted bool   `json:"already_minted"`
}

func formatDate(t time.Time) string { return t.UTC().Format("2006-01-02") }
func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatDate(*t)
	return &s
}

// formatTimestamp matches the wire.IsoNaive layout used across the rest of
// the Go API (microsecond precision, naive UTC). Cf. internal/wire/types.go.
func formatTimestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.999999") }

func (h *Handler) toResponse(r Recurring) response {
	skips := make([]string, 0, len(r.SkippedDates))
	for _, s := range r.SkippedDates {
		skips = append(skips, formatDate(s))
	}
	var next *string
	if n, ok := NextOccurrence(r, h.now().UTC()); ok {
		s := formatDate(n)
		next = &s
	}
	return response{
		ID:              r.ID,
		AccountID:       r.AccountID,
		Amount:          r.Amount.StringFixed(2),
		OwnerUserID:     r.OwnerUserID,
		TransactionType: r.TransactionType,
		Category:        r.Category,
		Description:     r.Description,
		Frequency:       string(r.Frequency),
		DayOfMonth:      r.DayOfMonth,
		StartDate:       formatDate(r.StartDate),
		EndDate:         formatDatePtr(r.EndDate),
		Active:          r.Active,
		SkippedDates:    skips,
		LastRunDate:     formatDatePtr(r.LastRunDate),
		NextOccurrence:  next,
		CreatedAt:       formatTimestamp(r.CreatedAt),
		UpdatedAt:       formatTimestamp(r.UpdatedAt),
	}
}

// List serves GET /api/recurring.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list recurring", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, h.toResponse(rows[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurring": out})
}

// Create serves POST /api/recurring.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		writeValidation(w, "body", "Invalid JSON body")
		return
	}
	in, vErr := buildInput(raw)
	if vErr != nil {
		writeValidation(w, vErr.Field, vErr.Msg)
		return
	}
	exists, err := h.store.AccountExists(r.Context(), in.AccountID)
	if err != nil {
		h.logger.Error("account check", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "Account not found or inactive")
		return
	}
	created, err := h.store.Create(r.Context(), in)
	if err != nil {
		h.logger.Error("create recurring", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, h.toResponse(created))
}

// Get serves GET /api/recurring/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	row, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(row))
}

// Update serves PUT /api/recurring/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	raw, err := readBody(r)
	if err != nil {
		writeValidation(w, "body", "Invalid JSON body")
		return
	}
	in, vErr := buildInput(raw)
	if vErr != nil {
		writeValidation(w, vErr.Field, vErr.Msg)
		return
	}
	exists, err := h.store.AccountExists(r.Context(), in.AccountID)
	if err != nil {
		h.logger.Error("account check", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "Account not found or inactive")
		return
	}
	updated, err := h.store.Update(r.Context(), id, in)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(updated))
}

// Delete serves DELETE /api/recurring/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RunNow serves POST /api/recurring/{id}/run-now — mints an ad-hoc
// transaction from the template, dated today.
func (h *Handler) RunNow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	row, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	today := h.now().UTC().Truncate(24 * time.Hour)
	txID, err := h.store.MintOccurrence(r.Context(), row, today)
	if err != nil && !IsAlreadyMinted(err) {
		h.logger.Error("run-now mint", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resp := runNowResponse{Date: formatDate(today), AlreadyMinted: IsAlreadyMinted(err)}
	if !resp.AlreadyMinted {
		resp.TransactionID = &txID
	}
	status := http.StatusCreated
	if resp.AlreadyMinted {
		status = http.StatusOK
	}
	writeJSON(w, status, resp)
}

// Skip serves POST /api/recurring/{id}/skip — adds the supplied date (or
// today's next occurrence by default) to skipped_dates.
func (h *Handler) Skip(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	row, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	date := r.URL.Query().Get("date")
	var target time.Time
	if date == "" {
		next, found := NextOccurrence(row, h.now().UTC())
		if !found {
			writeError(w, http.StatusBadRequest, "No upcoming occurrence to skip")
			return
		}
		target = next
	} else {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			writeValidation(w, "date", "must be YYYY-MM-DD")
			return
		}
		target = t
	}
	if err := h.store.AppendSkip(r.Context(), id, target); err != nil {
		h.writeStoreError(w, err)
		return
	}
	updated, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(updated))
}

// Unskip serves POST /api/recurring/{id}/unskip?date=YYYY-MM-DD — removes
// the date from skipped_dates so the cadence resumes generating it.
func (h *Handler) Unskip(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		writeValidation(w, "date", "required")
		return
	}
	target, err := time.Parse("2006-01-02", date)
	if err != nil {
		writeValidation(w, "date", "must be YYYY-MM-DD")
		return
	}
	if err := h.store.RemoveSkip(r.Context(), id, target); err != nil {
		h.writeStoreError(w, err)
		return
	}
	updated, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(updated))
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Recurring transaction not found")
		return
	}
	h.logger.Error("recurring store", "err", err)
	writeError(w, http.StatusInternalServerError, "Internal Server Error")
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		writeValidation(w, "id", "must be a positive integer")
		return 0, false
	}
	return id, true
}

func readBody(r *http.Request) (map[string]json.RawMessage, error) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func writeValidation(w http.ResponseWriter, field, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"detail": []map[string]any{
			{"type": "value_error", "loc": []string{"body", field}, "msg": msg},
		},
	})
}
