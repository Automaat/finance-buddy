package quotes

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
	"github.com/Automaat/finance-buddy/backend-go/internal/metrics"
)

const (
	// QuotesSource tags rows written by the Stooq scheduler. Frontend uses
	// it to distinguish auto-pulled vs. manual quotes.
	QuotesSource = "stooq"

	refreshHour   = 18 // 18:00 Europe/Warsaw — after EU market close
	refreshMinute = 5
	startupStale  = 18 * time.Hour
)

// Fetcher fetches Stooq quotes. Latest is keyless; Daily requires apikey.
// Implemented by StooqFetcher; an interface so the scheduler can be tested
// without HTTP.
type Fetcher interface {
	Latest(ctx context.Context, symbol string) (Quote, error)
	Daily(ctx context.Context, symbol string, from, to time.Time) ([]Quote, error)
}

// HoldingsStore is the slice of holdings.Store the scheduler needs.
type HoldingsStore interface {
	ListSecurities(ctx context.Context) ([]holdings.Security, error)
	LastStooqQuoteDate(ctx context.Context, securityID int) (time.Time, error)
	FirstLotDate(ctx context.Context, securityID int) (time.Time, error)
	UpsertAutomatedQuote(ctx context.Context, q holdings.PriceQuote) (bool, error)
	BulkUpsertAutomatedQuotes(ctx context.Context, rows []holdings.PriceQuote) (int, error)
	LatestStooqQuoteTime(ctx context.Context) (time.Time, error)
}

// backfillPadDays is how far before the first lot date we start a brand-new
// security's history pull — a small buffer so the chart has a visible
// pre-purchase baseline.
const backfillPadDays = 7

// Scheduler refreshes prices from a Fetcher once a day. Modeled after
// scheduler.CPIScheduler — single-instance, hand-rolled timer, no cron lib.
type Scheduler struct {
	store   HoldingsStore
	fetcher Fetcher
	logger  *slog.Logger
	now     func() time.Time
	loc     *time.Location
}

// NewScheduler wires the scheduler. Falls back to UTC if Europe/Warsaw tzdata
// is missing (matches CPI scheduler).
func NewScheduler(store HoldingsStore, fetcher Fetcher, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		logger.Warn("quotes: Europe/Warsaw tz unavailable, using UTC", "err", err)
		loc = time.UTC
	}
	return &Scheduler{
		store:   store,
		fetcher: fetcher,
		logger:  logger,
		now:     time.Now,
		loc:     loc,
	}
}

// Run does a stale-check refresh, then loops daily until ctx is canceled.
// Intended to run in its own goroutine.
func (s *Scheduler) Run(ctx context.Context) {
	s.startupRefresh(ctx)
	for {
		now := s.now().In(s.loc)
		next := nextRefresh(now)
		wait := max(next.Sub(now), 0)
		s.logger.Info("quotes: next refresh", "at", next.Format(time.RFC3339))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("quotes: stopped")
			return
		case <-timer.C:
			s.refresh(ctx)
		}
	}
}

func (s *Scheduler) startupRefresh(ctx context.Context) {
	last, err := s.store.LatestStooqQuoteTime(ctx)
	if err != nil {
		s.logger.Warn("quotes: staleness check failed", "err", err)
		return
	}
	// Both sides are nominally UTC: price_quotes.created_at is `timestamp
	// without time zone` defaulted to `now() at time zone 'utc'`, and pgx
	// scans naive timestamps with time.UTC. s.now().UTC() makes the same
	// assumption explicit on the application side. If price_quotes ever
	// migrates to timestamptz, both branches still subtract correctly.
	if last.IsZero() || s.now().UTC().Sub(last) > startupStale {
		s.logger.Info("quotes: stale at startup, refreshing", "last", last)
		s.refresh(ctx)
	}
}

// refresh runs a full pass: gap-fill each security from its last stored
// stooq date to today via Daily; fall back to Latest when the daily endpoint
// is unavailable (no apikey, or it errored). One bad security must not stop
// the rest — every error is logged and skipped.
func (s *Scheduler) refresh(ctx context.Context) {
	secs, err := s.store.ListSecurities(ctx)
	if err != nil {
		s.logger.Warn("quotes: list securities failed", "err", err)
		metrics.SchedulerRun("quotes", "error")
		return
	}
	totals := RefreshTotals{Total: len(secs)}
	for _, sec := range secs {
		s.refreshOne(ctx, sec, &totals)
	}
	s.logger.Info("quotes: refresh complete",
		"written", totals.Written, "skipped_manual", totals.SkippedManual,
		"failed", totals.Failed, "total", totals.Total)
	// A pass that listed securities is a successful run even if individual
	// per-security fetches failed (those are tracked in totals.Failed).
	metrics.SchedulerRun("quotes", "success")
}

// RefreshTotals is the per-pass counter shared between scheduler runs and
// the on-demand HTTP handler.
type RefreshTotals struct {
	Total         int
	Written       int
	SkippedManual int
	Failed        int
}

// refreshOne handles a single security: pick the right date window, fetch,
// upsert, and bump the counters in place.
func (s *Scheduler) refreshOne(ctx context.Context, sec holdings.Security, totals *RefreshTotals) {
	from, ok, err := s.backfillStart(ctx, sec.ID)
	if err != nil {
		s.logger.Warn("quotes: backfill-start lookup failed", "symbol", sec.Symbol, "err", err)
		totals.Failed++
		return
	}
	if !ok {
		// No lots and no prior history — nothing to value yet, skip.
		return
	}
	today := s.now().UTC().Truncate(24 * time.Hour)
	if from.After(today) {
		return // already current
	}
	rows, err := s.fetcher.Daily(ctx, sec.Symbol, from, today)
	if err == nil {
		s.persistDaily(ctx, sec, rows, totals)
		return
	}
	if !errors.Is(err, ErrNoAPIKey) {
		s.logger.Warn("quotes: daily fetch failed, falling back to latest",
			"symbol", sec.Symbol, "err", err)
	}
	s.persistLatest(ctx, sec, totals)
}

// backfillStart returns the first date we should request from Stooq for sec.
// (zero, false, nil) means "nothing to do" (no lots and no history).
func (s *Scheduler) backfillStart(ctx context.Context, securityID int) (time.Time, bool, error) {
	last, err := s.store.LastStooqQuoteDate(ctx, securityID)
	if err != nil {
		return time.Time{}, false, err
	}
	if !last.IsZero() {
		return last.AddDate(0, 0, 1), true, nil
	}
	first, err := s.store.FirstLotDate(ctx, securityID)
	if err != nil {
		return time.Time{}, false, err
	}
	if first.IsZero() {
		return time.Time{}, false, nil
	}
	return first.AddDate(0, 0, -backfillPadDays), true, nil
}

func (s *Scheduler) persistDaily(ctx context.Context, sec holdings.Security, rows []Quote, totals *RefreshTotals) {
	if len(rows) == 0 {
		return
	}
	batch := make([]holdings.PriceQuote, len(rows))
	for i, q := range rows {
		batch[i] = holdings.PriceQuote{
			SecurityID: sec.ID,
			Date:       q.Date,
			Price:      q.Close,
			Source:     QuotesSource,
		}
	}
	written, err := s.store.BulkUpsertAutomatedQuotes(ctx, batch)
	if err != nil {
		s.logger.Warn("quotes: bulk upsert failed", "symbol", sec.Symbol, "err", err)
		totals.Failed++
		return
	}
	totals.Written += written
	totals.SkippedManual += len(batch) - written
}

func (s *Scheduler) persistLatest(ctx context.Context, sec holdings.Security, totals *RefreshTotals) {
	q, err := s.fetcher.Latest(ctx, sec.Symbol)
	if err != nil {
		if errors.Is(err, ErrNoData) {
			s.logger.Info("quotes: no data", "symbol", sec.Symbol)
		} else {
			s.logger.Warn("quotes: latest fetch failed", "symbol", sec.Symbol, "err", err)
		}
		totals.Failed++
		return
	}
	ok, err := s.store.UpsertAutomatedQuote(ctx, holdings.PriceQuote{
		SecurityID: sec.ID,
		Date:       q.Date,
		Price:      q.Close,
		Source:     QuotesSource,
	})
	if err != nil {
		s.logger.Warn("quotes: upsert failed", "symbol", sec.Symbol, "err", err)
		totals.Failed++
		return
	}
	if ok {
		totals.Written++
	} else {
		totals.SkippedManual++
	}
}

// nextRefresh returns the next 18:05 strictly after `from`, in from's location.
func nextRefresh(from time.Time) time.Time {
	cand := time.Date(from.Year(), from.Month(), from.Day(), refreshHour, refreshMinute, 0, 0, from.Location())
	if !cand.After(from) {
		cand = cand.AddDate(0, 0, 1)
	}
	return cand
}
