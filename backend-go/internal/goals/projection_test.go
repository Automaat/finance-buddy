package goals

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var fixedNow = time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)

func TestProjectHitDateCompleted(t *testing.T) {
	hit := projectHitDate(
		decimal.NewFromInt(100),
		decimal.NewFromInt(0),
		decimal.NewFromInt(10),
		true,
		fixedNow,
	)
	if hit == nil || !hit.Equal(fixedNow.UTC().Truncate(24*time.Hour)) {
		t.Fatalf("completed goal should hit today, got %+v", hit)
	}
}

func TestProjectHitDateAlreadyReached(t *testing.T) {
	hit := projectHitDate(
		decimal.NewFromInt(100),
		decimal.NewFromInt(150),
		decimal.NewFromInt(10),
		false,
		fixedNow,
	)
	if hit == nil || !hit.Equal(fixedNow.UTC().Truncate(24*time.Hour)) {
		t.Fatalf("reached goal should hit today, got %+v", hit)
	}
}

func TestProjectHitDateNoContribution(t *testing.T) {
	hit := projectHitDate(
		decimal.NewFromInt(100),
		decimal.NewFromInt(0),
		decimal.Zero,
		false,
		fixedNow,
	)
	if hit != nil {
		t.Fatalf("zero contribution should be nil, got %+v", hit)
	}
}

func TestProjectHitDateBasic(t *testing.T) {
	// (100 - 0) / 10 = 10 months -> Nov 30 from Jan 31 (clamped to last day)
	hit := projectHitDate(
		decimal.NewFromInt(100),
		decimal.NewFromInt(0),
		decimal.NewFromInt(10),
		false,
		fixedNow,
	)
	if hit == nil {
		t.Fatal("expected non-nil hit date")
	}
	want := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	if !hit.Equal(want) {
		t.Fatalf("expected %s, got %s", want, hit)
	}
}

func TestProjectHitDateCeilingPartialMonth(t *testing.T) {
	// (100 - 0) / 30 = 3.33 -> ceil 4
	hit := projectHitDate(
		decimal.NewFromInt(100),
		decimal.NewFromInt(0),
		decimal.NewFromInt(30),
		false,
		fixedNow,
	)
	if hit == nil {
		t.Fatal("expected non-nil hit date")
	}
	want := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if !hit.Equal(want) {
		t.Fatalf("expected %s, got %s", want, hit)
	}
}

func TestAddMonthsDayClamp(t *testing.T) {
	jan31 := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	feb := addMonths(jan31, 1)
	want := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if !feb.Equal(want) {
		t.Fatalf("Jan31+1 should clamp to Feb 28, got %s", feb)
	}
}

func TestAddMonthsLeapYear(t *testing.T) {
	jan31 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	feb := addMonths(jan31, 1)
	want := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	if !feb.Equal(want) {
		t.Fatalf("leap Feb should be 29, got %s", feb)
	}
}

func TestAddMonthsForwardYearRoll(t *testing.T) {
	dec := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	if got := addMonths(dec, 1); !got.Equal(time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Dec+1 should roll to Jan, got %s", got)
	}
}

func TestAddMonthsBackwardYearRoll(t *testing.T) {
	jan := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if got := addMonths(jan, -1); !got.Equal(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Jan-1 should roll back, got %s", got)
	}
}

func TestDaysInMonths(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  int
	}{
		{2026, time.January, 31},
		{2026, time.February, 28},
		{2024, time.February, 29},
		{2026, time.April, 30},
		{2026, time.December, 31},
	}
	for _, tc := range cases {
		if got := daysIn(tc.year, tc.month); got != tc.want {
			t.Errorf("daysIn(%d, %s) = %d, want %d", tc.year, tc.month, got, tc.want)
		}
	}
}
