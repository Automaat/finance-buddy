package salaries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// createRequest is the validated POST body.
type createRequest struct {
	Date         time.Time
	GrossAmount  decimal.Decimal
	ContractType string
	Company      string
	Owner        string
}

// buildCreateRequest validates the body. Numbers are parsed via
// decimal.NewFromString on raw token bytes (preserves precision); required
// fields surface "Field required" on missing.
func buildCreateRequest(raw map[string]json.RawMessage, now func() time.Time) (createRequest, *validationError) {
	var r createRequest
	t, vErr := requireDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	if t.After(truncateDay(now().UTC())) {
		return r, &validationError{Field: "date", Msg: "Date cannot be in the future"}
	}
	r.Date = t

	amount, vErr := requirePositiveDecimal(raw, "gross_amount", "Gross amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.GrossAmount = amount

	contract, vErr := requireEnumString(raw, "contract_type", validContractTypes)
	if vErr != nil {
		return r, vErr
	}
	r.ContractType = contract

	company, vErr := requireString(raw, "company", "Company cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Company = company

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Owner = owner
	return r, nil
}

// buildUpdatePatch reads the PATCH body. Null = no-op (Pydantic semantics).
func buildUpdatePatch(raw map[string]json.RawMessage, now func() time.Time) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if v, ok := raw["date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(truncateDay(now().UTC())) {
			return p, &validationError{Field: "date", Msg: "Date cannot be in the future"}
		}
		p.Date = &t
	}
	if v, ok := raw["gross_amount"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return p, &validationError{Field: "gross_amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return p, &validationError{Field: "gross_amount", Msg: "Gross amount must be greater than 0"}
		}
		p.GrossAmount = &d
	}
	if v, ok := raw["contract_type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "contract_type", Msg: "must be a string"}
		}
		if _, ok := validContractTypes[s]; !ok {
			return p, &validationError{Field: "contract_type", Msg: fmt.Sprintf("invalid contract type %q", s)}
		}
		p.ContractType = &s
	}
	if v, ok := raw["company"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "company", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &validationError{Field: "company", Msg: "Company cannot be empty"}
		}
		p.Company = &s
	}
	if v, ok := raw["owner"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "owner", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &validationError{Field: "owner", Msg: "Owner cannot be empty"}
		}
		p.Owner = &s
	}
	return p, nil
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

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
