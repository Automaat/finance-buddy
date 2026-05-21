package accounts

import (
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
)

var (
	validAccountTypes = map[string]struct{}{"asset": {}, "liability": {}}
	validCategories   = map[string]struct{}{
		"bank": {}, "saving_account": {}, "stock": {}, "bond": {}, "gold": {},
		"real_estate": {}, "ppk": {}, "fund": {}, "etf": {}, "vehicle": {},
		"mortgage": {}, "installment": {},
	}
	validPurposes = map[string]struct{}{
		"retirement": {}, "emergency_fund": {}, "general": {},
	}
	validWrappers = map[string]struct{}{
		"IKE": {}, "IKZE": {}, "PPK": {},
	}
)

type response struct {
	ID                    int      `json:"id"`
	Name                  string   `json:"name"`
	Type                  string   `json:"type"`
	Category              string   `json:"category"`
	Owner                 string   `json:"owner"`
	Currency              string   `json:"currency"`
	AccountWrapper        *string  `json:"account_wrapper"`
	Purpose               string   `json:"purpose"`
	SquareMeters          *pyFloat `json:"square_meters"`
	IsActive              bool     `json:"is_active"`
	ReceivesContributions bool     `json:"receives_contributions"`
	CreatedAt             isoNaive `json:"created_at"`
	CurrentValue          pyFloat  `json:"current_value"`
}

type listResponse struct {
	Assets      []response `json:"assets"`
	Liabilities []response `json:"liabilities"`
}

// Handler is the HTTP boundary for /api/accounts.
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

func toResponse(a *Account, currentValue decimal.Decimal) response {
	cv, _ := currentValue.Float64()
	out := response{
		ID:                    a.ID,
		Name:                  a.Name,
		Type:                  a.Type,
		Category:              a.Category,
		Owner:                 a.Owner,
		Currency:              a.Currency,
		AccountWrapper:        a.AccountWrapper,
		Purpose:               a.Purpose,
		IsActive:              a.IsActive,
		ReceivesContributions: a.ReceivesContributions,
		CreatedAt:             isoNaive(a.CreatedAt.UTC()),
		CurrentValue:          pyFloat(cv),
	}
	if a.SquareMeters != nil {
		f, _ := a.SquareMeters.Float64()
		pf := pyFloat(f)
		out.SquareMeters = &pf
	}
	return out
}

// List serves GET /api/accounts.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list accounts", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ids := make([]int, 0, len(accounts))
	for i := range accounts {
		ids = append(ids, accounts[i].ID)
	}
	values, err := h.store.LatestValuesByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("latest values", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Assets: []response{}, Liabilities: []response{}}
	for i := range accounts {
		a := &accounts[i]
		cv := values[a.ID]
		resp := toResponse(a, cv)
		if a.Type == "asset" {
			out.Assets = append(out.Assets, resp)
		} else {
			out.Liabilities = append(out.Liabilities, resp)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// Create serves POST /api/accounts.
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
	created, err := h.store.Create(r.Context(), &Account{
		Name:                  req.Name,
		Type:                  req.Type,
		Category:              req.Category,
		Owner:                 req.Owner,
		Currency:              req.Currency,
		AccountWrapper:        req.AccountWrapper,
		Purpose:               req.Purpose,
		SquareMeters:          req.SquareMeters,
		ReceivesContributions: req.ReceivesContributions,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateName) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Account with name '%s' already exists", req.Name))
			return
		}
		h.logger.Error("create account", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created, decimal.Zero))
}

// Update serves PUT /api/accounts/{id}.
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
		h.writeStoreError(w, err, id, patch.Name)
		return
	}
	cv, err := h.store.LatestValueForAccount(r.Context(), id)
	if err != nil {
		h.logger.Error("latest value", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated, cv))
}

// Delete serves DELETE /api/accounts/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err, id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int, name *string) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrDuplicateName):
		n := ""
		if name != nil {
			n = *name
		}
		writeDetailError(w, http.StatusConflict,
			fmt.Sprintf("Account with name '%s' already exists", n))
	default:
		h.logger.Error("account store", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "account_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// --- shared wire types ---

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
