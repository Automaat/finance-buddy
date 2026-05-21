package bonusevents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// buildCreateRequest reads + validates the POST body. JSON numbers are parsed
// from raw bytes (decimal.NewFromString) to preserve Numeric precision —
// going through float64 introduces IEEE754 rounding that diverges from
// Python's Decimal(str(float)) round-trip.
func buildCreateRequest(raw map[string]json.RawMessage) (*createRequest, *validationError) {
	r := &createRequest{}

	t, vErr := requireDate(raw, "date")
	if vErr != nil {
		return nil, vErr
	}
	if t.After(today()) {
		return nil, &validationError{Field: "date", Msg: "Date cannot be in the future"}
	}
	r.Date = t

	amount, vErr := requirePositiveDecimal(raw, "amount", "Amount must be greater than 0")
	if vErr != nil {
		return nil, vErr
	}
	r.Amount = amount

	currency, vErr := optionalCurrency(raw, "PLN")
	if vErr != nil {
		return nil, vErr
	}
	r.Currency = currency

	r.Type, vErr = requireEnumString(raw, "type", validBonusTypes)
	if vErr != nil {
		return nil, vErr
	}

	company, vErr := requireString(raw, "company", "Company cannot be empty")
	if vErr != nil {
		return nil, vErr
	}
	r.Company = company

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return nil, vErr
	}
	r.Owner = owner

	r.ContractType, vErr = requireEnumString(raw, "contract_type", validContractTypes)
	if vErr != nil {
		return nil, vErr
	}

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return nil, &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

// buildUpdatePatch parses the PATCH body. Null fields are no-ops (Pydantic
// semantics).
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if vErr := patchScalars(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchEnums(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

func patchScalars(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	if v, ok := raw["date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(today()) {
			return &validationError{Field: "date", Msg: "Date cannot be in the future"}
		}
		p.Date = &t
	}
	if v, ok := raw["amount"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return &validationError{Field: "amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return &validationError{Field: "amount", Msg: "Amount must be greater than 0"}
		}
		p.Amount = &d
	}
	if v, ok := raw["currency"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "currency", Msg: "must be a string"}
		}
		s = strings.ToUpper(strings.TrimSpace(s))
		if _, ok := validCurrencies[s]; !ok {
			return &validationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
		}
		p.Currency = &s
	}
	if v, ok := raw["company"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "company", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return &validationError{Field: "company", Msg: "Company cannot be empty"}
		}
		p.Company = &s
	}
	if v, ok := raw["owner"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "owner", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return &validationError{Field: "owner", Msg: "Owner cannot be empty"}
		}
		p.Owner = &s
	}
	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	return nil
}

func patchEnums(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	if v, ok := raw["type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "type", Msg: "must be a string"}
		}
		if _, ok := validBonusTypes[s]; !ok {
			return &validationError{Field: "type", Msg: fmt.Sprintf("invalid bonus type %q", s)}
		}
		p.Type = &s
	}
	if v, ok := raw["contract_type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "contract_type", Msg: "must be a string"}
		}
		if _, ok := validContractTypes[s]; !ok {
			return &validationError{Field: "contract_type", Msg: fmt.Sprintf("invalid contract type %q", s)}
		}
		p.ContractType = &s
	}
	return nil
}

// --- shared decoders ---

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

func optionalCurrency(raw map[string]json.RawMessage, fallback string) (string, *validationError) {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return fallback, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: "currency", Msg: "must be a string"}
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	if _, ok := validCurrencies[s]; !ok {
		return "", &validationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
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

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
