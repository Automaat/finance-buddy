package recurring

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

func buildInput(raw map[string]json.RawMessage) (CreateInput, *httputil.ValidationError) {
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

func readAccountAndAmount(raw map[string]json.RawMessage, in *CreateInput) *httputil.ValidationError {
	accountID, err := validation.RequiredInt(raw, "account_id", "required", "must be integer")
	if err != nil {
		return err
	}
	in.AccountID = accountID
	if in.AccountID <= 0 {
		return &httputil.ValidationError{Field: "account_id", Msg: "must be positive"}
	}
	amount, err := requireDecimal(raw, "amount")
	if err != nil {
		return err
	}
	in.Amount = amount
	ownerID, vErr := validation.OptionalInt(raw, "owner_user_id", "must be integer or null")
	if vErr != nil {
		return vErr
	}
	if ownerID != nil {
		in.OwnerUserID = ownerID
	}
	return nil
}

func readOptionalStrings(raw map[string]json.RawMessage, in *CreateInput) *httputil.ValidationError {
	if v, ok := raw["transaction_type"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &httputil.ValidationError{Field: "transaction_type", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			in.TransactionType = &s
		}
	}
	if v, ok := raw["category"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &httputil.ValidationError{Field: "category", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			if len(s) > 50 {
				return &httputil.ValidationError{Field: "category", Msg: "max 50 chars"}
			}
			in.Category = &s
		}
	}
	if v, ok := raw["description"]; ok && string(v) != "null" {
		if jerr := json.Unmarshal(v, &in.Description); jerr != nil {
			return &httputil.ValidationError{Field: "description", Msg: "must be a string"}
		}
	}
	if len(in.Description) > 200 {
		return &httputil.ValidationError{Field: "description", Msg: "max 200 chars"}
	}
	return nil
}

func readCadence(raw map[string]json.RawMessage, in *CreateInput) *httputil.ValidationError {
	freq, vErr := validation.RequiredString(raw, "frequency", "required", "cannot be empty")
	if vErr != nil {
		return vErr
	}
	if !IsValidFrequency(freq) {
		return &httputil.ValidationError{Field: "frequency", Msg: "invalid cadence"}
	}
	in.Frequency = Frequency(freq)
	if v, ok := raw["day_of_month"]; ok && string(v) != "null" {
		var d int
		if jerr := json.Unmarshal(v, &d); jerr != nil {
			return &httputil.ValidationError{Field: "day_of_month", Msg: "must be integer or null"}
		}
		if d < 1 || d > 31 {
			return &httputil.ValidationError{Field: "day_of_month", Msg: "must be 1-31"}
		}
		in.DayOfMonth = &d
	}
	return nil
}

func readDatesAndActive(raw map[string]json.RawMessage, in *CreateInput) *httputil.ValidationError {
	startDate, err := requireDate(raw, "start_date")
	if err != nil {
		return err
	}
	in.StartDate = startDate
	if v, ok := raw["end_date"]; ok && string(v) != "null" {
		var s string
		if jerr := json.Unmarshal(v, &s); jerr != nil {
			return &httputil.ValidationError{Field: "end_date", Msg: "must be YYYY-MM-DD or null"}
		}
		t, derr := time.Parse("2006-01-02", s)
		if derr != nil {
			return &httputil.ValidationError{Field: "end_date", Msg: "must be YYYY-MM-DD"}
		}
		if t.Before(in.StartDate) {
			return &httputil.ValidationError{Field: "end_date", Msg: "must be on or after start_date"}
		}
		in.EndDate = &t
	}
	in.Active = true
	active, vErr := validation.OptionalBoolDefault(raw, "active", true)
	if vErr != nil {
		return vErr
	}
	in.Active = active
	return nil
}

func requireDecimal(raw map[string]json.RawMessage, key string) (decimal.Decimal, *httputil.ValidationError) {
	return validation.RequiredDecimalStringOrNumber(raw, key, "required")
}

func requireDate(raw map[string]json.RawMessage, key string) (time.Time, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be YYYY-MM-DD"}
	}
	return t, nil
}
