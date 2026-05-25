package debts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var validDebtTypes = map[string]struct{}{
	"mortgage": {}, "installment_0percent": {},
}

type createRequest struct {
	Name          string
	DebtType      string
	StartDate     time.Time
	InitialAmount decimal.Decimal
	InterestRate  decimal.Decimal
	Currency      string
	Notes         *string
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	name, vErr := requireString(raw, "name", "Name cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Name = name

	dt, vErr := requireEnumString(raw, "debt_type", validDebtTypes)
	if vErr != nil {
		return r, vErr
	}
	r.DebtType = dt

	sd, vErr := requireDateNotFuture(raw, "start_date", "Start date cannot be in the future")
	if vErr != nil {
		return r, vErr
	}
	r.StartDate = sd

	ia, vErr := requirePositiveDecimal(raw, "initial_amount", "Initial amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.InitialAmount = ia

	ir, vErr := requireNonNegativeDecimal(raw, "interest_rate", "Interest rate must be greater than or equal to 0")
	if vErr != nil {
		return r, vErr
	}
	r.InterestRate = ir

	cur, vErr := optionalCurrencyPLN(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Currency = cur

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if v, ok := raw["name"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "name", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &validationError{Field: "name", Msg: "Name cannot be empty"}
		}
		p.Name = &s
	}
	if v, ok := raw["debt_type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "debt_type", Msg: "must be a string"}
		}
		if _, ok := validDebtTypes[s]; !ok {
			return p, &validationError{Field: "debt_type", Msg: fmt.Sprintf("invalid value %q", s)}
		}
		p.DebtType = &s
	}
	if v, ok := raw["start_date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "start_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &validationError{Field: "start_date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(today()) {
			return p, &validationError{Field: "start_date", Msg: "Start date cannot be in the future"}
		}
		p.StartDate = &t
	}
	if v, ok := raw["initial_amount"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return p, &validationError{Field: "initial_amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return p, &validationError{Field: "initial_amount", Msg: "Initial amount must be greater than 0"}
		}
		p.InitialAmount = &d
	}
	if v, ok := raw["interest_rate"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return p, &validationError{Field: "interest_rate", Msg: "must be a number"}
		}
		if d.IsNegative() {
			return p, &validationError{Field: "interest_rate", Msg: "Interest rate must be greater than or equal to 0"}
		}
		p.InterestRate = &d
	}
	if v, ok := raw["currency"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "currency", Msg: "must be a string"}
		}
		if s != "PLN" {
			return p, &validationError{Field: "currency", Msg: "Currency must be 'PLN'"}
		}
		p.Currency = &s
	}
	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "notes", Msg: "must be a string"}
		}
		p.NotesSet = true
		p.Notes = &s
	}
	return p, nil
}

func requireOwnerUserID(raw map[string]json.RawMessage) (*int, *validationError) {
	const key = "owner_user_id"
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

func requireString(raw map[string]json.RawMessage, key, emptyMsg string) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &validationError{Field: key, Msg: emptyMsg}
	}
	return s, nil
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

func requireDateNotFuture(raw map[string]json.RawMessage, key, msg string) (time.Time, *validationError) {
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
	if t.After(today()) {
		return time.Time{}, &validationError{Field: key, Msg: msg}
	}
	return t, nil
}

func requirePositiveDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "Field required"}
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "must be a number"}
	}
	if !d.IsPositive() {
		return decimal.Decimal{}, &validationError{Field: key, Msg: msg}
	}
	return d, nil
}

func requireNonNegativeDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "Field required"}
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "must be a number"}
	}
	if d.IsNegative() {
		return decimal.Decimal{}, &validationError{Field: key, Msg: msg}
	}
	return d, nil
}

func optionalCurrencyPLN(raw map[string]json.RawMessage) (string, *validationError) {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return "PLN", nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: "currency", Msg: "must be a string"}
	}
	if s != "PLN" {
		return "", &validationError{Field: "currency", Msg: "Currency must be 'PLN'"}
	}
	return s, nil
}

func today() time.Time {
	t := time.Now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
