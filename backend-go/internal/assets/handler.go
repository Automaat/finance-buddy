package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

type response struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	IsActive     bool          `json:"is_active"`
	CreatedAt    wire.IsoNaive `json:"created_at"`
	CurrentValue wire.PyFloat  `json:"current_value"`
}

type listResponse struct {
	Assets []response `json:"assets"`
}

// Handler is the HTTP boundary for /api/assets.
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

func toResponse(a *Asset, current decimal.Decimal) response {
	cv, _ := current.Float64()
	return response{
		ID:           a.ID,
		Name:         a.Name,
		IsActive:     a.IsActive,
		CreatedAt:    wire.IsoNaive(a.CreatedAt.UTC()),
		CurrentValue: wire.PyFloat(cv),
	}
}

// List serves GET /api/assets.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list assets", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ids := make([]int, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	values, err := h.store.LatestValuesByAsset(r.Context(), ids)
	if err != nil {
		h.logger.Error("latest asset values", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Assets: make([]response, 0, len(rows))}
	for i := range rows {
		a := &rows[i]
		out.Assets = append(out.Assets, toResponse(a, values[a.ID]))
	}
	writeJSON(w, http.StatusOK, out)
}

// Create serves POST /api/assets.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	name, vErr := requireName(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrDuplicateName) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Asset with name '%s' already exists", name))
			return
		}
		h.logger.Error("create asset", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created, decimal.Zero))
}

// Update serves PUT /api/assets/{id}.
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
	namePtr, vErr := optionalName(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, namePtr)
	if err != nil {
		h.writeStoreError(w, err, id, namePtr)
		return
	}
	cv, err := h.store.LatestValueForAsset(r.Context(), id)
	if err != nil {
		h.logger.Error("latest asset value", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated, cv))
}

// Delete serves DELETE /api/assets/{id}.
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
			fmt.Sprintf("Asset with id %d not found", id))
	case errors.Is(err, ErrDuplicateName):
		n := ""
		if name != nil {
			n = *name
		}
		writeDetailError(w, http.StatusConflict,
			fmt.Sprintf("Asset with name '%s' already exists", n))
	default:
		h.logger.Error("asset store", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "asset_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}
