package validation

import (
	"encoding/json"
	"fmt"
	"strings"
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

// OptionalDate reads an optional YYYY-MM-DD field. Missing or explicit null
// returns nil.
func OptionalDate(raw map[string]json.RawMessage, key string) (*time.Time, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	t, err := RawDate(v)
	if err != nil {
		if IsRawDateFormatError(err) {
			return nil, &httputil.ValidationError{Field: key, Msg: "must be YYYY-MM-DD"}
		}
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	return &t, nil
}

// RequiredDateNotFuture reads a required YYYY-MM-DD field and rejects dates
// after the current UTC day returned by now.
func RequiredDateNotFuture(
	raw map[string]json.RawMessage,
	key string,
	now func() time.Time,
) (time.Time, *httputil.ValidationError) {
	return RequiredDateNotFutureWithMessage(raw, key, now, "Date cannot be in the future")
}

// RequiredDateNotFutureWithMessage reads a required YYYY-MM-DD field and
// rejects dates after the current UTC day returned by now with futureMsg.
func RequiredDateNotFutureWithMessage(
	raw map[string]json.RawMessage,
	key string,
	now func() time.Time,
	futureMsg string,
) (time.Time, *httputil.ValidationError) {
	t, vErr := RequiredDate(raw, key)
	if vErr != nil {
		return time.Time{}, vErr
	}
	if t.After(todayUTC(now)) {
		return time.Time{}, &httputil.ValidationError{Field: key, Msg: futureMsg}
	}
	return t, nil
}

// OptionalDateNotFuture reads an optional YYYY-MM-DD field and rejects dates
// after the current UTC day returned by now with futureMsg.
func OptionalDateNotFuture(
	raw map[string]json.RawMessage,
	key string,
	now func() time.Time,
	futureMsg string,
) (*time.Time, *httputil.ValidationError) {
	t, vErr := OptionalDate(raw, key)
	if vErr != nil {
		return nil, vErr
	}
	if t == nil {
		return nil, nil
	}
	if t.After(todayUTC(now)) {
		return nil, &httputil.ValidationError{Field: key, Msg: futureMsg}
	}
	return t, nil
}

// OptionalBool reads an optional boolean field. Missing and explicit null
// return nil.
func OptionalBool(raw map[string]json.RawMessage, key string) (*bool, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a boolean"}
	}
	return &b, nil
}

// OptionalBoolDefault is the default-valued variant of OptionalBool. Missing
// and explicit null return fallback.
func OptionalBoolDefault(raw map[string]json.RawMessage, key string, fallback bool) (bool, *httputil.ValidationError) {
	b, vErr := OptionalBool(raw, key)
	if vErr != nil {
		return false, vErr
	}
	if b == nil {
		return fallback, nil
	}
	return *b, nil
}

func todayUTC(now func() time.Time) time.Time {
	if now == nil {
		now = time.Now
	}
	today := now().UTC()
	return time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
}

// OptionalInt reads an optional integer field. Missing or explicit null
// returns nil.
func OptionalInt(
	raw map[string]json.RawMessage,
	key string,
	typeMsg string,
) (*int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	n, err := RawInt(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: typeMsg}
	}
	return &n, nil
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

// RequiredInt reads a required integer field and returns the supplied messages
// so callers can preserve existing endpoint wire contracts.
func RequiredInt(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
	typeMsg string,
) (int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: missingMsg}
	}
	n, err := RawInt(v)
	if err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: typeMsg}
	}
	return n, nil
}

// RequiredIntRange reads a required integer field and validates it falls
// within the inclusive [lo, hi] range.
func RequiredIntRange(
	raw map[string]json.RawMessage,
	key string,
	lo int,
	hi int,
	rangeMsg string,
) (int, *httputil.ValidationError) {
	n, vErr := RequiredInt(raw, key, "Field required", "must be an integer")
	if vErr != nil {
		return 0, vErr
	}
	if n < lo || n > hi {
		return 0, &httputil.ValidationError{Field: key, Msg: rangeMsg}
	}
	return n, nil
}

// RequiredPositiveInt reads a required integer field and rejects zero or
// negative values with the supplied positiveMsg.
func RequiredPositiveInt(
	raw map[string]json.RawMessage,
	key string,
	positiveMsg string,
) (int, *httputil.ValidationError) {
	n, vErr := RequiredInt(raw, key, "Field required", "must be an integer")
	if vErr != nil {
		return 0, vErr
	}
	if n <= 0 {
		return 0, &httputil.ValidationError{Field: key, Msg: positiveMsg}
	}
	return n, nil
}

// OptionalPositiveInt reads an optional integer field and rejects zero or
// negative values with the supplied positiveMsg.
func OptionalPositiveInt(
	raw map[string]json.RawMessage,
	key string,
	positiveMsg string,
) (*int, *httputil.ValidationError) {
	n, vErr := OptionalInt(raw, key, "must be an integer")
	if vErr != nil {
		return nil, vErr
	}
	if n != nil && *n <= 0 {
		return nil, &httputil.ValidationError{Field: key, Msg: positiveMsg}
	}
	return n, nil
}

// OptionalNonNegativeInt reads an optional integer field and rejects negative
// values with the supplied nonNegativeMsg.
func OptionalNonNegativeInt(
	raw map[string]json.RawMessage,
	key string,
	nonNegativeMsg string,
) (*int, *httputil.ValidationError) {
	n, vErr := OptionalInt(raw, key, "must be an integer")
	if vErr != nil {
		return nil, vErr
	}
	if n != nil && *n < 0 {
		return nil, &httputil.ValidationError{Field: key, Msg: nonNegativeMsg}
	}
	return n, nil
}

// OptionalNonNegativeIntDefault is the default-valued variant of
// OptionalNonNegativeInt. Missing and explicit null return fallback.
func OptionalNonNegativeIntDefault(
	raw map[string]json.RawMessage,
	key string,
	nonNegativeMsg string,
	fallback int,
) (int, *httputil.ValidationError) {
	n, vErr := OptionalNonNegativeInt(raw, key, nonNegativeMsg)
	if vErr != nil {
		return 0, vErr
	}
	if n == nil {
		return fallback, nil
	}
	return *n, nil
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

// OptionalEnumString reads an optional string field and checks it belongs to
// the provided allowed set. Missing and explicit null return fallback.
func OptionalEnumString(
	raw map[string]json.RawMessage,
	key string,
	allowed map[string]struct{},
	fallback string,
) (string, *httputil.ValidationError) {
	s, vErr := OptionalEnumStringPtr(raw, key, allowed)
	if vErr != nil {
		return "", vErr
	}
	if s == nil {
		return fallback, nil
	}
	return *s, nil
}

// OptionalEnumStringPtr is the sparse-update variant of OptionalEnumString.
// Missing and explicit null return nil.
func OptionalEnumStringPtr(
	raw map[string]json.RawMessage,
	key string,
	allowed map[string]struct{},
) (*string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	s, err := RawString(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return nil, &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return &s, nil
}

// OptionalUpperTrimmedEnumString reads an optional string field, trims
// whitespace, uppercases it, and checks it belongs to the provided allowed set.
// Missing and explicit null return fallback.
func OptionalUpperTrimmedEnumString(
	raw map[string]json.RawMessage,
	key string,
	allowed map[string]struct{},
	fallback string,
	invalidMsg string,
) (string, *httputil.ValidationError) {
	s, vErr := OptionalUpperTrimmedEnumStringPtr(raw, key, allowed, invalidMsg)
	if vErr != nil {
		return "", vErr
	}
	if s == nil {
		return fallback, nil
	}
	return *s, nil
}

// OptionalUpperTrimmedEnumStringPtr is the sparse-update variant of
// OptionalUpperTrimmedEnumString. Missing and explicit null return nil.
func OptionalUpperTrimmedEnumStringPtr(
	raw map[string]json.RawMessage,
	key string,
	allowed map[string]struct{},
	invalidMsg string,
) (*string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || IsNull(v) {
		return nil, nil
	}
	s, err := RawString(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	if _, ok := allowed[s]; !ok {
		return nil, &httputil.ValidationError{Field: key, Msg: invalidMsg}
	}
	return &s, nil
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

// RequiredNonNegativeDecimal reads a required JSON number field and rejects
// negative values with the supplied nonNegativeMsg.
func RequiredNonNegativeDecimal(
	raw map[string]json.RawMessage,
	key string,
	missingMsg string,
	nonNegativeMsg string,
) (decimal.Decimal, *httputil.ValidationError) {
	d, vErr := RequiredDecimal(raw, key, missingMsg)
	if vErr != nil {
		return decimal.Zero, vErr
	}
	if d.IsNegative() {
		return decimal.Zero, &httputil.ValidationError{Field: key, Msg: nonNegativeMsg}
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

// OptionalPositiveDecimal reads an optional JSON number field and rejects zero
// or negative values with the supplied positiveMsg.
func OptionalPositiveDecimal(
	raw map[string]json.RawMessage,
	key string,
	positiveMsg string,
) (*decimal.Decimal, *httputil.ValidationError) {
	d, vErr := OptionalDecimal(raw, key)
	if vErr != nil {
		return nil, vErr
	}
	if d != nil && !d.IsPositive() {
		return nil, &httputil.ValidationError{Field: key, Msg: positiveMsg}
	}
	return d, nil
}

// OptionalNonNegativeDecimal reads an optional JSON number field and rejects
// negative values with the supplied nonNegativeMsg.
func OptionalNonNegativeDecimal(
	raw map[string]json.RawMessage,
	key string,
	nonNegativeMsg string,
) (*decimal.Decimal, *httputil.ValidationError) {
	d, vErr := OptionalDecimal(raw, key)
	if vErr != nil {
		return nil, vErr
	}
	if d != nil && d.IsNegative() {
		return nil, &httputil.ValidationError{Field: key, Msg: nonNegativeMsg}
	}
	return d, nil
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
