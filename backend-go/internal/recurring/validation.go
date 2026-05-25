package recurring

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type validationError struct {
	Field string
	Msg   string
}

func buildInput(raw map[string]json.RawMessage) (CreateInput, *validationError) {
	var in CreateInput
	if vErr := readAccountAndAmount(raw, &in); vErr != nil {
		return in, vErr
	}
	if vErr := readOptionalStrings(raw, &in); vErr != nil {
		return in, vErr
	}
	if vErr := readCadence(raw, &in); vErr != nil {
		return in, vErr
	}
	return in, readDatesAndActive(raw, &in)
}

func readAccountAndAmount(raw map[string]json.RawMessage, in *CreateInput) *validationError {
	if err := requireInt(raw, "account_id", &in.AccountID); err != nil {
		return err
	}
	if in.AccountID <= 0 {
		return &validationError{Field: "account_id", Msg: "must be positive"}
	}
	amount, err := requireDecimal(raw, "amount")
	if err != nil {
		return err
	}
	in.Amount = amount
	if v, ok := raw["owner_user_id"]; ok && string(v) != "null" {
		var n int
		if jerr := json.Unmarshal(v, &n); jerr != nil {
			return &validationError{Field: "owner_user_id", Msg: "must be integer or null"}
		}
		in.OwnerUserID = &n
	}
	return nil
}

func readOptionalStrings(raw map[string]json.RawMessage, in *CreateInput) *validationError {
	if v, ok := raw["transaction_type"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &validationError{Field: "transaction_type", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			in.TransactionType = &s
		}
	}
	if v, ok := raw["category"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &validationError{Field: "category", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			if len(s) > 50 {
				return &validationError{Field: "category", Msg: "max 50 chars"}
			}
			in.Category = &s
		}
	}
	if v, ok := raw["description"]; ok && string(v) != "null" {
		if jerr := json.Unmarshal(v, &in.Description); jerr != nil {
			return &validationError{Field: "description", Msg: "must be a string"}
		}
	}
	if len(in.Description) > 200 {
		return &validationError{Field: "description", Msg: "max 200 chars"}
	}
	return nil
}

func readCadence(raw map[string]json.RawMessage, in *CreateInput) *validationError {
	var freq string
	if err := requireString(raw, "frequency", &freq); err != nil {
		return err
	}
	if !IsValidFrequency(freq) {
		return &validationError{Field: "frequency", Msg: "invalid cadence"}
	}
	in.Frequency = Frequency(freq)
	if v, ok := raw["day_of_month"]; ok && string(v) != "null" {
		var d int
		if jerr := json.Unmarshal(v, &d); jerr != nil {
			return &validationError{Field: "day_of_month", Msg: "must be integer or null"}
		}
		if d < 1 || d > 31 {
			return &validationError{Field: "day_of_month", Msg: "must be 1-31"}
		}
		in.DayOfMonth = &d
	}
	return nil
}

func readDatesAndActive(raw map[string]json.RawMessage, in *CreateInput) *validationError {
	startStr, err := requireDate(raw, "start_date")
	if err != nil {
		return err
	}
	in.StartDate = startStr
	if v, ok := raw["end_date"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &validationError{Field: "end_date", Msg: "must be YYYY-MM-DD or null"}
		}
		t, derr := time.Parse("2006-01-02", s)
		if derr != nil {
			return &validationError{Field: "end_date", Msg: "must be YYYY-MM-DD"}
		}
		if t.Before(in.StartDate) {
			return &validationError{Field: "end_date", Msg: "must be on or after start_date"}
		}
		in.EndDate = &t
	}
	in.Active = true
	if v, ok := raw["active"]; ok && string(v) != "null" {
		if jerr := json.Unmarshal(v, &in.Active); jerr != nil {
			return &validationError{Field: "active", Msg: "must be a boolean"}
		}
	}
	return nil
}

func requireInt(raw map[string]json.RawMessage, key string, dest *int) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be integer"}
	}
	return nil
}

func requireString(raw map[string]json.RawMessage, key string, dest *string) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	if *dest == "" {
		return &validationError{Field: key, Msg: "cannot be empty"}
	}
	return nil
}

func requireDecimal(raw map[string]json.RawMessage, key string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return decimal.Zero, &validationError{Field: key, Msg: "required"}
	}
	// Parse from JSON bytes to preserve precision per CLAUDE.md guidance.
	s := strings.Trim(string(v), `"`)
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, &validationError{Field: key, Msg: "must be a number"}
	}
	return d, nil
}

func requireDate(raw map[string]json.RawMessage, key string) (time.Time, *validationError) {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return time.Time{}, &validationError{Field: key, Msg: "required"}
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
