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

	sd, vErr := validation.RequiredDateNotFutureWithMessage(raw, "start_date", time.Now, "Start date cannot be in the future")
	if vErr != nil {
		return r, vErr
	}
	r.StartDate = sd

	ia, vErr := validation.RequiredPositiveDecimal(raw, "initial_amount", "Field required", "Initial amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.InitialAmount = ia

	ir, vErr := validation.RequiredNonNegativeDecimal(
		raw,
		"interest_rate",
		"Field required",
		"Interest rate must be greater than or equal to 0",
	)
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
	if t, vErr := validation.OptionalDateNotFuture(
		raw,
		"start_date",
		time.Now,
		"Start date cannot be in the future",
	); vErr != nil {
		return p, vErr
	} else if t != nil {
		p.StartDate = t
	}
	if d, vErr := validation.OptionalPositiveDecimal(
		raw,
		"initial_amount",
		"Initial amount must be greater than 0",
	); vErr != nil {
		return p, vErr
	} else if d != nil {
		p.InitialAmount = d
	}
	if d, vErr := validation.OptionalNonNegativeDecimal(
		raw,
		"interest_rate",
		"Interest rate must be greater than or equal to 0",
	); vErr != nil {
		return p, vErr
	} else if d != nil {
		p.InterestRate = d
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
