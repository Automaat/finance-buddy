package fx

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestToPLNNilAmountReturnsFalse(t *testing.T) {
	_, ok := ToPLN(nil, "USD", Result{Rate: decimal.NewFromInt(4), Found: true})
	if ok {
		t.Fatal("nil amount should return ok=false")
	}
}

func TestToPLNPLNPassesThrough(t *testing.T) {
	amt := decimal.NewFromInt(100)
	got, ok := ToPLN(&amt, "PLN", Result{})
	if !ok {
		t.Fatal("PLN should always be ok")
	}
	if !got.Equal(amt) {
		t.Fatalf("PLN should pass through, got %s", got)
	}
}

func TestToPLNPlnLowercase(t *testing.T) {
	amt := decimal.NewFromInt(100)
	got, ok := ToPLN(&amt, "pln", Result{})
	if !ok || !got.Equal(amt) {
		t.Fatalf("lowercase pln should also pass through, got %s ok=%v", got, ok)
	}
}

func TestToPLNNoRateReturnsFalse(t *testing.T) {
	amt := decimal.NewFromInt(100)
	if _, ok := ToPLN(&amt, "USD", Result{Found: false}); ok {
		t.Fatal("expected ok=false when rate is not found")
	}
}

func TestToPLNMultipliesByRate(t *testing.T) {
	amt := decimal.NewFromInt(100)
	got, ok := ToPLN(&amt, "USD", Result{Rate: decimal.RequireFromString("4.05"), Found: true})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !got.Equal(decimal.RequireFromString("405")) {
		t.Fatalf("expected 405, got %s", got)
	}
}

func TestTruncateDayStripsTime(t *testing.T) {
	in := time.Date(2026, 5, 23, 13, 45, 30, 999, time.UTC)
	got := truncateDay(in)
	want := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestTruncateDayForcesUTC(t *testing.T) {
	// Use a fixed zone so the test doesn't depend on the tzdata being
	// installed on the runner.
	warsaw := time.FixedZone("Europe/Warsaw", 2*60*60)
	in := time.Date(2026, 5, 23, 23, 30, 0, 0, warsaw)
	got := truncateDay(in)
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC, got %s", got.Location())
	}
}
