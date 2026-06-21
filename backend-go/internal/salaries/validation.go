package salaries

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// createRequest is the validated POST body. OwnerUserID is required as a
// key; an explicit null yields nil (jointly owned).
type createRequest struct {
	Date         time.Time
	GrossAmount  decimal.Decimal
	ContractType string
	Company      string
	OwnerUserID  *int
}

// buildCreateRequest validates the body. Numbers are parsed via
// decimal.NewFromString on raw token bytes (preserves precision); required
// fields surface "Field required" on missing.
func buildCreateRequest(raw map[string]json.RawMessage, now func() time.Time) (createRequest, *httputil.ValidationError) {
	var r createRequest
	t, vErr := validation.RequiredDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	if t.After(truncateDay(now().UTC())) {
		return r, &httputil.ValidationError{Field: "date", Msg: "Date cannot be in the future"}
	}
	r.Date = t

	amount, vErr := requirePositiveDecimal(raw, "gross_amount", "Gross amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.GrossAmount = amount

	contract, vErr := validation.RequiredEnumString(raw, "contract_type", validContractTypes)
	if vErr != nil {
		return r, vErr
	}
	r.ContractType = contract

	company, vErr := requireString(raw, "company", "Company cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Company = company

	ownerID, vErr := validation.RequiredIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = ownerID
	return r, nil
}

// buildUpdatePatch reads the PATCH body. Null = no-op (Pydantic semantics).
func buildUpdatePatch(raw map[string]json.RawMessage, now func() time.Time) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if v, ok := raw["date"]; ok && !validation.IsNull(v) {
		t, err := validation.RawDate(v)
		if err != nil {
			if validation.IsRawDateFormatError(err) {
				return p, &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
			}
			return p, &httputil.ValidationError{Field: "date", Msg: "must be a string"}
		}
		if t.After(truncateDay(now().UTC())) {
			return p, &httputil.ValidationError{Field: "date", Msg: "Date cannot be in the future"}
		}
		p.Date = &t
	}
	if v, ok := raw["gross_amount"]; ok && !validation.IsNull(v) {
		d, err := validation.RawDecimal(v)
		if err != nil {
			return p, &httputil.ValidationError{Field: "gross_amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return p, &httputil.ValidationError{Field: "gross_amount", Msg: "Gross amount must be greater than 0"}
		}
		p.GrossAmount = &d
	}
	if v, ok := raw["contract_type"]; ok && !validation.IsNull(v) {
		s, err := validation.RawString(v)
		if err != nil {
			return p, &httputil.ValidationError{Field: "contract_type", Msg: "must be a string"}
		}
		if _, ok := validContractTypes[s]; !ok {
			return p, &httputil.ValidationError{Field: "contract_type", Msg: fmt.Sprintf("invalid contract type %q", s)}
		}
		p.ContractType = &s
	}
	if v, ok := raw["company"]; ok && !validation.IsNull(v) {
		s, err := validation.RawTrimmedString(v)
		if err != nil {
			return p, &httputil.ValidationError{Field: "company", Msg: "must be a string"}
		}
		if s == "" {
			return p, &httputil.ValidationError{Field: "company", Msg: "Company cannot be empty"}
		}
		p.Company = &s
	}
	if v, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		if validation.IsNull(v) {
			p.OwnerUserID = nil
		} else {
			n, err := validation.RawInt(v)
			if err != nil {
				return p, &httputil.ValidationError{Field: "owner_user_id", Msg: "must be an integer"}
			}
			p.OwnerUserID = &n
		}
	}
	return p, nil
}

func requireString(raw map[string]json.RawMessage, key, emptyMsg string) (string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return "", &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	s, err := validation.RawTrimmedString(v)
	if err != nil {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if s == "" {
		return "", &httputil.ValidationError{Field: key, Msg: emptyMsg}
	}
	return s, nil
}

func requirePositiveDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	d, err := validation.RawDecimal(v)
	if err != nil {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	if !d.IsPositive() {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return d, nil
}
