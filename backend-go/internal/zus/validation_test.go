package zus

import (
	"encoding/json"
	"testing"
	"time"
)

var fixedNow = func() time.Time { return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC) }

func rawJSON(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", k, err)
		}
		out[k] = b
	}
	return out
}

func validInputs() map[string]any {
	return map[string]any{
		"owner_user_id":                1,
		"birth_date":                   "1990-06-15",
		"gender":                       "M",
		"retirement_age":               65,
		"current_gross_monthly_salary": 10000,
		"work_start_year":              2015,
	}
}

func TestBuildInputsOK(t *testing.T) {
	c, vErr := buildInputs(rawJSON(t, validInputs()), fixedNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if c.Inputs.Gender != "M" || c.Inputs.RetirementAge != 65 {
		t.Fatalf("inputs mismatch: %+v", c)
	}
	if c.BirthDate.Year() != 1990 {
		t.Fatalf("birth year mismatch: %s", c.BirthDate)
	}
	if c.Inputs.BirthYear != 1990 {
		t.Fatalf("birth year not propagated: %+v", c.Inputs)
	}
}

func TestBuildInputsInvalidGender(t *testing.T) {
	m := validInputs()
	m["gender"] = "Z"
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "gender" {
		t.Fatalf("expected gender error, got %+v", vErr)
	}
}

func TestBuildInputsRetirementAgeOutOfRange(t *testing.T) {
	m := validInputs()
	m["retirement_age"] = 40
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "retirement_age" {
		t.Fatalf("expected retirement_age error, got %+v", vErr)
	}
}

func TestBuildInputsNegativeSalary(t *testing.T) {
	m := validInputs()
	m["current_gross_monthly_salary"] = -1
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "current_gross_monthly_salary" {
		t.Fatalf("expected salary error, got %+v", vErr)
	}
}

func TestBuildInputsRateOutOfRange(t *testing.T) {
	m := validInputs()
	m["inflation_rate"] = 25.0
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "inflation_rate" {
		t.Fatalf("expected inflation_rate error, got %+v", vErr)
	}
}

func TestBuildInputsRetirementBeforeCurrentAge(t *testing.T) {
	m := validInputs()
	m["birth_date"] = "1950-06-15"
	m["retirement_age"] = 55
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "retirement_age" {
		t.Fatalf("expected retirement_age error, got %+v", vErr)
	}
}

func TestValidateAgeRelationship(t *testing.T) {
	birth := time.Date(1990, 6, 15, 0, 0, 0, 0, time.UTC)
	if vErr := validateAgeRelationship(birth, 65, fixedNow); vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if vErr := validateAgeRelationship(birth, 30, fixedNow); vErr == nil {
		t.Fatal("expected error when retirement <= current age")
	}
}

func TestSalaryHistoryParses(t *testing.T) {
	m := validInputs()
	m["salary_history"] = []map[string]any{
		{"year": 2020, "annual_gross": 60000},
		{"year": 2021, "annual_gross": 70000},
	}
	c, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if len(c.Inputs.SalaryHistory) != 2 {
		t.Fatalf("expected 2 entries, got %+v", c.Inputs.SalaryHistory)
	}
	if c.Inputs.SalaryHistory[2020] != 60000 {
		t.Fatalf("entry mismatch: %+v", c.Inputs.SalaryHistory)
	}
}

func TestSalaryHistoryMissingField(t *testing.T) {
	m := validInputs()
	m["salary_history"] = []map[string]any{{"year": 2020}}
	_, vErr := buildInputs(rawJSON(t, m), fixedNow)
	if vErr == nil || vErr.Field != "salary_history" {
		t.Fatalf("expected salary_history error, got %+v", vErr)
	}
}

func TestOptionalIntRangeAccepts(t *testing.T) {
	raw := rawJSON(t, map[string]any{"x": 5})
	n, vErr := optionalIntRange(raw, "x", 10, 0, 10, "msg")
	if vErr != nil || n != 5 {
		t.Fatalf("expected 5, got %d err=%+v", n, vErr)
	}
}

func TestOptionalIntRangeFallback(t *testing.T) {
	n, vErr := optionalIntRange(map[string]json.RawMessage{}, "x", 7, 0, 10, "msg")
	if vErr != nil || n != 7 {
		t.Fatalf("expected fallback 7, got %d err=%+v", n, vErr)
	}
}

func TestOptionalIntRangeOutOfRange(t *testing.T) {
	raw := rawJSON(t, map[string]any{"x": 50})
	_, vErr := optionalIntRange(raw, "x", 0, 0, 10, "msg")
	if vErr == nil || vErr.Field != "x" {
		t.Fatalf("expected range error, got %+v", vErr)
	}
}

func TestOptionalIntRangeNullIsError(t *testing.T) {
	raw := rawJSON(t, map[string]any{"x": nil})
	_, vErr := optionalIntRange(raw, "x", 0, 0, 10, "msg")
	if vErr == nil || vErr.Field != "x" {
		t.Fatalf("expected error on null, got %+v", vErr)
	}
}
