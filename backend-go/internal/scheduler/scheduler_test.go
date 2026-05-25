package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
)

func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	got, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return got
}

func TestNextRefresh_BeforeSixteenthInMonth(t *testing.T) {
	from := mustParse(t, "2025-06-10T12:00:00+02:00")
	got := nextRefresh(from)
	want := time.Date(2025, 6, 16, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextRefresh_SameDayBeforeFour(t *testing.T) {
	from := mustParse(t, "2025-06-16T03:00:00+02:00")
	got := nextRefresh(from)
	want := time.Date(2025, 6, 16, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want today 04:00, got %v", got)
	}
}

func TestNextRefresh_SameDayExactlyFourSkipsToNextMonth(t *testing.T) {
	// !After means equality counts as "already passed".
	from := mustParse(t, "2025-06-16T04:00:00+02:00")
	got := nextRefresh(from)
	want := time.Date(2025, 7, 16, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want next month, got %v", got)
	}
}

func TestNextRefresh_AfterSixteenthGoesToNextMonth(t *testing.T) {
	from := mustParse(t, "2025-06-20T09:30:00+02:00")
	got := nextRefresh(from)
	want := time.Date(2025, 7, 16, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextRefresh_DecemberRollsToNextYear(t *testing.T) {
	from := mustParse(t, "2025-12-20T09:30:00+01:00")
	got := nextRefresh(from)
	want := time.Date(2026, 1, 16, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextRefresh_PreservesLocation(t *testing.T) {
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Skip("Europe/Warsaw tz unavailable")
	}
	from := time.Date(2025, 7, 10, 12, 0, 0, 0, warsaw)
	got := nextRefresh(from)
	if got.Location() != warsaw {
		t.Errorf("location: want Europe/Warsaw, got %v", got.Location())
	}
}

// fakeStore captures calls + returns canned answers.
type fakeStore struct {
	mu               sync.Mutex
	needsRefreshFn   func(ctx context.Context, after time.Duration) (bool, error)
	upsertFn         func(ctx context.Context, source string, rows []cpi.YearRate) (int, error)
	needsRefreshCnt  int
	upsertCalls      []upsertCall
	upsertReturnRows int
}

type upsertCall struct {
	source string
	rows   []cpi.YearRate
}

func (s *fakeStore) NeedsRefresh(ctx context.Context, after time.Duration) (bool, error) {
	s.mu.Lock()
	s.needsRefreshCnt++
	s.mu.Unlock()
	if s.needsRefreshFn != nil {
		return s.needsRefreshFn(ctx, after)
	}
	return false, nil
}

func (s *fakeStore) Upsert(ctx context.Context, source string, rows []cpi.YearRate) (int, error) {
	s.mu.Lock()
	s.upsertCalls = append(s.upsertCalls, upsertCall{source: source, rows: rows})
	s.mu.Unlock()
	if s.upsertFn != nil {
		return s.upsertFn(ctx, source, rows)
	}
	return s.upsertReturnRows, nil
}

// fakeFetcher captures fetch calls + returns canned answers.
type fakeFetcher struct {
	mu       sync.Mutex
	fetchFn  func(ctx context.Context) ([]cpi.YearRate, error)
	fetchCnt int
}

func (f *fakeFetcher) Fetch(ctx context.Context) ([]cpi.YearRate, error) {
	f.mu.Lock()
	f.fetchCnt++
	f.mu.Unlock()
	if f.fetchFn != nil {
		return f.fetchFn(ctx)
	}
	return nil, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestStartupRefresh_NoOpWhenFresh(t *testing.T) {
	store := &fakeStore{needsRefreshFn: func(context.Context, time.Duration) (bool, error) {
		return false, nil
	}}
	fetcher := &fakeFetcher{}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	s.startupRefresh(t.Context())
	if store.needsRefreshCnt != 1 {
		t.Errorf("NeedsRefresh call count: want 1, got %d", store.needsRefreshCnt)
	}
	if fetcher.fetchCnt != 0 {
		t.Errorf("Fetch must not run when fresh, got %d calls", fetcher.fetchCnt)
	}
	if len(store.upsertCalls) != 0 {
		t.Errorf("Upsert must not run when fresh, got %d calls", len(store.upsertCalls))
	}
}

func TestStartupRefresh_RunsWhenStale(t *testing.T) {
	store := &fakeStore{needsRefreshFn: func(context.Context, time.Duration) (bool, error) {
		return true, nil
	}}
	fetcher := &fakeFetcher{fetchFn: func(context.Context) ([]cpi.YearRate, error) {
		return []cpi.YearRate{{Year: 2024}}, nil
	}}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	s.startupRefresh(t.Context())
	if fetcher.fetchCnt != 1 {
		t.Errorf("Fetch: want 1 call, got %d", fetcher.fetchCnt)
	}
	if len(store.upsertCalls) != 1 {
		t.Fatalf("Upsert: want 1 call, got %d", len(store.upsertCalls))
	}
	if store.upsertCalls[0].source != cpi.GUSSourceTag {
		t.Errorf("Upsert source: want %q, got %q", cpi.GUSSourceTag, store.upsertCalls[0].source)
	}
}

func TestStartupRefresh_StalenessCheckError_Swallowed(t *testing.T) {
	store := &fakeStore{needsRefreshFn: func(context.Context, time.Duration) (bool, error) {
		return false, errors.New("db down")
	}}
	fetcher := &fakeFetcher{}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	// Must not panic and must not run refresh on staleness error.
	s.startupRefresh(t.Context())
	if fetcher.fetchCnt != 0 || len(store.upsertCalls) != 0 {
		t.Errorf("refresh should be skipped on staleness error")
	}
}

func TestRefresh_GUSFetchErrorSkipsUpsert(t *testing.T) {
	store := &fakeStore{}
	fetcher := &fakeFetcher{fetchFn: func(context.Context) ([]cpi.YearRate, error) {
		return nil, errors.New("connection refused")
	}}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	s.refresh(t.Context())
	if fetcher.fetchCnt != 1 {
		t.Errorf("Fetch: want 1, got %d", fetcher.fetchCnt)
	}
	if len(store.upsertCalls) != 0 {
		t.Errorf("Upsert must not run after fetch error, got %d calls", len(store.upsertCalls))
	}
}

func TestRefresh_UpsertErrorSwallowed(t *testing.T) {
	store := &fakeStore{upsertFn: func(context.Context, string, []cpi.YearRate) (int, error) {
		return 0, errors.New("constraint violation")
	}}
	fetcher := &fakeFetcher{fetchFn: func(context.Context) ([]cpi.YearRate, error) {
		return []cpi.YearRate{{Year: 2024}}, nil
	}}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	// Must not panic — scheduler logs and returns.
	s.refresh(t.Context())
	if len(store.upsertCalls) != 1 {
		t.Errorf("Upsert: want 1 call, got %d", len(store.upsertCalls))
	}
}

func TestRefresh_PassesFetchedRowsToUpsert(t *testing.T) {
	want := []cpi.YearRate{{Year: 2023}, {Year: 2024}}
	store := &fakeStore{upsertReturnRows: 2}
	fetcher := &fakeFetcher{fetchFn: func(context.Context) ([]cpi.YearRate, error) {
		return want, nil
	}}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	s.refresh(t.Context())
	if len(store.upsertCalls) != 1 {
		t.Fatalf("Upsert calls: want 1, got %d", len(store.upsertCalls))
	}
	got := store.upsertCalls[0].rows
	if len(got) != len(want) || got[0].Year != 2023 || got[1].Year != 2024 {
		t.Errorf("rows passed to Upsert: want %+v, got %+v", want, got)
	}
}

func TestRun_StopsOnContextCancel(t *testing.T) {
	// Pin "now" so the next refresh is far in the future, guaranteeing the
	// loop is blocked on the timer when ctx is canceled.
	pinned := time.Date(2025, 6, 17, 5, 0, 0, 0, time.UTC) // just past a refresh
	store := &fakeStore{}
	fetcher := &fakeFetcher{}
	s := NewCPIScheduler(store, fetcher, discardLogger())
	s.now = func() time.Time { return pinned }

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop within 2s of ctx cancel")
	}
}
