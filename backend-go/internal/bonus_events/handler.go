package bonusevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
)

var (
	validCurrencies = map[string]struct{}{
		"PLN": {}, "USD": {}, "EUR": {}, "GBP": {}, "CHF": {},
	}
	validBonusTypes = map[string]struct{}{
		"annual": {}, "signon": {}, "spot": {}, "retention": {},
	}
	validContractTypes = map[string]struct{}{
		"UOP": {}, "UZ": {}, "UoD": {}, "B2B": {},
	}
)

// response mirrors backend/app/schemas/bonus_events.BonusEventResponse.
type response struct {
	ID           int      `json:"id"`
	Date         isoDate  `json:"date"`
	Amount       pyFloat  `json:"amount"`
	Currency     string   `json:"currency"`
	Type         string   `json:"type"`
	Company      string   `json:"company"`
	Owner        string   `json:"owner"`
	ContractType string   `json:"contract_type"`
	Notes        *string  `json:"notes"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    isoNaive `json:"created_at"`
	AmountPLN    *pyFloat `json:"amount_pln"`
	FXRate       *pyFloat `json:"fx_rate"`
}

type listResponse struct {
	BonusEvents        []response `json:"bonus_events"`
	TotalCount         int        `json:"total_count"`
	AvailableCompanies []string   `json:"available_companies"`
}

// createRequest captures parsed-and-validated input ready for Store.Create.
type createRequest struct {
	Date         time.Time
	Amount       decimal.Decimal
	Currency     string
	Type         string
	Company      string
	Owner        string
	ContractType string
	Notes        *string
}

// Handler is the HTTP boundary for /api/bonuses.
type Handler struct {
	store  *Store
	fx     *fx.Service
	logger *slog.Logger
}

// NewHandler wires the store, FX service and logger.
func NewHandler(store *Store, fxSvc *fx.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, fx: fxSvc, logger: logger}
}

func (h *Handler) toResponse(ctx context.Context, b *BonusEvent) (response, error) {
	rate, err := h.fx.GetRateToPLN(ctx, b.Currency, b.Date)
	if err != nil {
		return response{}, err
	}
	plnAmount, hasPLN := fx.ToPLN(&b.Amount, b.Currency, rate)

	amt, _ := b.Amount.Float64()
	out := response{
		ID:           b.ID,
		Date:         isoDate(b.Date),
		Amount:       pyFloat(amt),
		Currency:     b.Currency,
		Type:         b.Type,
		Company:      b.Company,
		Owner:        b.Owner,
		ContractType: b.ContractType,
		Notes:        b.Notes,
		IsActive:     b.IsActive,
		CreatedAt:    isoNaive(b.CreatedAt),
	}
	if hasPLN {
		f, _ := plnAmount.Float64()
		pf := pyFloat(f)
		out.AmountPLN = &pf
	}
	if rate.Found {
		f, _ := rate.Rate.Float64()
		pf := pyFloat(f)
		out.FXRate = &pf
	}
	return out, nil
}

// List serves GET /api/bonuses.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, vErr := parseListFilter(r.URL.Query())
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	rows, companies, err := h.store.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list bonuses", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{
		BonusEvents:        make([]response, 0, len(rows)),
		TotalCount:         len(rows),
		AvailableCompanies: companies,
	}
	for i := range rows {
		resp, err := h.toResponse(r.Context(), &rows[i])
		if err != nil {
			h.logger.Error("fx lookup", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		out.BonusEvents = append(out.BonusEvents, resp)
	}
	writeJSON(w, http.StatusOK, out)
}

// Get serves GET /api/bonuses/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	b, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	h.writeBonusResponse(w, r, http.StatusOK, b)
}

// Create serves POST /api/bonuses.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	b := &BonusEvent{
		Date:         req.Date,
		Amount:       req.Amount,
		Currency:     req.Currency,
		Type:         req.Type,
		Company:      req.Company,
		Owner:        req.Owner,
		ContractType: req.ContractType,
		Notes:        req.Notes,
	}
	created, err := h.store.Create(r.Context(), b)
	if err != nil {
		h.logger.Error("create bonus", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.writeBonusResponse(w, r, http.StatusCreated, created)
}

// Update serves PATCH /api/bonuses/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	h.writeBonusResponse(w, r, http.StatusOK, updated)
}

// writeBonusResponse marshals a single bonus with its FX-derived fields,
// emitting a 500 if the FX lookup hits a DB error.
func (h *Handler) writeBonusResponse(w http.ResponseWriter, r *http.Request, status int, b *BonusEvent) {
	resp, err := h.toResponse(r.Context(), b)
	if err != nil {
		h.logger.Error("fx lookup", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, status, resp)
}

// Delete serves DELETE /api/bonuses/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int) {
	if errors.Is(err, ErrNotFound) {
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Bonus event with id %d not found", id))
		return
	}
	h.logger.Error("bonus store", "err", err)
	writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

func parseListFilter(q map[string][]string) (ListFilter, *validationError) {
	f := ListFilter{}
	if v := first(q["owner"]); v != "" {
		s := v
		f.Owner = &s
	}
	if v := first(q["company"]); v != "" {
		s := v
		f.Company = &s
	}
	if v := first(q["date_from"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &validationError{Field: "date_from", Msg: "must be YYYY-MM-DD"}
		}
		f.DateFrom = &t
	}
	if v := first(q["date_to"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &validationError{Field: "date_to", Msg: "must be YYYY-MM-DD"}
		}
		f.DateTo = &t
	}
	return f, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "bonus_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// --- shared wire-format helpers (copied per package; promote when third user needs them) ---

type isoDate time.Time

const isoDateLayout = "2006-01-02"

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

func (d *isoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date must be a string: %w", err)
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}
	*d = isoDate(t)
	return nil
}

type isoNaive time.Time

func (t isoNaive) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02T15:04:05.999999") + `"`), nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
