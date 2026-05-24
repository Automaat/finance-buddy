package recurring

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	out, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return out
}

func intPtr(v int) *int {
	out := new(int)
	*out = v
	return out
}

func TestNextOccurrence_MonthlyAlignsToDay(t *testing.T) {
	r := Recurring{
		Active:     true,
		Frequency:  FrequencyMonthly,
		StartDate:  mustParse(t, "2026-01-01"),
		DayOfMonth: intPtr(15),
		Amount:     decimal.NewFromInt(1000),
	}
	got, ok := NextOccurrence(r, mustParse(t, "2026-01-20"))
	if !ok {
		t.Fatalf("expected occurrence, got none")
	}
	if got.Format("2006-01-02") != "2026-02-15" {
		t.Errorf("expected 2026-02-15, got %s", got.Format("2006-01-02"))
	}
}

func TestNextOccurrence_MonthlyClampsToShortMonth(t *testing.T) {
	r := Recurring{
		Active:     true,
		Frequency:  FrequencyMonthly,
		StartDate:  mustParse(t, "2026-01-31"),
		DayOfMonth: intPtr(31),
	}
	got, ok := NextOccurrence(r, mustParse(t, "2026-02-01"))
	if !ok {
		t.Fatalf("expected occurrence")
	}
	if got.Format("2006-01-02") != "2026-02-28" {
		t.Errorf("February clamp failed: got %s", got.Format("2006-01-02"))
	}
}

func TestNextOccurrence_WeeklyAdvancesBy7(t *testing.T) {
	r := Recurring{
		Active:    true,
		Frequency: FrequencyWeekly,
		StartDate: mustParse(t, "2026-01-05"),
	}
	got, ok := NextOccurrence(r, mustParse(t, "2026-01-06"))
	if !ok {
		t.Fatalf("expected occurrence")
	}
	if got.Format("2006-01-02") != "2026-01-12" {
		t.Errorf("expected 2026-01-12, got %s", got.Format("2006-01-02"))
	}
}

func TestNextOccurrence_RespectsEndDate(t *testing.T) {
	end := mustParse(t, "2026-02-28")
	r := Recurring{
		Active:     true,
		Frequency:  FrequencyMonthly,
		StartDate:  mustParse(t, "2026-01-15"),
		DayOfMonth: intPtr(15),
		EndDate:    &end,
	}
	_, ok := NextOccurrence(r, mustParse(t, "2026-03-01"))
	if ok {
		t.Errorf("should have no occurrence past EndDate")
	}
}

func TestNextOccurrence_SkipsListedDates(t *testing.T) {
	r := Recurring{
		Active:       true,
		Frequency:    FrequencyMonthly,
		StartDate:    mustParse(t, "2026-01-15"),
		DayOfMonth:   intPtr(15),
		SkippedDates: []time.Time{mustParse(t, "2026-02-15")},
	}
	got, ok := NextOccurrence(r, mustParse(t, "2026-02-01"))
	if !ok {
		t.Fatalf("expected occurrence")
	}
	if got.Format("2006-01-02") != "2026-03-15" {
		t.Errorf("expected March 15 (Feb skipped), got %s", got.Format("2006-01-02"))
	}
}

func TestNextOccurrence_InactiveReturnsNone(t *testing.T) {
	r := Recurring{Active: false, Frequency: FrequencyMonthly, StartDate: mustParse(t, "2026-01-01")}
	if _, ok := NextOccurrence(r, mustParse(t, "2026-01-01")); ok {
		t.Errorf("inactive template should yield no occurrence")
	}
}

func TestDueOccurrences_FromStartUpToNow(t *testing.T) {
	r := Recurring{
		Active:     true,
		Frequency:  FrequencyMonthly,
		StartDate:  mustParse(t, "2026-01-15"),
		DayOfMonth: intPtr(15),
	}
	got := DueOccurrences(r, mustParse(t, "2026-04-20"))
	if len(got) != 4 {
		t.Fatalf("expected 4 occurrences Jan/Feb/Mar/Apr, got %d", len(got))
	}
}

func TestDueOccurrences_StartsAfterLastRun(t *testing.T) {
	last := mustParse(t, "2026-02-15")
	r := Recurring{
		Active:      true,
		Frequency:   FrequencyMonthly,
		StartDate:   mustParse(t, "2026-01-15"),
		DayOfMonth:  intPtr(15),
		LastRunDate: &last,
	}
	got := DueOccurrences(r, mustParse(t, "2026-04-20"))
	if len(got) != 2 {
		t.Fatalf("expected Mar+Apr, got %d", len(got))
	}
	if got[0].Format("2006-01-02") != "2026-03-15" {
		t.Errorf("first should be Mar 15, got %s", got[0])
	}
}

func TestDueOccurrences_SkipsListedDates(t *testing.T) {
	r := Recurring{
		Active:       true,
		Frequency:    FrequencyMonthly,
		StartDate:    mustParse(t, "2026-01-15"),
		DayOfMonth:   intPtr(15),
		SkippedDates: []time.Time{mustParse(t, "2026-02-15")},
	}
	got := DueOccurrences(r, mustParse(t, "2026-03-20"))
	if len(got) != 2 {
		t.Fatalf("expected Jan + Mar (Feb skipped), got %d", len(got))
	}
}

func TestDueOccurrences_RespectsEndDate(t *testing.T) {
	end := mustParse(t, "2026-02-28")
	r := Recurring{
		Active:     true,
		Frequency:  FrequencyMonthly,
		StartDate:  mustParse(t, "2026-01-15"),
		DayOfMonth: intPtr(15),
		EndDate:    &end,
	}
	got := DueOccurrences(r, mustParse(t, "2026-05-01"))
	if len(got) != 2 {
		t.Fatalf("expected Jan + Feb only, got %d", len(got))
	}
}

func TestIsValidFrequency(t *testing.T) {
	if !IsValidFrequency("monthly") {
		t.Errorf("monthly should be valid")
	}
	if IsValidFrequency("biweekly") {
		t.Errorf("biweekly should be invalid")
	}
}
