package scenarios

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
)

// response mirrors the wire shape returned to the frontend. CreatedAt /
// UpdatedAt are naive timestamps (no tz suffix) to stay consistent with
// the rest of the API where the DB stores `timestamp without time zone`.
type response struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	InputsJSON json.RawMessage `json:"inputs_json"`
	CreatedAt  isoNaive        `json:"created_at"`
	UpdatedAt  isoNaive        `json:"updated_at"`
}

type listResponse struct {
	Scenarios []response `json:"scenarios"`
}

type createRequest struct {
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	InputsJSON json.RawMessage `json:"inputs_json"`
}

type updateRequest struct {
	Name       string          `json:"name"`
	InputsJSON json.RawMessage `json:"inputs_json"`
}

type cloneRequest struct {
	Name string `json:"name"`
}

// Handler is the HTTP boundary for /api/scenarios.
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

// List serves GET /api/scenarios?kind=<kind>.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind != "" {
		if _, ok := validKinds[kind]; !ok {
			writeValidationError(w, "kind", "Unknown kind", kind)
			return
		}
	}
	rows, err := h.store.List(r.Context(), kind)
	if err != nil {
		h.logger.Error("scenarios list", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, listResponse{Scenarios: out})
}

// Get serves GET /api/scenarios/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	sc, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Scenario not found")
			return
		}
		h.logger.Error("scenarios get", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(sc))
}

// Create serves POST /api/scenarios.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxInputsBytes+4096)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if vErr := validateCreate(req.Name, req.Kind, req.InputsJSON); vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	sc, err := h.store.Create(r.Context(), strings.TrimSpace(req.Name), req.Kind, req.InputsJSON)
	if err != nil {
		h.logger.Error("scenarios create", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(sc))
}

// Update serves PUT /api/scenarios/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxInputsBytes+4096)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if vErr := validateUpdate(req.Name, req.InputsJSON); vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	sc, err := h.store.Update(r.Context(), id, strings.TrimSpace(req.Name), req.InputsJSON)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Scenario not found")
			return
		}
		h.logger.Error("scenarios update", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(sc))
}

// Clone serves POST /api/scenarios/{id}/clone. Body is optional — when
// `name` is omitted or blank, the clone gets the source name with a
// " (copy)" suffix.
func (h *Handler) Clone(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req cloneRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
			writeValidationError(w, "body", "Invalid JSON body", err.Error())
			return
		}
	}
	if vErr := validateCloneName(req.Name); vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	name := strings.TrimSpace(req.Name)
	var (
		sc  *Scenario
		err error
	)
	if name == "" {
		sc, err = h.store.CloneWithSuffix(r.Context(), id)
	} else {
		sc, err = h.store.Clone(r.Context(), id, name)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Scenario not found")
			return
		}
		h.logger.Error("scenarios clone", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(sc))
}

// Delete serves DELETE /api/scenarios/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Scenario not found")
			return
		}
		h.logger.Error("scenarios delete", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		writeDetailError(w, http.StatusNotFound, "Scenario not found")
		return 0, false
	}
	return id, true
}

func toResponse(sc *Scenario) response {
	return response{
		ID:         sc.ID,
		Name:       sc.Name,
		Kind:       sc.Kind,
		InputsJSON: sc.InputsJSON,
		CreatedAt:  isoNaive(sc.CreatedAt),
		UpdatedAt:  isoNaive(sc.UpdatedAt),
	}
}

// isoNaive marshals a time.Time as a naive ISO-8601 timestamp (no zone)
// matching the rest of the codebase's `timestamp without time zone` columns.
type isoNaive time.Time

func (t isoNaive) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", time.Time(t).UTC().Format("2006-01-02T15:04:05.999999")), nil
}
