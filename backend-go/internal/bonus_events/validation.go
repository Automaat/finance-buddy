package bonusevents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

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

	t, vErr := validation.RequiredDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	if t.After(today()) {
		return r, &httputil.ValidationError{Field: "date", Msg: "Date cannot be in the future"}
	}
	r.Date = t

	amount, vErr := requirePositiveDecimal(raw, "amount", "Amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.Amount = amount

	currency, vErr := optionalCurrency(raw, "PLN")
	if vErr != nil {
		return r, vErr
	}
	r.Currency = currency

	r.Type, vErr = validation.RequiredEnumString(raw, "type", validBonusTypes)
	if vErr != nil {
		return r, vErr
	}

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
	if v, ok := raw["date"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(today()) {
			return &httputil.ValidationError{Field: "date", Msg: "Date cannot be in the future"}
		}
		p.Date = &t
	}
	if v, ok := raw["amount"]; ok && !validation.IsNull(v) {
		d, err := validation.RawDecimal(v)
		if err != nil {
			return &httputil.ValidationError{Field: "amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return &httputil.ValidationError{Field: "amount", Msg: "Amount must be greater than 0"}
		}
		p.Amount = &d
	}
	if v, ok := raw["currency"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
		}
		s = strings.ToUpper(strings.TrimSpace(s))
		if _, ok := validCurrencies[s]; !ok {
			return &httputil.ValidationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
		}
		p.Currency = &s
	}
	if v, ok := raw["company"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "company", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return &httputil.ValidationError{Field: "company", Msg: "Company cannot be empty"}
		}
		p.Company = &s
	}
	if v, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		if validation.IsNull(v) {
			p.OwnerUserID = nil
		} else {
			var n int
			if err := json.Unmarshal(v, &n); err != nil {
				return &httputil.ValidationError{Field: "owner_user_id", Msg: "must be an integer"}
			}
			p.OwnerUserID = &n
		}
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

// --- shared decoders ---

func requireString(raw map[string]json.RawMessage, key, emptyMsg string) (string, *httputil.ValidationError) {
	return validation.RequiredTrimmedString(raw, key, "Field required", emptyMsg)
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

func optionalCurrency(raw map[string]json.RawMessage, fallback string) (string, *httputil.ValidationError) {
	v, ok := raw["currency"]
	if !ok || validation.IsNull(v) {
		return fallback, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	if _, ok := validCurrencies[s]; !ok {
		return "", &httputil.ValidationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
	}
	return s, nil
}

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
