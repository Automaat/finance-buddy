package retirement

import (
	"testing"
	"time"
)

func mustParsePPK(t *testing.T, value string) time.Time {
	t.Helper()
	got, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return got
}

func TestNextPPKRun_BeforeThirteenth(t *testing.T) {
	from := mustParsePPK(t, "2025-06-10T12:00:00+02:00")
	got := nextPPKRun(from)
	want := time.Date(2025, 6, 13, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextPPKRun_SameDayBeforeFour(t *testing.T) {
	from := mustParsePPK(t, "2025-06-13T03:00:00+02:00")
	got := nextPPKRun(from)
	want := time.Date(2025, 6, 13, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want today 04:00, got %v", got)
	}
}

func TestNextPPKRun_SameDayExactlyFourRollsToNextMonth(t *testing.T) {
	from := mustParsePPK(t, "2025-06-13T04:00:00+02:00")
	got := nextPPKRun(from)
	want := time.Date(2025, 7, 13, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want next month, got %v", got)
	}
}

func TestNextPPKRun_AfterThirteenthGoesToNextMonth(t *testing.T) {
	from := mustParsePPK(t, "2025-06-20T09:30:00+02:00")
	got := nextPPKRun(from)
	want := time.Date(2025, 7, 13, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextPPKRun_DecemberRollsToNextYear(t *testing.T) {
	from := mustParsePPK(t, "2025-12-20T09:30:00+01:00")
	got := nextPPKRun(from)
	want := time.Date(2026, 1, 13, 4, 0, 0, 0, from.Location())
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNextPPKRun_PreservesLocation(t *testing.T) {
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Skip("Europe/Warsaw tz unavailable")
	}
	from := time.Date(2025, 7, 10, 12, 0, 0, 0, warsaw)
	got := nextPPKRun(from)
	if got.Location() != warsaw {
		t.Errorf("location: want Europe/Warsaw, got %v", got.Location())
	}
}
