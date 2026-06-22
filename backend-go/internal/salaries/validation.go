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
	t, vErr := validation.RequiredDateNotFuture(raw, "date", now)
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	amount, vErr := validation.RequiredPositiveDecimal(raw, "gross_amount", "Field required", "Gross amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.GrossAmount = amount

	contract, vErr := validation.RequiredEnumString(raw, "contract_type", validContractTypes)
	if vErr != nil {
		return r, vErr
	}
	r.ContractType = contract

	company, vErr := validation.RequiredTrimmedString(raw, "company", "Field required", "Company cannot be empty")
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
	if t, vErr := validation.OptionalDateNotFuture(raw, "date", now, "Date cannot be in the future"); vErr != nil {
		return p, vErr
	} else if t != nil {
		p.Date = t
	}
	if d, vErr := validation.OptionalPositiveDecimal(raw, "gross_amount", "Gross amount must be greater than 0"); vErr != nil {
		return p, vErr
	} else if d != nil {
		p.GrossAmount = d
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
	if _, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		ownerID, vErr := validation.OptionalInt(raw, "owner_user_id", "must be an integer")
		if vErr != nil {
			return p, vErr
		}
		p.OwnerUserID = ownerID
	}
	return p, nil
}
