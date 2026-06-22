package bonusevents

import (
	"encoding/json"
	"fmt"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// buildCreateRequest reads + validates the POST body. JSON numbers are parsed
// from raw bytes (decimal.NewFromString) to preserve Numeric precision —
// going through float64 introduces IEEE754 rounding that diverges from
// Python's Decimal(str(float)) round-trip.
//
// Returns the value type rather than a pointer so callers can use the result
// directly without a nil-check on success (nilaway-friendly).
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest

	t, vErr := validation.RequiredDateNotFuture(raw, "date", nil)
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	amount, vErr := validation.RequiredPositiveDecimal(raw, "amount", "Field required", "Amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.Amount = amount

	currency, vErr := validation.OptionalUpperTrimmedEnumString(
		raw,
		"currency",
		validCurrencies,
		"PLN",
		"Currency must be one of [CHF, EUR, GBP, PLN, USD]",
	)
	if vErr != nil {
		return r, vErr
	}
	r.Currency = currency

	r.Type, vErr = validation.RequiredEnumString(raw, "type", validBonusTypes)
	if vErr != nil {
		return r, vErr
	}

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

	r.ContractType, vErr = validation.RequiredEnumString(raw, "contract_type", validContractTypes)
	if vErr != nil {
		return r, vErr
	}

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

// buildUpdatePatch parses the PATCH body. Null fields are no-ops (Pydantic
// semantics).
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if vErr := patchScalars(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchEnums(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

func patchScalars(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if t, vErr := validation.OptionalDateNotFuture(raw, "date", nil, "Date cannot be in the future"); vErr != nil {
		return vErr
	} else if t != nil {
		p.Date = t
	}
	if d, vErr := validation.OptionalPositiveDecimal(raw, "amount", "Amount must be greater than 0"); vErr != nil {
		return vErr
	} else if d != nil {
		p.Amount = d
	}
	if s, vErr := validation.OptionalUpperTrimmedEnumStringPtr(
		raw,
		"currency",
		validCurrencies,
		"Currency must be one of [CHF, EUR, GBP, PLN, USD]",
	); vErr != nil {
		return vErr
	} else if s != nil {
		p.Currency = s
	}
	if s, vErr := validation.OptionalTrimmedString(raw, "company", "Company cannot be empty"); vErr != nil {
		return vErr
	} else if s != nil {
		p.Company = s
	}
	if _, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		ownerID, vErr := validation.OptionalInt(raw, "owner_user_id", "must be an integer")
		if vErr != nil {
			return vErr
		}
		p.OwnerUserID = ownerID
	}
	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	return nil
}

func patchEnums(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if v, ok := raw["type"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "type", Msg: "must be a string"}
		}
		if _, ok := validBonusTypes[s]; !ok {
			return &httputil.ValidationError{Field: "type", Msg: fmt.Sprintf("invalid bonus type %q", s)}
		}
		p.Type = &s
	}
	if v, ok := raw["contract_type"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "contract_type", Msg: "must be a string"}
		}
		if _, ok := validContractTypes[s]; !ok {
			return &httputil.ValidationError{Field: "contract_type", Msg: fmt.Sprintf("invalid contract type %q", s)}
		}
		p.ContractType = &s
	}
	return nil
}
