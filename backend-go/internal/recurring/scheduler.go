package recurring

import (
	"context"
	"log/slog"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/metrics"
)

// Scheduler walks all active recurring templates and mints concrete
// transactions for every occurrence that fell on or before `asOf` and hasn't
// already been generated. ProcessDue is idempotent thanks to the
// last_run_date watermark and per-template skip list.
type Scheduler struct {
	store  *Store
	logger *slog.Logger
}

// NewScheduler wires the store + logger.
func NewScheduler(store *Store, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{store: store, logger: logger}
}

// ProcessDue iterates active templates and inserts any due transactions.
// Returns the number of transactions created.
func (s *Scheduler) ProcessDue(ctx context.Context, asOf time.Time) (int, error) {
	rows, err := s.store.ListActive(ctx)
	if err != nil {
		return 0, err
	}
	created := 0
	for i := range rows {
		r := &rows[i]
		due := DueOccurrences(*r, asOf)
		for _, d := range due {
			_, err := s.store.MintOccurrence(ctx, *r, d)
			if IsAlreadyMinted(err) {
				continue
			}
			if err != nil {
				s.logger.Error("recurring: mint",
					"recurring_id", r.ID, "date", d.Format("2006-01-02"), "err", err)
				continue
			}
			created++
		}
	}
	return created, nil
}

// Run loops fire ProcessDue daily at the configured hour (Europe/Warsaw if
// available, else UTC). Intended to run in its own goroutine.
func (s *Scheduler) Run(ctx context.Context) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		s.logger.Warn("recurring: tz unavailable, using UTC", "err", err)
		loc = time.UTC
	}
	// Process immediately on boot so a fresh start mints any missed runs.
	if n, err := s.ProcessDue(ctx, time.Now().UTC()); err != nil {
		s.logger.Warn("recurring: startup pass failed", "err", err)
		metrics.SchedulerRun("recurring", "error")
	} else {
		metrics.SchedulerRun("recurring", "success")
		if n > 0 {
			s.logger.Info("recurring: startup pass minted transactions", "count", n)
		}
	}
	for {
		now := time.Now().In(loc)
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 4, 0, 0, 0, loc)
		wait := next.Sub(now)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if n, err := s.ProcessDue(ctx, time.Now().UTC()); err != nil {
				s.logger.Warn("recurring: nightly pass failed", "err", err)
				metrics.SchedulerRun("recurring", "error")
			} else {
				metrics.SchedulerRun("recurring", "success")
				if n > 0 {
					s.logger.Info("recurring: nightly pass minted transactions", "count", n)
				}
			}
		}
	}
}
