// Package equitygrants implements /api/equity-grants — CRUD + vesting math
// + paper-value computation (FMV per share × vested × FX).
//
// Vesting math is a pure-functions port of backend/app/services/vesting.py
// kept in this file so the equity package owns its own math.
package equitygrants

import (
	"time"
)

// Schedule is the input to vesting calculations.
type Schedule struct {
	TotalShares            int
	VestStartDate          time.Time
	VestCliffMonths        int
	VestTotalMonths        int
	VestFrequencyMonths    int // 1 monthly / 3 quarterly / 12 yearly
	CustomSchedule         []CustomScheduleEntry
	RequiresLiquidityEvent bool
	LiquidityEventDate     *time.Time
}

// CustomScheduleEntry is a {month, pct} event — pct is added at the month
// index, summed across all events at or before the elapsed month count.
type CustomScheduleEntry struct {
	Month int     `json:"month"`
	Pct   float64 `json:"pct"`
}

// MonthsBetween returns whole months elapsed from start to end. Uses
// anniversary semantics: if end.day < start.day, the current month hasn't
// completed yet. Matches Python's months_between.
func MonthsBetween(start, end time.Time) int {
	months := (end.Year()-start.Year())*12 + (int(end.Month()) - int(start.Month()))
	if end.Day() < start.Day() {
		months--
	}
	return months
}

// VestedSharesAt computes vested share count at on_date.
func VestedSharesAt(s Schedule, onDate time.Time) int {
	if s.RequiresLiquidityEvent {
		if s.LiquidityEventDate == nil {
			return 0
		}
		if onDate.Before(*s.LiquidityEventDate) {
			return 0
		}
	}
	if onDate.Before(s.VestStartDate) {
		return 0
	}
	elapsed := MonthsBetween(s.VestStartDate, onDate)
	if elapsed < s.VestCliffMonths {
		return 0
	}
	capped := min(elapsed, s.VestTotalMonths)
	if len(s.CustomSchedule) > 0 {
		totalPct := 0.0
		for _, event := range s.CustomSchedule {
			if event.Month <= capped {
				totalPct += event.Pct
			}
		}
		vested := int(float64(s.TotalShares) * totalPct / 100)
		if vested > s.TotalShares {
			return s.TotalShares
		}
		return vested
	}
	if s.VestTotalMonths <= 0 {
		return s.TotalShares
	}
	freq := s.VestFrequencyMonths
	if freq <= 0 {
		freq = 1
	}
	monthsAfterCliff := capped - s.VestCliffMonths
	extraPeriods := monthsAfterCliff / freq
	vestingMonthCount := min(s.VestCliffMonths+extraPeriods*freq, s.VestTotalMonths)
	return int(float64(s.TotalShares) * float64(vestingMonthCount) / float64(s.VestTotalMonths))
}

// VestingProgressPct returns the vested fraction as a percentage (0–100).
func VestingProgressPct(s Schedule, onDate time.Time) float64 {
	if s.TotalShares <= 0 {
		return 0
	}
	vested := VestedSharesAt(s, onDate)
	return float64(vested) / float64(s.TotalShares) * 100
}

// FreqMonthsFromString maps the Python VestingFrequency enum string to the
// month count used in the math.
func FreqMonthsFromString(freq string) int {
	switch freq {
	case "monthly":
		return 1
	case "quarterly":
		return 3
	case "yearly":
		return 12
	default:
		return 1
	}
}
