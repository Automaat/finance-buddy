package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/metrics"
)

// monthlyBackfillStart is how far back the scheduler asks Eurostat for
// HICP YoY values on first run. Covers every retail bond emission you'd
// plausibly hold today (EDO 10y issued ≥ 2014 hits period 11+ today, so
// 2014 is the practical floor; we go a year earlier for headroom).
var monthlyBackfillStart = time.Date(2013, time.January, 1, 0, 0, 0, 0, time.UTC)

// MonthlyCPIStore is the slice of *cpi.Store the monthly scheduler uses.
type MonthlyCPIStore interface {
	NeedsMonthlyRefresh(ctx context.Context, staleAfter time.Duration) (bool, error)
	UpsertMonthly(ctx context.Context, source string, rows []cpi.MonthYearRate) (int, error)
}

// MonthlyCPIFetcher is the slice of *cpi.EurostatHICPFetcher used.
type MonthlyCPIFetcher interface {
	FetchSince(ctx context.Context, since time.Time) ([]cpi.MonthYearRate, error)
}

// MonthlyCPIScheduler runs the monthly-CPI refresh loop. Modeled after
// CPIScheduler — single instance, hand-rolled timer, same 16th-of-month
// cadence (Eurostat usually publishes a new month around day 17, so by
// day 16 we capture the prior month's release).
type MonthlyCPIScheduler struct {
	store   MonthlyCPIStore
	fetcher MonthlyCPIFetcher
	logger  *slog.Logger
	now     func() time.Time
	loc     *time.Location
}

// NewMonthlyCPIScheduler wires the scheduler. Falls back to UTC if the
// Europe/Warsaw zone isn't available (missing tzdata).
func NewMonthlyCPIScheduler(store MonthlyCPIStore, fetcher MonthlyCPIFetcher, logger *slog.Logger) *MonthlyCPIScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		logger.Warn("scheduler: Europe/Warsaw tz unavailable, using UTC", "err", err)
		loc = time.UTC
	}
	return &MonthlyCPIScheduler{
		store:   store,
		fetcher: fetcher,
		logger:  logger,
		now:     time.Now,
		loc:     loc,
	}
}

// Run does the startup staleness check, then loops firing the monthly
// refresh until ctx is canceled.
func (s *MonthlyCPIScheduler) Run(ctx context.Context) {
	s.startupRefresh(ctx)
	for {
		now := s.now().In(s.loc)
		next := nextRefresh(now)
		wait := max(next.Sub(now), 0)
		s.logger.Info("scheduler: next monthly CPI refresh", "at", next.Format(time.RFC3339))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("scheduler: monthly CPI stopped")
			return
		case <-timer.C:
			s.refresh(ctx)
		}
	}
}

func (s *MonthlyCPIScheduler) startupRefresh(ctx context.Context) {
	stale, err := s.store.NeedsMonthlyRefresh(ctx, cpiStaleAfter)
	if err != nil {
		s.logger.Warn("scheduler: monthly CPI staleness check failed", "err", err)
		return
	}
	if stale {
		s.logger.Info("scheduler: monthly CPI stale at startup, refreshing")
		s.refresh(ctx)
	}
}

// refresh fetches Eurostat HICP YoY values from monthlyBackfillStart and
// upserts them. The upsert is change-only so most refreshes write nothing.
func (s *MonthlyCPIScheduler) refresh(ctx context.Context) {
	rows, err := s.fetcher.FetchSince(ctx, monthlyBackfillStart)
	if err != nil {
		s.logger.Warn("scheduler: Eurostat fetch failed", "err", err)
		metrics.SchedulerRun("cpi_monthly", "error")
		return
	}
	written, err := s.store.UpsertMonthly(ctx, cpi.EurostatMonthlySource, rows)
	if err != nil {
		s.logger.Warn("scheduler: monthly CPI upsert failed", "err", err)
		metrics.SchedulerRun("cpi_monthly", "error")
		return
	}
	s.logger.Info("scheduler: monthly CPI refresh complete",
		"rows_written", written, "rows_seen", len(rows))
	metrics.SchedulerRun("cpi_monthly", "success")
}
