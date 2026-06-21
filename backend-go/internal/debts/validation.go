package debts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
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

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest
	name, vErr := validation.RequiredTrimmedString(raw, "name", "Field required", "Name cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Name = name

	dt, vErr := validation.RequiredEnumString(raw, "debt_type", validDebtTypes)
	if vErr != nil {
		return r, vErr
	}
	r.DebtType = dt

	sd, vErr := requireDateNotFuture(raw, "start_date", "Start date cannot be in the future")
	if vErr != nil {
		return r, vErr
	}
	r.StartDate = sd

	ia, vErr := validation.RequiredPositiveDecimal(raw, "initial_amount", "Field required", "Initial amount must be greater than 0")
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

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if v, ok := raw["name"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "name", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
		}
		p.Name = &s
	}
	if v, ok := raw["debt_type"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "debt_type", Msg: "must be a string"}
		}
		if _, ok := validDebtTypes[s]; !ok {
			return p, &httputil.ValidationError{Field: "debt_type", Msg: fmt.Sprintf("invalid value %q", s)}
		}
		p.DebtType = &s
	}
	if v, ok := raw["start_date"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "start_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &httputil.ValidationError{Field: "start_date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(today()) {
			return p, &httputil.ValidationError{Field: "start_date", Msg: "Start date cannot be in the future"}
		}
		p.StartDate = &t
	}
	if v, ok := raw["initial_amount"]; ok && !validation.IsNull(v) {
		d, err := validation.RawDecimal(v)
		if err != nil {
			return p, &httputil.ValidationError{Field: "initial_amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return p, &httputil.ValidationError{Field: "initial_amount", Msg: "Initial amount must be greater than 0"}
		}
		p.InitialAmount = &d
	}
	if v, ok := raw["interest_rate"]; ok && !validation.IsNull(v) {
		d, err := validation.RawDecimal(v)
		if err != nil {
			return p, &httputil.ValidationError{Field: "interest_rate", Msg: "must be a number"}
		}
		if d.IsNegative() {
			return p, &httputil.ValidationError{Field: "interest_rate", Msg: "Interest rate must be greater than or equal to 0"}
		}
		p.InterestRate = &d
	}
	if v, ok := raw["currency"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
		}
		if s != "PLN" {
			return p, &httputil.ValidationError{Field: "currency", Msg: "Currency must be 'PLN'"}
		}
		p.Currency = &s
	}
	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		p.NotesSet = true
		p.Notes = &s
	}
	return p, nil
}

func requireOwnerUserID(raw map[string]json.RawMessage) (*int, *httputil.ValidationError) {
	return validation.RequiredIntOrNull(raw, "owner_user_id")
}

func requireDateNotFuture(raw map[string]json.RawMessage, key, msg string) (time.Time, *httputil.ValidationError) {
	t, vErr := validation.RequiredDate(raw, key)
	if vErr != nil {
		return time.Time{}, vErr
	}
	if t.After(today()) {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return t, nil
}

func requireNonNegativeDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	d, err := validation.RawDecimal(v)
	if err != nil {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	if d.IsNegative() {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return d, nil
}

func optionalCurrencyPLN(raw map[string]json.RawMessage) (string, *httputil.ValidationError) {
	v, ok := raw["currency"]
	if !ok || validation.IsNull(v) {
		return "PLN", nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
	}
	if s != "PLN" {
		return "", &httputil.ValidationError{Field: "currency", Msg: "Currency must be 'PLN'"}
	}
	return s, nil
}

func today() time.Time {
	t := time.Now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
