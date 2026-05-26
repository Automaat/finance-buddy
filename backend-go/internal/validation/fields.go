package validation

import (
	"encoding/json"
	"time"

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
