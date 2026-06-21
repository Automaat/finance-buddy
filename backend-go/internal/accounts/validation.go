package accounts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

type createRequest struct {
	Name                  string
	Type                  string
	Category              string
	OwnerUserID           *int
	Currency              string
	AccountWrapper        *string
	Purpose               string
	SquareMeters          *decimal.Decimal
	ReceivesContributions bool
	ExcludedFromFire      bool
	InterestRatePct       *decimal.Decimal
}

// buildCreateRequest validates the POST body.
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest
	name, vErr := validation.RequiredTrimmedString(raw, "name", "Field required", "Name cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Name = name

	t, vErr := validation.RequiredEnumString(raw, "type", validAccountTypes)
	if vErr != nil {
		return r, vErr
	}
	r.Type = t

	cat, vErr := validation.RequiredEnumString(raw, "category", validCategories)
	if vErr != nil {
		return r, vErr
	}
	r.Category = cat

	ownerID, vErr := validation.RequiredIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = ownerID

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

	purpose, vErr := validation.RequiredEnumString(raw, "purpose", validPurposes)
	if vErr != nil {
		return r, vErr
	}
	r.Purpose = purpose

	sq, vErr := optionalDecimal(raw, "square_meters")
	if vErr != nil {
		return r, vErr
	}
	r.SquareMeters = sq

	recv, vErr := optionalBoolDefaultTrue(raw, "receives_contributions")
	if vErr != nil {
		return r, vErr
	}
	r.ReceivesContributions = recv

	excl, vErr := optionalBoolDefaultFalse(raw, "excluded_from_fire")
	if vErr != nil {
		return r, vErr
	}
	r.ExcludedFromFire = excl

	rate, vErr := optionalInterestRatePct(raw)
	if vErr != nil {
		return r, vErr
	}
	r.InterestRatePct = rate
	return r, nil
}

// buildUpdatePatch reads PUT body.
//
// Field semantics mirror Pydantic's AccountUpdate (all fields Optional):
//   - absent OR explicit null on non-nullable fields -> no change
//     (Python's service uses `if data.X is not None` and skips Nones)
//   - explicit null on account_wrapper or square_meters -> set to NULL
//     (Pydantic's `Wrapper | None` lets None through and the service writes it)
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
	if vErr := patchEnumString(raw, "category", validCategories, &p.Category); vErr != nil {
		return p, vErr
	}
	if _, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		ownerID, vErr := validation.OptionalInt(raw, "owner_user_id", "must be an integer")
		if vErr != nil {
			return p, vErr
		}
		p.OwnerUserID = ownerID
	}
	if vErr := patchPlainString(raw, "currency", &p.Currency); vErr != nil {
		return p, vErr
	}
	if vErr := patchEnumString(raw, "purpose", validPurposes, &p.Purpose); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["account_wrapper"]; ok {
		p.AccountWrapperSet = true
		if validation.IsNull(v) {
			p.AccountWrapper = nil
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return p, &httputil.ValidationError{Field: "account_wrapper", Msg: "must be a string"}
			}
			if _, ok := validWrappers[s]; !ok {
				return p, &httputil.ValidationError{Field: "account_wrapper", Msg: fmt.Sprintf("invalid wrapper %q", s)}
			}
			p.AccountWrapper = &s
		}
	}
	if v, ok := raw["square_meters"]; ok {
		p.SquareMetersSet = true
		if validation.IsNull(v) {
			p.SquareMeters = nil
		} else {
			d, err := validation.RawDecimal(v)
			if err != nil {
				return p, &httputil.ValidationError{Field: "square_meters", Msg: "must be a number"}
			}
			p.SquareMeters = &d
		}
	}
	if v, ok := raw["receives_contributions"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return p, &httputil.ValidationError{Field: "receives_contributions", Msg: "must be a boolean"}
		}
		p.ReceivesContributions = &b
	}
	if v, ok := raw["excluded_from_fire"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return p, &httputil.ValidationError{Field: "excluded_from_fire", Msg: "must be a boolean"}
		}
		p.ExcludedFromFire = &b
	}
	if v, ok := raw["interest_rate_pct"]; ok {
		p.InterestRatePctSet = true
		if validation.IsNull(v) {
			p.InterestRatePct = nil
		} else {
			d, vErr := parseInterestRate(v)
			if vErr != nil {
				return p, vErr
			}
			p.InterestRatePct = &d
		}
	}
	return p, nil
}

// optionalInterestRatePct parses an optional interest_rate_pct on create.
// Absent or explicit null -> nil. Out-of-range or non-numeric -> validation
// error. The same range gate runs on update (patch) via parseInterestRate.
func optionalInterestRatePct(raw map[string]json.RawMessage) (*decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw["interest_rate_pct"]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	d, vErr := parseInterestRate(v)
	if vErr != nil {
		return nil, vErr
	}
	return &d, nil
}

// interestRateMaxPct caps the accepted nominal-yield input at 50%. The widget
// reports real yield after CPI + Belka and is meant for cash-like holdings
// (savings, EDO/COI/ROR); higher inputs almost always mean the user typed a
// decimal as basis points (e.g. 535 for 5.35%), so we surface a validation
// error instead of silently rendering a misleading number.
const interestRateMaxPct = 50

func parseInterestRate(v json.RawMessage) (decimal.Decimal, *httputil.ValidationError) {
	d, err := validation.RawDecimal(v)
	if err != nil {
		return decimal.Zero, &httputil.ValidationError{Field: "interest_rate_pct", Msg: "must be a number"}
	}
	if d.IsNegative() || d.GreaterThan(decimal.NewFromInt(interestRateMaxPct)) {
		return decimal.Zero, &httputil.ValidationError{
			Field: "interest_rate_pct",
			Msg:   fmt.Sprintf("Interest rate must be between 0 and %d", interestRateMaxPct),
		}
	}
	return d, nil
}

// --- helpers ---

func optionalString(raw map[string]json.RawMessage, key, fallback string) (string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok {
		return fallback, nil
	}
	if validation.IsNull(v) {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	return s, nil
}

func optionalEnumOrNull(raw map[string]json.RawMessage, key string, allowed map[string]struct{}) (*string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return nil, &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return &s, nil
}

func optionalDecimal(raw map[string]json.RawMessage, key string) (*decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	d, err := validation.RawDecimal(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return &d, nil
}

func optionalBoolDefaultTrue(raw map[string]json.RawMessage, key string) (bool, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok {
		return true, nil
	}
	if validation.IsNull(v) {
		return false, &httputil.ValidationError{Field: key, Msg: "must be a boolean"}
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, &httputil.ValidationError{Field: key, Msg: "must be a boolean"}
	}
	return b, nil
}

func optionalBoolDefaultFalse(raw map[string]json.RawMessage, key string) (bool, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return false, nil
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, &httputil.ValidationError{Field: key, Msg: "must be a boolean"}
	}
	return b, nil
}

func patchEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, dest **string) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	*dest = &s
	return nil
}

func patchPlainString(raw map[string]json.RawMessage, key string, dest **string) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	*dest = &s
	return nil
}
