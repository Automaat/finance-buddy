// Package validation holds primitive helpers for parsing request bodies
// that arrive as map[string]json.RawMessage (the PATCH/null-vs-missing
// pattern). Domain packages keep their require*/patch* wrappers so error
// messages remain wire-stable; this package only owns the small parsing
// primitives that were duplicated across every domain.
package validation

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/shopspring/decimal"
)

// IsNull reports whether the raw value is the JSON literal `null`. Used
// to distinguish "missing key" from "explicit null" in PATCH bodies.
func IsNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

// RawString decodes a json.RawMessage as a string. Returns the unmarshal
// error verbatim so callers can map it to whatever domain-specific
// validation message they need.
func RawString(v json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", err
	}
	return s, nil
}

// RawTrimmedString is RawString followed by strings.TrimSpace — the most
// common pattern across domain validators.
func RawTrimmedString(v json.RawMessage) (string, error) {
	s, err := RawString(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

// RawInt decodes a json.RawMessage as an int.
func RawInt(v json.RawMessage) (int, error) {
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, err
	}
	return n, nil
}

// RawDecimal decodes a JSON *number* as a decimal.Decimal. Quoted
// strings ("123.45") are rejected on purpose — callers map the error
// into a "must be a number" validation message, matching the Python
// pydantic behavior the bb-tests pin. Numeric values stay exact —
// never round-trip through float64.
func RawDecimal(v json.RawMessage) (decimal.Decimal, error) {
	return decimal.NewFromString(string(bytes.TrimSpace(v)))
}
