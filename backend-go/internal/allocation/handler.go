package allocation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// response is the wire shape for a single allocation_targets row.
type response struct {
	ID          int           `json:"id"`
	Category    string        `json:"category"`
	OwnerUserID *int          `json:"owner_user_id"`
	TargetPct   float64       `json:"target_pct"`
	CreatedAt   wire.IsoNaive `json:"created_at"`
}

type listResponse struct {
	Targets []response `json:"targets"`
}

type createRequest struct {
	Category    string  `json:"category"`
	OwnerUserID *int    `json:"owner_user_id"`
	TargetPct   float64 `json:"target_pct"`
}

type updateRequest struct {
	TargetPct *float64 `json:"target_pct"`
}

type replaceItem struct {
	Category  string  `json:"category"`
	TargetPct float64 `json:"target_pct"`
}

type replaceRequest struct {
	OwnerUserID *int          `json:"owner_user_id"`
	Targets     []replaceItem `json:"targets"`
}

type driftItemWire struct {
	Category          string  `json:"category"`
	OwnerUserID       *int    `json:"owner_user_id"`
	CurrentValue      float64 `json:"current_value"`
	CurrentPercentage float64 `json:"current_percentage"`
	TargetPercentage  float64 `json:"target_percentage"`
	DriftPp           float64 `json:"drift_pp"`
	Severity          string  `json:"severity"`
	RebalanceAmount   float64 `json:"rebalance_amount"`
}

type driftScopeWire struct {
	OwnerUserID       *int            `json:"owner_user_id"`
	TotalValue        float64         `json:"total_value"`
	TargetSumPct      float64         `json:"target_sum_pct"`
	HasCompleteTarget bool            `json:"has_complete_target"`
	Items             []driftItemWire `json:"items"`
}

type driftResponse struct {
	Scopes []driftScopeWire `json:"scopes"`
}

// HoldingsProvider is what the handler needs to render drift — the source
// of latest-snapshot (category, owner, value) buckets. Implemented by the
// real store + injected from server wiring so the handler does not depend
// on the dashboard package's internals.
type HoldingsProvider interface {
	LatestHoldings(ctx context.Context) ([]CategoryAllocation, error)
}

// Handler is the HTTP boundary for /api/allocation.
type Handler struct {
	store    *Store
	holdings HoldingsProvider
	logger   *slog.Logger
}

// NewHandler wires the store, holdings source and logger.
func NewHandler(store *Store, holdings HoldingsProvider, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, holdings: holdings, logger: logger}
}

func toResponse(t *Target) response {
	pct, _ := t.TargetPct.Float64()
	return response{
		ID:          t.ID,
		Category:    t.Category,
		OwnerUserID: t.OwnerUserID,
		TargetPct:   pct,
		CreatedAt:   wire.IsoNaive(t.CreatedAt),
	}
}

// List serves GET /api/allocation/targets.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list allocation targets", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Targets: make([]response, 0, len(rows))}
	for i := range rows {
		out.Targets = append(out.Targets, toResponse(&rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/allocation/targets.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validateCreate(&req); vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	t := &Target{
		Category:    req.Category,
		OwnerUserID: req.OwnerUserID,
		TargetPct:   decimal.NewFromFloat(req.TargetPct),
	}
	created, err := h.store.Create(r.Context(), t)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PUT /api/allocation/targets/{id}. Only target_pct is
// patchable (changing scope is delete + create).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	t, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(t))
}

// Delete serves DELETE /api/allocation/targets/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Replace serves PUT /api/allocation/targets/replace — atomic replace of
// every target for one owner scope. The payload's targets must sum to 100.
func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	var req replaceRequest
	if !httputil.DecodeJSON(w, r, 1<<16, &req) {
		return
	}
	if vErr := validateReplaceBatch(req.Targets); vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	toInsert := make([]Target, 0, len(req.Targets))
	for _, it := range req.Targets {
		toInsert = append(toInsert, Target{
			Category:  it.Category,
			TargetPct: decimal.NewFromFloat(it.TargetPct),
		})
	}
	inserted, err := h.store.Replace(r.Context(), req.OwnerUserID, toInsert)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	out := listResponse{Targets: make([]response, 0, len(inserted))}
	for i := range inserted {
		out.Targets = append(out.Targets, toResponse(&inserted[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Drift serves GET /api/allocation/drift — the dashboard widget payload.
func (h *Handler) Drift(w http.ResponseWriter, r *http.Request) {
	targets, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("drift: list targets", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	holdings, err := h.holdings.LatestHoldings(r.Context())
	if err != nil {
		h.logger.Error("drift: load holdings", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	analysis := ComputeDrift(holdings, targets)
	out := driftResponse{Scopes: make([]driftScopeWire, 0, len(analysis.Scopes))}
	for _, scope := range analysis.Scopes {
		s := driftScopeWire{
			OwnerUserID:       scope.OwnerUserID,
			TotalValue:        scope.TotalValue,
			TargetSumPct:      scope.TargetSumPct,
			HasCompleteTarget: scope.HasCompleteTarget,
			Items:             make([]driftItemWire, 0, len(scope.Items)),
		}
		for _, it := range scope.Items {
			s.Items = append(s.Items, driftItemWire{
				Category:          it.Category,
				OwnerUserID:       it.OwnerUserID,
				CurrentValue:      it.CurrentValue,
				CurrentPercentage: it.CurrentPercentage,
				TargetPercentage:  it.TargetPercentage,
				DriftPp:           it.DriftPP,
				Severity:          it.Severity,
				RebalanceAmount:   it.RebalanceAmount,
			})
		}
		out.Scopes = append(out.Scopes, s)
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	var ownerMissing *OwnerMissingError
	var dupScope *DuplicateScopeError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound, "Allocation target not found")
	case errors.As(err, &ownerMissing):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("User with id %d not found", ownerMissing.OwnerUserID))
	case errors.As(err, &dupScope):
		httputil.WriteDetailError(w, http.StatusConflict, dupScope.Error())
	default:
		h.logger.Error("allocation store", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		httputil.WriteBodyValidationError(w, "target_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// HoldingsFromSnapshots is a default HoldingsProvider — pulls the latest
// snapshot, joins to accounts for category + owner, and returns one
// CategoryAllocation per (owner, category) holding. Liabilities and
// non-asset categories are excluded.
type HoldingsFromSnapshots struct {
	pool *pgxpool.Pool
}

// NewHoldingsFromSnapshots wraps a pool.
func NewHoldingsFromSnapshots(pool *pgxpool.Pool) *HoldingsFromSnapshots {
	return &HoldingsFromSnapshots{pool: pool}
}

// LatestHoldings reads the most recent snapshot's values and rolls them
// up by (owner_user_id, category) for active asset accounts only.
// Soft-deleted accounts are excluded — drift is forward-looking, unlike
// dashboard net worth which keeps historical accounts for trend math.
func (h *HoldingsFromSnapshots) LatestHoldings(ctx context.Context) ([]CategoryAllocation, error) {
	var latestID int
	err := h.pool.QueryRow(ctx, `SELECT id FROM snapshots ORDER BY date DESC, id DESC LIMIT 1`).Scan(&latestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []CategoryAllocation{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest snapshot id: %w", err)
	}
	rows, err := h.pool.Query(ctx, `
		SELECT a.owner_user_id, a.category, sv.value
		FROM snapshot_values sv
		JOIN accounts a ON a.id = sv.account_id
		WHERE sv.snapshot_id = $1 AND a.type = 'asset' AND a.is_active = true
	`, latestID)
	if err != nil {
		return nil, fmt.Errorf("query latest snapshot values: %w", err)
	}
	defer rows.Close()
	out := []CategoryAllocation{}
	for rows.Next() {
		var owner *int
		var category string
		var value decimal.Decimal
		if err := rows.Scan(&owner, &category, &value); err != nil {
			return nil, fmt.Errorf("scan snapshot value: %w", err)
		}
		f, _ := value.Float64()
		out = append(out, CategoryAllocation{OwnerUserID: owner, Category: category, Value: f})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot values: %w", err)
	}
	return out, nil
}
