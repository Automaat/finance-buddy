package validation

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// RequiredTrimmedString reads a required string field, trims whitespace, and
// returns the supplied missing/empty messages so domain wire contracts stay
// unchanged while callers share parsing logic.
func RequiredTrimmedString(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
	emptyMsg string,
) (string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return "", &httputil.ValidationError{Field: key, Msg: missingMsg}
	}
	s, err := RawTrimmedString(v)
	if err != nil {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if s == "" {
		return "", &httputil.ValidationError{Field: key, Msg: emptyMsg}
	}
	return s, nil
}

// OptionalTrimmedString reads an optional string field. Missing or explicit
// null returns nil; empty string is a validation error.
func OptionalTrimmedString(
	raw map[string]json.RawMessage,
	key string,
	emptyMsg string,
) (*string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	s, err := RawTrimmedString(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if s == "" {
		return nil, &httputil.ValidationError{Field: key, Msg: emptyMsg}
	}
	return &s, nil
}

// RequiredDate reads a required YYYY-MM-DD field.
func RequiredDate(raw map[string]json.RawMessage, key string) (time.Time, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	t, err := RawDate(v)
	if err != nil {
		if IsRawDateFormatError(err) {
			return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be YYYY-MM-DD"}
		}
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	return t, nil
}

// RequiredDateNotFuture reads a required YYYY-MM-DD field and rejects dates
// after the current UTC day returned by now.
func RequiredDateNotFuture(
	raw map[string]json.RawMessage,
	key string,
	now func() time.Time,
) (time.Time, *httputil.ValidationError) {
	t, vErr := RequiredDate(raw, key)
	if vErr != nil {
		return time.Time{}, vErr
	}
	if now == nil {
		now = time.Now
	}
	today := now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if t.After(today) {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: "Date cannot be in the future"}
	}
	return t, nil
}

// RequiredIntOrNull reads a required integer field where explicit null is
// allowed and maps to nil.
func RequiredIntOrNull(raw map[string]json.RawMessage, key string) (*int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok {
		return nil, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	if IsNull(v) {
		return nil, nil
	}
	n, err := RawInt(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	return &n, nil
}

// RequiredEnumString reads a required string field and checks it belongs to
// the provided allowed set.
func RequiredEnumString(
	raw map[string]json.RawMessage,
	key string,
	allowed map[string]struct{},
) (string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return "", &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	s, err := RawString(v)
	if err != nil {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

// RequiredDecimal reads a required JSON number field. Quoted numeric strings
// are rejected to preserve the backend's pydantic-compatible wire contract.
func RequiredDecimal(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
) (decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: missingMsg}
	}
	d, err := RawDecimal(v)
	if err != nil {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return d, nil
}

// RequiredPositiveDecimal reads a required JSON number field and rejects zero
// or negative values with the supplied positiveMsg.
func RequiredPositiveDecimal(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
	positiveMsg string,
) (decimal.Decimal, *httputil.ValidationError) {
	d, vErr := RequiredDecimal(raw, key, missingMsg)
	if vErr != nil {
		return decimal.Zero, vErr
	}
	if !d.IsPositive() {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: positiveMsg}
	}
	return d, nil
}

// OptionalDecimal reads an optional JSON number field. Missing and explicit
// null are treated the same; quoted numeric strings are rejected.
func OptionalDecimal(
	raw map[string]json.RawMessage,
	key string,
) (*decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	d, err := RawDecimal(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return &d, nil
}

// RequiredDecimalStringOrNumber reads a required decimal field that may arrive
// as a JSON number or quoted numeric string. This preserves compatibility for
// endpoints backed by string-valued frontend form controls.
func RequiredDecimalStringOrNumber(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
) (decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: missingMsg}
	}
	d, err := RawDecimalStringOrNumber(v)
	if err != nil {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return d, nil
}

// OptionalDecimalStringOrNumber reads an optional decimal field that may
// arrive as a JSON number or quoted numeric string.
func OptionalDecimalStringOrNumber(
	raw map[string]json.RawMessage,
	key string,
) (*decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	d, err := RawDecimalStringOrNumber(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	return &d, nil
}
