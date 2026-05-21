package accounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type createRequest struct {
	Name                  string
	Type                  string
	Category              string
	Owner                 string
	Currency              string
	AccountWrapper        *string
	Purpose               string
	SquareMeters          *decimal.Decimal
	ReceivesContributions bool
}

// buildCreateRequest validates the POST body.
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	name, vErr := requireString(raw, "name", "Name cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Name = name

	t, vErr := requireEnumString(raw, "type", validAccountTypes)
	if vErr != nil {
		return r, vErr
	}
	r.Type = t

	cat, vErr := requireEnumString(raw, "category", validCategories)
	if vErr != nil {
		return r, vErr
	}
	r.Category = cat

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Owner = owner

	cur, vErr := optionalString(raw, "currency", "PLN")
	if vErr != nil {
		return r, vErr
	}
	r.Currency = cur

	wrap, vErr := optionalEnumOrNull(raw, "account_wrapper", validWrappers)
	if vErr != nil {
		return r, vErr
	}
	r.AccountWrapper = wrap

	purpose, vErr := requireEnumString(raw, "purpose", validPurposes)
	if vErr != nil {
		return r, vErr
	}
	r.Purpose = purpose

	sq, vErr := optionalDecimal(raw, "square_meters")
	if vErr != nil {
		return r, vErr
	}
	r.SquareMeters = sq

	recv, vErr := optionalBool(raw, "receives_contributions", true)
	if vErr != nil {
		return r, vErr
	}
	r.ReceivesContributions = recv
	return r, nil
}

// buildUpdatePatch reads PUT body.
//
// Field semantics mirror Pydantic's AccountUpdate (all fields Optional):
//   - absent OR explicit null on non-nullable fields -> no change
//     (Python's service uses `if data.X is not None` and skips Nones)
//   - explicit null on account_wrapper or square_meters -> set to NULL
//     (Pydantic's `Wrapper | None` lets None through and the service writes it)
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
	if vErr := patchEnumString(raw, "category", validCategories, &p.Category); vErr != nil {
		return p, vErr
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
	if vErr := patchPlainString(raw, "currency", &p.Currency); vErr != nil {
		return p, vErr
	}
	if vErr := patchEnumString(raw, "purpose", validPurposes, &p.Purpose); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["account_wrapper"]; ok {
		p.AccountWrapperSet = true
		if isNull(v) {
			p.AccountWrapper = nil
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return p, &validationError{Field: "account_wrapper", Msg: "must be a string"}
			}
			if _, ok := validWrappers[s]; !ok {
				return p, &validationError{Field: "account_wrapper", Msg: fmt.Sprintf("invalid wrapper %q", s)}
			}
			p.AccountWrapper = &s
		}
	}
	if v, ok := raw["square_meters"]; ok {
		p.SquareMetersSet = true
		if isNull(v) {
			p.SquareMeters = nil
		} else {
			d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
			if err != nil {
				return p, &validationError{Field: "square_meters", Msg: "must be a number"}
			}
			p.SquareMeters = &d
		}
	}
	if v, ok := raw["receives_contributions"]; ok && !isNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return p, &validationError{Field: "receives_contributions", Msg: "must be a boolean"}
		}
		p.ReceivesContributions = &b
	}
	return p, nil
}

// --- helpers ---

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

func optionalString(raw map[string]json.RawMessage, key, fallback string) (string, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	return s, nil
}

func optionalEnumOrNull(raw map[string]json.RawMessage, key string, allowed map[string]struct{}) (*string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil, &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return nil, &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return &s, nil
}

func optionalDecimal(raw map[string]json.RawMessage, key string) (*decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil, nil
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return nil, &validationError{Field: key, Msg: "must be a number"}
	}
	return &d, nil
}

func optionalBool(raw map[string]json.RawMessage, key string, fallback bool) (bool, *validationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if isNull(v) {
		return false, &validationError{Field: key, Msg: "must be a boolean"}
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, &validationError{Field: key, Msg: "must be a boolean"}
	}
	return b, nil
}

func patchEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, dest **string) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	*dest = &s
	return nil
}

func patchPlainString(raw map[string]json.RawMessage, key string, dest **string) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	*dest = &s
	return nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
