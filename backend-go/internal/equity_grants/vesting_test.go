package equitygrants

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestMonthsBetween(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		want       int
	}{
		{"same day", "2025-01-15", "2025-01-15", 0},
		{"one full month", "2025-01-15", "2025-02-15", 1},
		{"one day short of full month", "2025-01-15", "2025-02-14", 0},
		{"twelve months", "2024-01-15", "2025-01-15", 12},
		{"thirteen months minus a day", "2024-01-15", "2025-02-14", 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MonthsBetween(date(tt.start), date(tt.end))
			if got != tt.want {
				t.Fatalf("MonthsBetween(%s, %s) = %d, want %d", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestVestedSharesAt_standardFourYearOneYearCliff(t *testing.T) {
	s := Schedule{
		TotalShares:         4800,
		VestStartDate:       date("2025-01-01"),
		VestCliffMonths:     12,
		VestTotalMonths:     48,
		VestFrequencyMonths: 1,
	}

	if got := VestedSharesAt(s, date("2025-06-01")); got != 0 {
		t.Errorf("pre-cliff vested = %d, want 0", got)
	}
	// On cliff: 12 months elapsed → exactly 1 year vested (1200 shares).
	if got := VestedSharesAt(s, date("2026-01-01")); got != 1200 {
		t.Errorf("at-cliff vested = %d, want 1200", got)
	}
	// At full vest.
	if got := VestedSharesAt(s, date("2029-01-01")); got != 4800 {
		t.Errorf("full-vest vested = %d, want 4800", got)
	}
	// After full vest, still capped.
	if got := VestedSharesAt(s, date("2030-01-01")); got != 4800 {
		t.Errorf("post-vest vested = %d, want 4800 (capped)", got)
	}
}

func TestVestedSharesAt_beforeStartReturnsZero(t *testing.T) {
	s := Schedule{
		TotalShares:         1000,
		VestStartDate:       date("2025-01-01"),
		VestCliffMonths:     0,
		VestTotalMonths:     12,
		VestFrequencyMonths: 1,
	}
	if got := VestedSharesAt(s, date("2024-12-31")); got != 0 {
		t.Errorf("pre-start vested = %d, want 0", got)
	}
}

func TestVestedSharesAt_liquidityEventBlocksVesting(t *testing.T) {
	liq := date("2027-06-01")
	s := Schedule{
		TotalShares:            1000,
		VestStartDate:          date("2025-01-01"),
		VestCliffMonths:        12,
		VestTotalMonths:        48,
		VestFrequencyMonths:    1,
		RequiresLiquidityEvent: true,
		LiquidityEventDate:     &liq,
	}
	// Past the cliff, but liquidity event still in the future → 0.
	if got := VestedSharesAt(s, date("2026-06-01")); got != 0 {
		t.Errorf("pre-liquidity vested = %d, want 0", got)
	}
	// Past liquidity event → normal time-based math kicks in.
	if got := VestedSharesAt(s, date("2027-06-01")); got == 0 {
		t.Errorf("post-liquidity vested = 0, want >0")
	}
}

func TestVestedSharesAt_liquidityEventMissingDateReturnsZero(t *testing.T) {
	s := Schedule{
		TotalShares:            1000,
		VestStartDate:          date("2025-01-01"),
		VestCliffMonths:        0,
		VestTotalMonths:        12,
		VestFrequencyMonths:    1,
		RequiresLiquidityEvent: true,
		LiquidityEventDate:     nil,
	}
	if got := VestedSharesAt(s, date("2026-01-01")); got != 0 {
		t.Errorf("vested without liquidity date = %d, want 0", got)
	}
}

func TestVestedSharesAt_customScheduleSumsAtOrBeforeElapsed(t *testing.T) {
	s := Schedule{
		TotalShares:   2000,
		VestStartDate: date("2025-01-01"),
		CustomSchedule: []CustomScheduleEntry{
			{Month: 12, Pct: 25},
			{Month: 24, Pct: 25},
			{Month: 36, Pct: 25},
			{Month: 48, Pct: 25},
		},
		VestTotalMonths: 48,
	}
	// At 6 months: nothing.
	if got := VestedSharesAt(s, date("2025-07-01")); got != 0 {
		t.Errorf("6-month custom vested = %d, want 0", got)
	}
	// At 12 months exactly: 25% → 500.
	if got := VestedSharesAt(s, date("2026-01-01")); got != 500 {
		t.Errorf("12-month custom vested = %d, want 500", got)
	}
	// At 48 months: 100% → 2000.
	if got := VestedSharesAt(s, date("2029-01-01")); got != 2000 {
		t.Errorf("48-month custom vested = %d, want 2000", got)
	}
}

func TestVestedSharesAt_customScheduleCapsAtTotal(t *testing.T) {
	// A schedule that over-promises (sums to 200%) should still cap at total.
	s := Schedule{
		TotalShares:   1000,
		VestStartDate: date("2025-01-01"),
		CustomSchedule: []CustomScheduleEntry{
			{Month: 12, Pct: 200},
		},
		VestTotalMonths: 48,
	}
	if got := VestedSharesAt(s, date("2026-01-01")); got != 1000 {
		t.Errorf("over-vest custom = %d, want 1000 (capped)", got)
	}
}

func TestVestedSharesAt_zeroTotalMonthsReturnsAll(t *testing.T) {
	s := Schedule{
		TotalShares:         1000,
		VestStartDate:       date("2025-01-01"),
		VestCliffMonths:     0,
		VestTotalMonths:     0,
		VestFrequencyMonths: 1,
	}
	if got := VestedSharesAt(s, date("2025-01-01")); got != 1000 {
		t.Errorf("zero-total-months vested = %d, want 1000", got)
	}
}

func TestVestingProgressPct(t *testing.T) {
	s := Schedule{
		TotalShares:         1000,
		VestStartDate:       date("2025-01-01"),
		VestCliffMonths:     12,
		VestTotalMonths:     48,
		VestFrequencyMonths: 1,
	}
	// At cliff: 25%.
	if got := VestingProgressPct(s, date("2026-01-01")); got != 25 {
		t.Errorf("cliff progress = %v, want 25", got)
	}
	// Pre-cliff: 0%.
	if got := VestingProgressPct(s, date("2025-06-01")); got != 0 {
		t.Errorf("pre-cliff progress = %v, want 0", got)
	}
}

func TestVestingProgressPct_zeroTotalSharesReturnsZero(t *testing.T) {
	s := Schedule{TotalShares: 0, VestStartDate: date("2025-01-01")}
	if got := VestingProgressPct(s, date("2026-01-01")); got != 0 {
		t.Errorf("zero-shares progress = %v, want 0", got)
	}
}

func TestFreqMonthsFromString(t *testing.T) {
	cases := map[string]int{
		"monthly":   1,
		"quarterly": 3,
		"yearly":    12,
		"":          1,
		"weekly":    1, // default fallback
	}
	for input, want := range cases {
		if got := FreqMonthsFromString(input); got != want {
			t.Errorf("FreqMonthsFromString(%q) = %d, want %d", input, got, want)
		}
	}
}
