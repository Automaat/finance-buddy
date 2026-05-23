// Package scheduler runs the in-process CPI refresh — the Go replacement for
// the Python APScheduler job (migration/scheduler.md).
//
// One job: pull annual CPI from GUS BDL on the 16th of each month at 04:00
// Europe/Warsaw, plus a startup refresh when the cpi_index table is stale
// (>7 days) or empty. Single-instance only, same as the Python original —
// no leader election.
//
// A hand-rolled timer is used rather than a cron library: the single monthly
// trigger + timezone handling is ~30 lines and adds no dependency.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
)

const (
	cpiStaleAfter = 7 * 24 * time.Hour
	refreshDay    = 16
	refreshHour   = 4
)

// CPIScheduler owns the monthly CPI refresh loop.
type CPIScheduler struct {
	store   *cpi.Store
	fetcher *cpi.GUSFetcher
	logger  *slog.Logger
	now     func() time.Time
	loc     *time.Location
}

// NewCPIScheduler wires the scheduler. Falls back to UTC if the
// Europe/Warsaw zone can't be loaded (missing tzdata).
func NewCPIScheduler(store *cpi.Store, fetcher *cpi.GUSFetcher, logger *slog.Logger) *CPIScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		logger.Warn("scheduler: Europe/Warsaw tz unavailable, using UTC", "err", err)
		loc = time.UTC
	}
	return &CPIScheduler{
		store:   store,
		fetcher: fetcher,
		logger:  logger,
		now:     time.Now,
		loc:     loc,
	}
}

// Run does the startup staleness check, then loops firing the monthly
// refresh until ctx is canceled. Intended to run in its own goroutine.
func (s *CPIScheduler) Run(ctx context.Context) {
	s.startupRefresh(ctx)
	for {
		now := s.now().In(s.loc)
		next := nextRefresh(now)
		wait := max(next.Sub(now), 0)
		s.logger.Info("scheduler: next CPI refresh", "at", next.Format(time.RFC3339))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("scheduler: stopped")
			return
		case <-timer.C:
			s.refresh(ctx)
		}
	}
}

// startupRefresh refreshes immediately when the CPI table is empty or stale,
// matching Python's _startup_refresh_if_stale.
func (s *CPIScheduler) startupRefresh(ctx context.Context) {
	stale, err := s.store.NeedsRefresh(ctx, cpiStaleAfter)
	if err != nil {
		s.logger.Warn("scheduler: staleness check failed", "err", err)
		return
	}
	if stale {
		s.logger.Info("scheduler: CPI stale at startup, refreshing")
		s.refresh(ctx)
	}
}

// refresh runs one GUS fetch + upsert. Failures are logged and swallowed —
// a transient GUS outage must not crash the process.
func (s *CPIScheduler) refresh(ctx context.Context) {
	rows, err := s.fetcher.Fetch(ctx)
	if err != nil {
		s.logger.Warn("scheduler: GUS fetch failed", "err", err)
		return
	}
	written, err := s.store.Upsert(ctx, cpi.GUSSourceTag, rows)
	if err != nil {
		s.logger.Warn("scheduler: CPI upsert failed", "err", err)
		return
	}
	s.logger.Info("scheduler: CPI refresh complete", "rows_written", written)
}

// nextRefresh returns the next 16th-of-month 04:00 strictly after `from`,
// in from's location.
func nextRefresh(from time.Time) time.Time {
	candidate := time.Date(from.Year(), from.Month(), refreshDay, refreshHour, 0, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = time.Date(from.Year(), from.Month()+1, refreshDay, refreshHour, 0, 0, 0, from.Location())
	}
	return candidate
}
