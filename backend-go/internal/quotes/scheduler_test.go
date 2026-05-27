package quotes

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
)

// fakeStore is an in-memory HoldingsStore for scheduler tests.
type fakeStore struct {
	securities    []holdings.Security
	lastStooq     map[int]time.Time
	firstLot      map[int]time.Time
	manualDates   map[int]map[string]bool // (security, date) → don't overwrite
	written       []holdings.PriceQuote
	latestStooqAt time.Time
	now           time.Time
}

func newFakeStore(secs []holdings.Security) *fakeStore {
	return &fakeStore{
		securities:  secs,
		lastStooq:   map[int]time.Time{},
		firstLot:    map[int]time.Time{},
		manualDates: map[int]map[string]bool{},
	}
}

func (f *fakeStore) ListSecurities(_ context.Context) ([]holdings.Security, error) {
	return f.securities, nil
}

func (f *fakeStore) LastStooqQuoteDate(_ context.Context, id int) (time.Time, error) {
	return f.lastStooq[id], nil
}

func (f *fakeStore) FirstLotDate(_ context.Context, id int) (time.Time, error) {
	return f.firstLot[id], nil
}

func (f *fakeStore) LatestStooqQuoteTime(_ context.Context) (time.Time, error) {
	return f.latestStooqAt, nil
}

func (f *fakeStore) UpsertAutomatedQuote(_ context.Context, q holdings.PriceQuote) (bool, error) {
	if f.manualDates[q.SecurityID][q.Date.Format("2006-01-02")] {
		return false, nil
	}
	f.written = append(f.written, q)
	return true, nil
}

func (f *fakeStore) BulkUpsertAutomatedQuotes(_ context.Context, rows []holdings.PriceQuote) (int, error) {
	n := 0
	for _, q := range rows {
		if f.manualDates[q.SecurityID][q.Date.Format("2006-01-02")] {
			continue
		}
		f.written = append(f.written, q)
		n++
	}
	return n, nil
}

// fakeFetcher tracks what Daily/Latest was called with and replays canned data.
type fakeFetcher struct {
	dailyErr       error
	daily          []Quote
	latest         Quote
	latestErr      error
	dailyCalledFor []time.Time // start dates
	latestCalls    int
}

func (f *fakeFetcher) Latest(_ context.Context, _ string) (Quote, error) {
	f.latestCalls++
	return f.latest, f.latestErr
}

func (f *fakeFetcher) Daily(_ context.Context, _ string, from, _ time.Time) ([]Quote, error) {
	f.dailyCalledFor = append(f.dailyCalledFor, from)
	if f.dailyErr != nil {
		return nil, f.dailyErr
	}
	return f.daily, nil
}

func sec(id int, sym string) holdings.Security {
	return holdings.Security{ID: id, Symbol: sym, Currency: "USD"}
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func mustDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

func TestScheduler_GapFillFromLastStooq(t *testing.T) {
	store := newFakeStore([]holdings.Security{
		sec(1, "ISAC.UK"),
		sec(2, "CSPX.UK"),
	})
	store.lastStooq[1] = mustDay(t, "2026-05-20")
	store.lastStooq[2] = mustDay(t, "2026-05-25")
	store.now = mustDay(t, "2026-05-27")
	fetcher := &fakeFetcher{daily: []Quote{
		{Symbol: "X", Date: mustDay(t, "2026-05-26"), Close: dec("100")},
	}}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if len(fetcher.dailyCalledFor) != 2 {
		t.Fatalf("daily calls = %d, want 2", len(fetcher.dailyCalledFor))
	}
	if got := fetcher.dailyCalledFor[0].Format("2006-01-02"); got != "2026-05-21" {
		t.Errorf("sec 1 daily start = %s, want 2026-05-21", got)
	}
	if got := fetcher.dailyCalledFor[1].Format("2006-01-02"); got != "2026-05-26" {
		t.Errorf("sec 2 daily start = %s, want 2026-05-26", got)
	}
}

func TestScheduler_FirstRunUsesFirstLotMinusPad(t *testing.T) {
	store := newFakeStore([]holdings.Security{sec(1, "ISAC.UK")})
	store.firstLot[1] = mustDay(t, "2025-05-02")
	store.now = mustDay(t, "2026-05-27")
	fetcher := &fakeFetcher{daily: []Quote{
		{Symbol: "ISAC.UK", Date: mustDay(t, "2025-05-02"), Close: dec("89.31")},
	}}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if got := fetcher.dailyCalledFor[0].Format("2006-01-02"); got != "2025-04-25" {
		t.Errorf("daily start = %s, want 2025-04-25 (firstLot - 7d)", got)
	}
}

func TestScheduler_SkipsSecurityWithNoLotsOrHistory(t *testing.T) {
	store := newFakeStore([]holdings.Security{sec(1, "WIPE.UK")})
	store.now = mustDay(t, "2026-05-27")
	fetcher := &fakeFetcher{}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if len(fetcher.dailyCalledFor) != 0 {
		t.Errorf("should not call Daily with no lots, got %d calls", len(fetcher.dailyCalledFor))
	}
	if fetcher.latestCalls != 0 {
		t.Errorf("should not call Latest with no lots, got %d", fetcher.latestCalls)
	}
}

func TestScheduler_FallsBackToLatestWhenNoAPIKey(t *testing.T) {
	store := newFakeStore([]holdings.Security{sec(1, "ISAC.UK")})
	store.lastStooq[1] = mustDay(t, "2026-05-26")
	store.now = mustDay(t, "2026-05-27")
	fetcher := &fakeFetcher{
		dailyErr: ErrNoAPIKey,
		latest:   Quote{Symbol: "ISAC.UK", Date: mustDay(t, "2026-05-27"), Close: dec("121.4")},
	}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if fetcher.latestCalls != 1 {
		t.Errorf("latest calls = %d, want 1 fallback", fetcher.latestCalls)
	}
	if len(store.written) != 1 {
		t.Errorf("written = %d, want 1", len(store.written))
	}
}

func TestScheduler_BulkUpsertPreservesManual(t *testing.T) {
	store := newFakeStore([]holdings.Security{sec(1, "ISAC.UK")})
	store.lastStooq[1] = mustDay(t, "2026-05-19")
	store.now = mustDay(t, "2026-05-22")
	store.manualDates[1] = map[string]bool{"2026-05-21": true}
	fetcher := &fakeFetcher{daily: []Quote{
		{Symbol: "ISAC.UK", Date: mustDay(t, "2026-05-20"), Close: dec("119")},
		{Symbol: "ISAC.UK", Date: mustDay(t, "2026-05-21"), Close: dec("118.68")}, // manual wins
		{Symbol: "ISAC.UK", Date: mustDay(t, "2026-05-22"), Close: dec("120.09")},
	}}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if len(store.written) != 2 {
		t.Errorf("written = %d, want 2 (manual 21st skipped)", len(store.written))
	}
}

func TestScheduler_AlreadyCurrentDoesNothing(t *testing.T) {
	store := newFakeStore([]holdings.Security{sec(1, "ISAC.UK")})
	store.lastStooq[1] = mustDay(t, "2026-05-27")
	store.now = mustDay(t, "2026-05-27")
	fetcher := &fakeFetcher{}
	s := NewScheduler(store, fetcher, slog.Default())
	s.now = func() time.Time { return store.now }
	s.refresh(context.Background())
	if len(fetcher.dailyCalledFor) != 0 || fetcher.latestCalls != 0 {
		t.Errorf("should be no-op when up to date; daily=%d latest=%d",
			len(fetcher.dailyCalledFor), fetcher.latestCalls)
	}
}

// Compile-time assertion that fakeStore satisfies HoldingsStore (catches
// drift when the interface changes).
var (
	_ HoldingsStore = (*fakeStore)(nil)
	_ Fetcher       = (*fakeFetcher)(nil)
)
