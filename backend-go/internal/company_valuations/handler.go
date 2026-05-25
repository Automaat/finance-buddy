package companyvaluations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

var (
	validCurrencies = map[string]struct{}{
		"PLN": {}, "USD": {}, "EUR": {}, "GBP": {}, "CHF": {},
	}
	validSources = map[string]struct{}{
		"409a": {}, "preferred_round": {}, "tender": {}, "estimate": {},
	}
)

// response mirrors backend/app/schemas/company_valuations.CompanyValuationResponse.
//
// Money fields use wire.PyFloat so JSON output preserves Python's `float(x)`
// formatting — `14.0` not `14`, matching Pydantic's number serialization.
type response struct {
	ID                     int           `json:"id"`
	Company                string        `json:"company"`
	Date                   wire.IsoDate  `json:"date"`
	Currency               string        `json:"currency"`
	FMVPerShare            wire.PyFloat  `json:"fmv_per_share"`
	FMVLow                 *wire.PyFloat `json:"fmv_low"`
	FMVHigh                *wire.PyFloat `json:"fmv_high"`
	Source                 string        `json:"source"`
	CommonStockDiscountPct *wire.PyFloat `json:"common_stock_discount_pct"`
	Notes                  *string       `json:"notes"`
	IsActive               bool          `json:"is_active"`
	CreatedAt              wire.IsoNaive `json:"created_at"`
}

type listResponse struct {
	CompanyValuations  []response `json:"company_valuations"`
	TotalCount         int        `json:"total_count"`
	AvailableCompanies []string   `json:"available_companies"`
}

// createRequest captures parsed-and-validated input ready for Store.Create.
// The JSON body is read into a raw map first so we can detect missing
// required fields and preserve Numeric column precision by feeding JSON
// number tokens directly into decimal.NewFromString — going through float64
// introduces IEEE754 rounding that diverges from Python's Decimal(str(...)).
type createRequest struct {
	Company                string
	Date                   time.Time
	Currency               string
	FMVPerShare            decimal.Decimal
	FMVLow                 *decimal.Decimal
	FMVHigh                *decimal.Decimal
	Source                 string
	CommonStockDiscountPct *decimal.Decimal
	Notes                  *string
}

// Handler is the HTTP boundary for /api/company-valuations.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires the store and logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

func toResponse(v *Valuation) response {
	fmv, _ := v.FMVPerShare.Float64()
	out := response{
		ID:          v.ID,
		Company:     v.Company,
		Date:        wire.IsoDate(v.Date),
		Currency:    v.Currency,
		FMVPerShare: wire.PyFloat(fmv),
		Source:      v.Source,
		Notes:       v.Notes,
		IsActive:    v.IsActive,
		CreatedAt:   wire.IsoNaive(v.CreatedAt),
	}
	if v.FMVLow != nil {
		f, _ := v.FMVLow.Float64()
		pf := wire.PyFloat(f)
		out.FMVLow = &pf
	}
	if v.FMVHigh != nil {
		f, _ := v.FMVHigh.Float64()
		pf := wire.PyFloat(f)
		out.FMVHigh = &pf
	}
	if v.CommonStockDiscountPct != nil {
		f, _ := v.CommonStockDiscountPct.Float64()
		pf := wire.PyFloat(f)
		out.CommonStockDiscountPct = &pf
	}
	return out
}

// List serves GET /api/company-valuations.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{}
	if c := r.URL.Query().Get("company"); c != "" {
		filter.Company = &c
	}
	rows, companies, err := h.store.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list valuations", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{
		CompanyValuations:  make([]response, 0, len(rows)),
		TotalCount:         len(rows),
		AvailableCompanies: companies,
	}
	for i := range rows {
		out.CompanyValuations = append(out.CompanyValuations, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Get serves GET /api/company-valuations/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	v, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(v))
}

// Create serves POST /api/company-valuations.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	v := requestToValuation(&req)
	created, err := h.store.Create(r.Context(), v)
	if err != nil {
		h.logger.Error("create valuation", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PATCH /api/company-valuations/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/company-valuations/{id}.
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
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Company valuation with id %d not found", id))
	case errors.Is(err, ErrRangeLow):
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "fmv_low cannot exceed fmv_per_share")
	case errors.Is(err, ErrRangeHigh):
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "fmv_high cannot be below fmv_per_share")
	default:
		h.logger.Error("valuation store", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func requestToValuation(req *createRequest) *Valuation {
	return &Valuation{
		Company:                req.Company,
		Date:                   req.Date,
		Currency:               req.Currency,
		FMVPerShare:            req.FMVPerShare,
		FMVLow:                 req.FMVLow,
		FMVHigh:                req.FMVHigh,
		Source:                 req.Source,
		CommonStockDiscountPct: req.CommonStockDiscountPct,
		Notes:                  req.Notes,
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		httputil.WriteBodyValidationError(w, "valuation_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}
