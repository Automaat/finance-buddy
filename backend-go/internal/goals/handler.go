package goals

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// validCategories mirrors backend/app/core/enums.Category. Only these strings
// are accepted on the wire; anything else returns a Pydantic-shaped 422.
var validCategories = map[string]struct{}{
	"bank": {}, "saving_account": {}, "stock": {}, "bond": {}, "gold": {},
	"real_estate": {}, "ppk": {}, "fund": {}, "etf": {}, "vehicle": {},
	"mortgage": {}, "installment": {}, "other": {},
}

// response mirrors backend/app/schemas/goals.GoalResponse byte-for-byte.
//
// Money fields are JSON numbers (Pydantic emits them as floats here, not
// strings — different convention from app_config + personas where the column
// type drove the Pydantic Decimal -> string default).
type response struct {
	ID                  int           `json:"id"`
	Name                string        `json:"name"`
	TargetAmount        float64       `json:"target_amount"`
	TargetDate          wire.IsoDate  `json:"target_date"`
	CurrentAmount       float64       `json:"current_amount"`
	MonthlyContribution float64       `json:"monthly_contribution"`
	IsCompleted         bool          `json:"is_completed"`
	AccountID           *int          `json:"account_id"`
	AccountName         *string       `json:"account_name"`
	Category            *string       `json:"category"`
	CreatedAt           wire.IsoNaive `json:"created_at"`
	ProgressPercent     float64       `json:"progress_percent"`
	RemainingAmount     float64       `json:"remaining_amount"`
	ProjectedHitDate    *wire.IsoDate `json:"projected_hit_date"`
}

type listResponse struct {
	Goals          []response `json:"goals"`
	TotalCount     int        `json:"total_count"`
	CompletedCount int        `json:"completed_count"`
}

type createRequest struct {
	Name                string       `json:"name"`
	TargetAmount        float64      `json:"target_amount"`
	TargetDate          wire.IsoDate `json:"target_date"`
	CurrentAmount       float64      `json:"current_amount"`
	MonthlyContribution float64      `json:"monthly_contribution"`
	IsCompleted         bool         `json:"is_completed"`
	AccountID           *int         `json:"account_id"`
	Category            *string      `json:"category"`
}

// Handler is the HTTP boundary for /api/goals.
type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store and logger. The `now` clock is injectable mainly
// for testing; defaults to time.Now.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger, now: time.Now}
}

func (h *Handler) toResponse(g *Goal, accountName string) response {
	target, _ := g.TargetAmount.Float64()
	current, _ := g.CurrentAmount.Float64()
	monthly, _ := g.MonthlyContribution.Float64()

	remaining := target - current
	if remaining < 0 {
		remaining = 0
	}
	progress := 0.0
	if target > 0 {
		progress = math.Min(100.0, current/target*100.0)
	}
	projected := projectHitDate(g.TargetAmount, g.CurrentAmount, g.MonthlyContribution, g.IsCompleted, h.now())

	out := response{
		ID:                  g.ID,
		Name:                g.Name,
		TargetAmount:        target,
		TargetDate:          wire.IsoDate(g.TargetDate),
		CurrentAmount:       current,
		MonthlyContribution: monthly,
		IsCompleted:         g.IsCompleted,
		AccountID:           g.AccountID,
		Category:            g.Category,
		CreatedAt:           wire.IsoNaive(g.CreatedAt),
		ProgressPercent:     progress,
		RemainingAmount:     remaining,
	}
	if accountName != "" {
		name := accountName
		out.AccountName = &name
	}
	if projected != nil {
		d := wire.IsoDate(*projected)
		out.ProjectedHitDate = &d
	}
	return out
}

// List serves GET /api/goals.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, names, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list goals", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Goals: make([]response, 0, len(rows))}
	completed := 0
	for i := range rows {
		g := &rows[i]
		var name string
		if g.AccountID != nil {
			name = names[*g.AccountID]
		}
		out.Goals = append(out.Goals, h.toResponse(g, name))
		if g.IsCompleted {
			completed++
		}
	}
	out.TotalCount = len(rows)
	out.CompletedCount = completed
	writeJSON(w, http.StatusOK, out)
}

// Get serves GET /api/goals/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	g, name, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Goal not found")
			return
		}
		h.logger.Error("get goal", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(g, name))
}

// Create serves POST /api/goals.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if vErr := validateCreate(&req); vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	g := &Goal{
		Name:                req.Name,
		TargetAmount:        decimal.NewFromFloat(req.TargetAmount),
		TargetDate:          time.Time(req.TargetDate),
		CurrentAmount:       decimal.NewFromFloat(req.CurrentAmount),
		MonthlyContribution: decimal.NewFromFloat(req.MonthlyContribution),
		IsCompleted:         req.IsCompleted,
		AccountID:           req.AccountID,
		Category:            req.Category,
	}
	created, name, err := h.store.Create(r.Context(), g)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.toResponse(created, name))
}

// Update serves PUT /api/goals/{id}.
//
// To distinguish "field omitted" from "field set to null" (Pydantic's
// model_fields_set semantics), the body is first decoded to a map of raw
// JSON tokens and each field is read explicitly.
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
	g, name, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(g, name))
}

// Delete serves DELETE /api/goals/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusNotFound, "Goal not found")
			return
		}
		h.logger.Error("delete goal", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	var missing *AccountMissingError
	switch {
	case errors.Is(err, ErrNotFound):
		writeDetailError(w, http.StatusNotFound, "Goal not found")
	case errors.As(err, &missing):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", missing.AccountID))
	default:
		h.logger.Error("goals store", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "goal_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}
