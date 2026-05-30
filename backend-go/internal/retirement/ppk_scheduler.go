package retirement

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/metrics"
)

const (
	ppkRunDay  = 13
	ppkRunHour = 4
)

// PPKScheduler auto-generates the current month's PPK contributions for every
// eligible owner on the 13th of each month (04:00 Europe/Warsaw). Eligibility
// = a UOP salary on record + configured PPK rates + an active PPK account;
// owners failing any check are skipped. Generation is idempotent via the
// store's per-month dedup, so a restart after the 13th re-runs harmlessly.
// Single instance, mirrors the CPI/recurring schedulers.
type PPKScheduler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
	loc    *time.Location
}

// NewPPKScheduler wires the scheduler. Falls back to UTC if the Europe/Warsaw
// zone isn't available (missing tzdata).
func NewPPKScheduler(store *Store, logger *slog.Logger) *PPKScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		logger.Warn("ppk scheduler: Europe/Warsaw tz unavailable, using UTC", "err", err)
		loc = time.UTC
	}
	return &PPKScheduler{store: store, logger: logger, now: time.Now, loc: loc}
}

// nextPPKRun returns the next 13th-of-month at ppkRunHour in from's location.
func nextPPKRun(from time.Time) time.Time {
	candidate := time.Date(from.Year(), from.Month(), ppkRunDay, ppkRunHour, 0, 0, 0, from.Location())
	if candidate.After(from) {
		return candidate
	}
	return time.Date(from.Year(), from.Month()+1, ppkRunDay, ppkRunHour, 0, 0, 0, from.Location())
}

// Run does a startup catch-up (when this month's 13th has already passed) then
// loops firing on each 13th until ctx is canceled.
func (s *PPKScheduler) Run(ctx context.Context) {
	now := s.now().In(s.loc)
	scheduled := time.Date(now.Year(), now.Month(), ppkRunDay, ppkRunHour, 0, 0, 0, s.loc)
	if !now.Before(scheduled) {
		s.logger.Info("ppk scheduler: catch-up pass at startup")
		s.generateMonth(ctx, int(now.Month()), now.Year())
	}
	for {
		now := s.now().In(s.loc)
		next := nextPPKRun(now)
		s.logger.Info("ppk scheduler: next run", "at", next.Format(time.RFC3339))
		timer := time.NewTimer(max(next.Sub(now), 0))
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("ppk scheduler: stopped")
			return
		case <-timer.C:
			run := s.now().In(s.loc)
			s.generateMonth(ctx, int(run.Month()), run.Year())
		}
	}
}

// generateMonth mints contributions for the given month/year for every named
// owner, skipping the ineligible. Idempotent across reruns.
func (s *PPKScheduler) generateMonth(ctx context.Context, month, year int) {
	ids, err := s.store.UserIDs(ctx)
	if err != nil {
		s.logger.Warn("ppk scheduler: list users failed", "err", err)
		metrics.SchedulerRun("ppk", "error")
		return
	}
	created := 0
	for i := range ids {
		owner := ids[i]
		_, err := GeneratePPK(ctx, s.store, &owner, month, year, GenerateOptions{})
		switch {
		case err == nil:
			created++
			s.logger.Info("ppk scheduler: generated",
				"owner", owner, "month", month, "year", year)
		case errors.Is(err, ErrContributionsExist):
			// Already minted this month — idempotent skip.
		case errors.Is(err, ErrNotUOP), errors.Is(err, ErrNoSalary),
			errors.Is(err, ErrNoPPKAccount), errors.Is(err, ErrUserNotFound):
			s.logger.Debug("ppk scheduler: owner not eligible", "owner", owner, "reason", err)
		default:
			s.logger.Error("ppk scheduler: generate failed", "owner", owner, "err", err)
		}
	}
	s.logger.Info("ppk scheduler: month pass complete",
		"month", month, "year", year, "created", created)
	metrics.SchedulerRun("ppk", "success")
}
