package goals

import (
	"time"

	"github.com/shopspring/decimal"
)

// projectHitDate mirrors backend/app/services/goals._project_hit_date.
//
// Returns today (UTC) if the goal is already completed or current >= target.
// Returns nil if monthly_contribution is non-positive — projection isn't
// possible. Otherwise ceil((target - current) / monthly) months forward from
// today.
func projectHitDate(target, current, monthly decimal.Decimal, isCompleted bool, now time.Time) *time.Time {
	today := now.UTC().Truncate(24 * time.Hour)
	if isCompleted || current.GreaterThanOrEqual(target) {
		return &today
	}
	if monthly.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	remaining := target.Sub(current)
	months := remaining.Div(monthly)
	monthsCeil := months.Ceil().IntPart()
	hit := addMonths(today, int(monthsCeil))
	return &hit
}

// addMonths is the Go port of backend/app/services/goals._add_months — adds
// calendar months and clamps the day to the last valid day of the target
// month (so e.g. Jan 31 + 1 month = Feb 28/29, not Mar 3).
func addMonths(d time.Time, months int) time.Time {
	monthIndex := int(d.Month()) - 1 + months
	year := d.Year() + monthIndex/12
	monthOffset := monthIndex % 12
	if monthOffset < 0 {
		monthOffset += 12
		year--
	}
	month := time.Month(monthOffset + 1)
	day := d.Day()
	lastDay := daysIn(year, month)
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// daysIn returns the last day-of-month for the given (year, month) — mirrors
// Python's calendar.monthrange()[1].
func daysIn(year int, month time.Month) int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 1, -1).Day()
}
