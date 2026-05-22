package zus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type calculateInputs struct {
	Inputs    Inputs
	BirthDate time.Time
}

// buildInputs validates the calculate POST body. Mirrors the Pydantic
// validators on ZusCalculatorInputs (gender M/F, retirement_age 55..70,
// non-negative salaries, rates 0..20, work_start_year 1970..2030,
// retirement_age strictly greater than current age).
func buildInputs(raw map[string]json.RawMessage, now func() time.Time) (calculateInputs, *validationError) {
	var c calculateInputs
	var err *validationError
	c.Inputs.OwnerUserID, err = requireIntOrNull(raw, "owner_user_id")
	if err != nil {
		return c, err
	}
	c.BirthDate, err = requireDate(raw, "birth_date")
	if err != nil {
		return c, err
	}
	c.Inputs.BirthYear = c.BirthDate.Year()

	c.Inputs.Gender, err = requireEnumString(raw, "gender", map[string]struct{}{"M": {}, "F": {}})
	if err != nil {
		return c, err
	}

	c.Inputs.RetirementAge, err = optionalIntRange(raw, "retirement_age", 65, 55, 70, "Retirement age must be between 55 and 70")
	if err != nil {
		return c, err
	}
	c.Inputs.CurrentGrossMonthlySalary, err = requireNonNegativeFloat(raw, "current_gross_monthly_salary", "Salary cannot be negative")
	if err != nil {
		return c, err
	}
	if vErr := readRates(raw, &c.Inputs); vErr != nil {
		return c, vErr
	}
	if vErr := readMisc(raw, &c.Inputs); vErr != nil {
		return c, vErr
	}
	if vErr := readSalaryHistory(raw, &c.Inputs); vErr != nil {
		return c, vErr
	}
	if vErr := validateAgeRelationship(c.BirthDate, c.Inputs.RetirementAge, now); vErr != nil {
		return c, vErr
	}
	return c, nil
}

func readRates(raw map[string]json.RawMessage, in *Inputs) *validationError {
	rates := []struct {
		key      string
		dest     *float64
		fallback float64
	}{
		{"salary_growth_rate", &in.SalaryGrowthRate, 3.0},
		{"inflation_rate", &in.InflationRate, 3.0},
		{"valorization_rate_konto", &in.ValorizationRateKonto, 5.0},
		{"valorization_rate_subkonto", &in.ValorizationRateSubkonto, 4.0},
	}
	for _, r := range rates {
		v, err := optionalFloatRange(raw, r.key, r.fallback, 0, 20, "Rate must be between 0% and 20%")
		if err != nil {
			return err
		}
		*r.dest = v
	}
	return nil
}

func readMisc(raw map[string]json.RawMessage, in *Inputs) *validationError {
	hasOFE, err := optionalBool(raw, "has_ofe", false)
	if err != nil {
		return err
	}
	in.HasOFE = hasOFE
	kapital, err := optionalNonNegativeFloat(raw, "kapital_poczatkowy", 0.0, "Kapitał początkowy cannot be negative")
	if err != nil {
		return err
	}
	in.KapitalPoczatkowy = kapital
	work, err := requireIntRange(raw, "work_start_year", 1970, 2030, "Work start year must be between 1970 and 2030")
	if err != nil {
		return err
	}
	in.WorkStartYear = work
	return nil
}

func readSalaryHistory(raw map[string]json.RawMessage, in *Inputs) *validationError {
	in.SalaryHistory = map[int]float64{}
	v, ok := raw["salary_history"]
	if !ok {
		return nil
	}
	if isNull(v) {
		return &validationError{Field: "salary_history", Msg: "must be an array"}
	}
	var entries []map[string]any
	if err := json.Unmarshal(v, &entries); err != nil {
		return &validationError{Field: "salary_history", Msg: "must be an array"}
	}
	for _, e := range entries {
		yVal, hasYear := e["year"]
		gVal, hasGross := e["annual_gross"]
		if !hasYear || !hasGross {
			return &validationError{Field: "salary_history", Msg: "entries require 'year' and 'annual_gross'"}
		}
		yf, ok := yVal.(float64)
		if !ok {
			return &validationError{Field: "salary_history", Msg: "year must be a number"}
		}
		gf, ok := gVal.(float64)
		if !ok {
			return &validationError{Field: "salary_history", Msg: "annual_gross must be a number"}
		}
		in.SalaryHistory[int(yf)] = gf
	}
	return nil
}

func validateAgeRelationship(birth time.Time, retirementAge int, now func() time.Time) *validationError {
	today := now().UTC()
	currentAge := today.Year() - birth.Year()
	if today.Month() < birth.Month() ||
		(today.Month() == birth.Month() && today.Day() < birth.Day()) {
		currentAge--
	}
	if retirementAge <= currentAge {
		return &validationError{
			Field: "retirement_age",
			Msg:   "Retirement age must be greater than current age",
		}
	}
	return nil
}

// --- helpers ---

// requireIntOrNull reads an integer key that must be present; an explicit
// null is allowed and yields nil (jointly owned).
func requireIntOrNull(raw map[string]json.RawMessage, key string) (*int, *validationError) {
	v, ok := raw[key]
	if !ok {
		return nil, &validationError{Field: key, Msg: "Field required"}
	}
	if isNull(v) {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return nil, &validationError{Field: key, Msg: "must be an integer"}
	}
	return &n, nil
}

func requireDate(raw map[string]json.RawMessage, key string) (time.Time, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return time.Time{}, &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be a string"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be YYYY-MM-DD"}
	}
	return t, nil
}

func requireEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

func requireNonNegativeFloat(raw map[string]json.RawMessage, key, msg string) (float64, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	if f < 0 {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return f, nil
}

// optionalNonNegativeFloat: omitted -> fallback. Present-as-null -> 422
// (Pydantic rejects null on non-Optional typed defaults).
func optionalNonNegativeFloat(raw map[string]json.RawMessage, key string, fallback float64, msg string) (float64, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	if f < 0 {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return f, nil
}

func optionalFloatRange(raw map[string]json.RawMessage, key string, fallback, lo, hi float64, msg string) (float64, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return 0, &validationError{Field: key, Msg: "must be a number"}
	}
	if f < lo || f > hi {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return f, nil
}

func optionalIntRange(raw map[string]json.RawMessage, key string, fallback, lo, hi int, msg string) (int, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	if n < lo || n > hi {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return n, nil
}

func requireIntRange(raw map[string]json.RawMessage, key string, lo, hi int, msg string) (int, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	if n < lo || n > hi {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return n, nil
}

func optionalBool(raw map[string]json.RawMessage, key string, fallback bool) (bool, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return false, &validationError{Field: key, Msg: "must be a boolean"}
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, &validationError{Field: key, Msg: "must be a boolean"}
	}
	return b, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
