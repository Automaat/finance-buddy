package quotes

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// refreshTimeout caps the synchronous Stooq pass. Picked so a single user
// click can't outlive typical proxy timeouts (Traefik default ~3min) — with
// the Stooq fetcher's 8s per-request timeout this leaves headroom for ~15
// securities even when both Daily and the Latest fallback run.
const refreshTimeout = 2 * time.Minute

// RefreshHandler exposes POST /api/holdings/refresh-quotes — the user-driven
// "fetch now" trigger. Reuses the scheduler's refreshOne so the behavior
// matches the daily job exactly (gap-fill via Daily, fall back to Latest).
type RefreshHandler struct {
	scheduler *Scheduler
	store     HoldingsStore
	logger    *slog.Logger
}

// NewRefreshHandler wires the dependencies. The scheduler is used purely as
// a per-security refresh primitive — its Run loop is not started here.
func NewRefreshHandler(store HoldingsStore, fetcher Fetcher, logger *slog.Logger) *RefreshHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &RefreshHandler{
		scheduler: NewScheduler(store, fetcher, logger),
		store:     store,
		logger:    logger,
	}
}

// RefreshResponse is the body of POST /api/holdings/refresh-quotes.
type RefreshResponse struct {
	Total         int `json:"total"`
	Written       int `json:"written"`
	SkippedManual int `json:"skipped_manual"`
	Failed        int `json:"failed"`
}

// Refresh runs an on-demand Stooq refresh pass and returns per-security
// totals. Same logic as scheduler.refresh, just synchronous + with a
// response payload. The pass is bounded by refreshTimeout so a stuck
// upstream can't pin the request goroutine indefinitely.
func (h *RefreshHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), refreshTimeout)
	defer cancel()
	totals, err := h.run(ctx)
	if err != nil {
		h.logger.Error("refresh-quotes: list securities failed", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Failed to list securities")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, RefreshResponse(totals))
}

func (h *RefreshHandler) run(ctx context.Context) (RefreshTotals, error) {
	secs, err := h.store.ListSecurities(ctx)
	if err != nil {
		return RefreshTotals{}, err
	}
	totals := RefreshTotals{Total: len(secs)}
	for _, sec := range secs {
		h.scheduler.refreshOne(ctx, sec, &totals)
	}
	return totals, nil
}
