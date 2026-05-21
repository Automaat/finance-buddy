package personas

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

// response mirrors backend/app/schemas/personas.PersonaResponse byte-for-byte.
type response struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	PPKEmployeeRate ppkRate  `json:"ppk_employee_rate"`
	PPKEmployerRate ppkRate  `json:"ppk_employer_rate"`
	CreatedAt       isoNaive `json:"created_at"`
}

type createRequest struct {
	Name            string           `json:"name"`
	PPKEmployeeRate *decimal.Decimal `json:"ppk_employee_rate"`
	PPKEmployerRate *decimal.Decimal `json:"ppk_employer_rate"`
}

type updateRequest struct {
	Name            *string          `json:"name"`
	PPKEmployeeRate *decimal.Decimal `json:"ppk_employee_rate"`
	PPKEmployerRate *decimal.Decimal `json:"ppk_employer_rate"`
}

// Handler is the HTTP boundary for /api/personas.
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

func toResponse(p *Persona) response {
	return response{
		ID:              p.ID,
		Name:            p.Name,
		PPKEmployeeRate: ppkRate(p.PPKEmployeeRate),
		PPKEmployerRate: ppkRate(p.PPKEmployerRate),
		CreatedAt:       isoNaive(p.CreatedAt),
	}
}

// List serves GET /api/personas.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list personas", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

// Create serves POST /api/personas.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	name, vErr := validateName(req.Name)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	employee, vErr := resolveRate(req.PPKEmployeeRate, defaultEmployeeRate, "ppk_employee_rate")
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	employer, vErr := resolveRate(req.PPKEmployerRate, defaultEmployerRate, "ppk_employer_rate")
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	p, err := h.store.Create(r.Context(), name, employee, employer)
	if err != nil {
		if errors.Is(err, ErrNameConflict) {
			writeDetailError(w, http.StatusConflict, fmt.Sprintf("Persona '%s' already exists", name))
			return
		}
		h.logger.Error("create persona", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(p))
}

// Update serves PUT /api/personas/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	var newName *string
	if req.Name != nil {
		stripped, vErr := validateName(*req.Name)
		if vErr != nil {
			writePydanticError(w, vErr)
			return
		}
		newName = &stripped
	}
	if req.PPKEmployeeRate != nil {
		if vErr := validatePPKRange(*req.PPKEmployeeRate, "ppk_employee_rate"); vErr != nil {
			writePydanticError(w, vErr)
			return
		}
	}
	if req.PPKEmployerRate != nil {
		if vErr := validatePPKRange(*req.PPKEmployerRate, "ppk_employer_rate"); vErr != nil {
			writePydanticError(w, vErr)
			return
		}
	}

	p, err := h.store.Update(r.Context(), id, newName, req.PPKEmployeeRate, req.PPKEmployerRate)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeDetailError(w, http.StatusNotFound, "Persona not found")
		case errors.Is(err, ErrNameConflict):
			conflictName := ""
			if newName != nil {
				conflictName = *newName
			}
			writeDetailError(w, http.StatusConflict, fmt.Sprintf("Persona '%s' already exists", conflictName))
		default:
			h.logger.Error("update persona", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	writeJSON(w, http.StatusOK, toResponse(p))
}

// Delete serves DELETE /api/personas/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	err := h.store.Delete(r.Context(), id)
	if err != nil {
		var conflict *ConflictError
		switch {
		case errors.Is(err, ErrNotFound):
			writeDetailError(w, http.StatusNotFound, "Persona not found")
		case errors.As(err, &conflict):
			writeDetailError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"Cannot delete persona '%s': referenced by %s",
					conflict.Name, strings.Join(conflict.Details, ", "),
				),
			)
		default:
			h.logger.Error("delete persona", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "persona_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// ppkRate is decimal.Decimal that marshals as a quoted JSON string with two
// decimals — matches Pydantic v2's Decimal serialization for Numeric(5,2).
type ppkRate decimal.Decimal

func (r ppkRate) MarshalJSON() ([]byte, error) {
	d := decimal.Decimal(r)
	return []byte(`"` + d.StringFixed(2) + `"`), nil
}

// isoNaive serializes a time.Time as ISO without a timezone suffix and trims
// trailing zeros from the microsecond fraction — matches Python's default
// datetime.isoformat() on naive datetimes that the persona table stores.
type isoNaive time.Time

func (t isoNaive) MarshalJSON() ([]byte, error) {
	s := time.Time(t).Format("2006-01-02T15:04:05.999999")
	return []byte(`"` + s + `"`), nil
}
