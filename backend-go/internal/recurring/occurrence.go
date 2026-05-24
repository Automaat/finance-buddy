package recurring

import "time"

// NextOccurrence returns the next date at or after `from` (inclusive) on which
// the recurring template should fire, taking into account StartDate, EndDate,
// SkippedDates, and the cadence. Returns (zero, false) when the template is
// terminated (past EndDate, inactive, or no further occurrence in 10 years).
func NextOccurrence(r Recurring, from time.Time) (time.Time, bool) {
	if !r.Active {
		return time.Time{}, false
	}
	skipped := skipSet(r.SkippedDates)
	candidate := r.StartDate
	if from.After(candidate) {
		candidate = from
	}
	candidate = candidate.UTC().Truncate(24 * time.Hour)
	limit := candidate.AddDate(10, 0, 0)

	candidate = alignToCadence(r, candidate)

	for candidate.Before(limit) {
		if r.EndDate != nil && candidate.After(*r.EndDate) {
			return time.Time{}, false
		}
		if !skipped[candidate.UTC().Format("2006-01-02")] {
			return candidate, true
		}
		candidate = advance(r, candidate)
	}
	return time.Time{}, false
}

// DueOccurrences enumerates dates at which a recurring template should fire
// strictly after LastRunDate (or its StartDate when never run) and on or
// before `asOf`. Skipped + post-end dates are filtered out. Used by the
// scheduler to mint concrete transactions on demand.
func DueOccurrences(r Recurring, asOf time.Time) []time.Time {
	if !r.Active {
		return nil
	}
	asOf = asOf.UTC().Truncate(24 * time.Hour)
	cursor := r.StartDate
	if r.LastRunDate != nil {
		cursor = advance(r, *r.LastRunDate)
	}
	cursor = alignToCadence(r, cursor)
	skipped := skipSet(r.SkippedDates)
	out := []time.Time{}
	limit := asOf.AddDate(1, 0, 0) // safety net
	for !cursor.After(asOf) && cursor.Before(limit) {
		if r.EndDate != nil && cursor.After(*r.EndDate) {
			break
		}
		if !skipped[cursor.Format("2006-01-02")] {
			out = append(out, cursor)
		}
		cursor = advance(r, cursor)
	}
	return out
}

func skipSet(dates []time.Time) map[string]bool {
	out := make(map[string]bool, len(dates))
	for _, d := range dates {
		out[d.UTC().Format("2006-01-02")] = true
	}
	return out
}

// alignToCadence pushes `t` forward to the next valid firing date for the
// template's cadence — for monthly/quarterly/yearly that means the configured
// day_of_month in the current period (or the next one if already past). For
// weekly cadences it snaps to the next StartDate + 7n date >= t.
func alignToCadence(r Recurring, t time.Time) time.Time {
	switch r.Frequency {
	case FrequencyDaily:
		return t
	case FrequencyWeekly:
		if !t.After(r.StartDate) {
			return r.StartDate
		}
		diffDays := int(t.Sub(r.StartDate).Hours() / 24)
		weeks := diffDays / 7
		candidate := r.StartDate.AddDate(0, 0, weeks*7)
		if candidate.Before(t) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		return candidate
	case FrequencyMonthly, FrequencyQuarterly, FrequencyYearly:
		day := dayOfMonth(r)
		candidate := monthlyAligned(t, day)
		if candidate.Before(t) {
			candidate = advance(r, candidate)
		}
		return candidate
	}
	return t
}

func monthlyAligned(t time.Time, day int) time.Time {
	year, month, _ := t.Date()
	last := lastDayOfMonth(year, month)
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func lastDayOfMonth(year int, month time.Month) int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 1, -1).Day()
}

func dayOfMonth(r Recurring) int {
	if r.DayOfMonth != nil && *r.DayOfMonth > 0 {
		return *r.DayOfMonth
	}
	return r.StartDate.Day()
}

// advance returns the next firing date after t for the template's cadence.
func advance(r Recurring, t time.Time) time.Time {
	switch r.Frequency {
	case FrequencyDaily:
		return t.AddDate(0, 0, 1)
	case FrequencyWeekly:
		return t.AddDate(0, 0, 7)
	case FrequencyMonthly:
		return monthlyAligned(t.AddDate(0, 1, 0), dayOfMonth(r))
	case FrequencyQuarterly:
		return monthlyAligned(t.AddDate(0, 3, 0), dayOfMonth(r))
	case FrequencyYearly:
		return monthlyAligned(t.AddDate(1, 0, 0), dayOfMonth(r))
	}
	return t.AddDate(0, 0, 1)
}
