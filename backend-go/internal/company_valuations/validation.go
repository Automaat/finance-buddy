package companyvaluations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// buildCreateRequest is the POST body parser + validator.
//
// JSON numbers are read as raw bytes and converted via decimal.NewFromString
// to preserve Numeric column precision (going through float64 introduces
// IEEE754 rounding that diverges from Python's Decimal(str(...)) round-trip).
// Required fields surface 422 "Field required" when missing — Pydantic
// behavior for the required-field case.
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest

	company, vErr := requireString(raw, "company", "Field required", "Company cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Company = company

	t, vErr := requireDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	currency, vErr := optionalCurrency(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Currency = currency

	fmv, vErr := requireNonNegativeDecimal(raw, "fmv_per_share", "FMV per share must be non-negative")
	if vErr != nil {
		return r, vErr
	}
	r.FMVPerShare = fmv

	r.FMVLow, vErr = optionalNonNegativeDecimal(raw, "fmv_low", "FMV range values must be non-negative")
	if vErr != nil {
		return r, vErr
	}
	r.FMVHigh, vErr = optionalNonNegativeDecimal(raw, "fmv_high", "FMV range values must be non-negative")
	if vErr != nil {
		return r, vErr
	}
	if r.FMVLow != nil && r.FMVLow.GreaterThan(r.FMVPerShare) {
		return r, &validationError{Field: "fmv_low", Msg: "fmv_low cannot exceed fmv_per_share"}
	}
	if r.FMVHigh != nil && r.FMVHigh.LessThan(r.FMVPerShare) {
		return r, &validationError{Field: "fmv_high", Msg: "fmv_high cannot be below fmv_per_share"}
	}

	r.CommonStockDiscountPct, vErr = optionalDiscountPct(raw)
	if vErr != nil {
		return r, vErr
	}

	source, vErr := requireString(raw, "source", "Field required", "Source cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	if _, ok := validSources[source]; !ok {
		return r, &validationError{Field: "source", Msg: fmt.Sprintf("invalid source %q", source)}
	}
	r.Source = source

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

// buildUpdatePatch follows Python's GoalUpdate convention: a field set to
// null is treated as "no-op", matching how Pydantic's nullable validators
// behave on None input.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if vErr := patchStrings(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchNumbers(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

func patchStrings(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
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
	if v, ok := raw["currency"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "currency", Msg: "must be a string"}
		}
		s = strings.ToUpper(strings.TrimSpace(s))
		if _, valid := validCurrencies[s]; !valid {
			return &validationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
		}
		p.Currency = &s
	}
	if v, ok := raw["source"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "source", Msg: "must be a string"}
		}
		if _, valid := validSources[s]; !valid {
			return &validationError{Field: "source", Msg: fmt.Sprintf("invalid source %q", s)}
		}
		p.Source = &s
	}
	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	if v, ok := raw["date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		p.Date = &t
	}
	return nil
}

func patchNumbers(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	for _, f := range [...]struct {
		key, field, msg string
		dest            **decimal.Decimal
	}{
		{"fmv_per_share", "fmv_per_share", "FMV values must be non-negative", &p.FMVPerShare},
		{"fmv_low", "fmv_low", "FMV values must be non-negative", &p.FMVLow},
		{"fmv_high", "fmv_high", "FMV values must be non-negative", &p.FMVHigh},
	} {
		d, vErr := optionalNonNegativeDecimal(raw, f.key, f.msg)
		if vErr != nil {
			vErr.Field = f.field
			return vErr
		}
		if d != nil {
			*f.dest = d
		}
	}
	p.CommonStockDiscountPct, _ = optionalDiscountPct(raw)
	// Re-walk to surface the validation error if any — optionalDiscountPct only
	// returns an error for invalid present values; the nil branch is benign.
	if _, vErr := optionalDiscountPct(raw); vErr != nil {
		return vErr
	}
	return nil
}

// --- shared decoders ---

func requireString(raw map[string]json.RawMessage, key, missingMsg, emptyMsg string) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: missingMsg}
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

func optionalCurrency(raw map[string]json.RawMessage) (string, *validationError) {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return "USD", nil // Python default
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

// requireNonNegativeDecimal parses a required JSON number as a Decimal using
// the raw token (no float64 intermediate) so Numeric column precision is
// preserved end-to-end.
func requireNonNegativeDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "Field required"}
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "must be a number"}
	}
	if d.IsNegative() {
		return decimal.Decimal{}, &validationError{Field: key, Msg: msg}
	}
	return d, nil
}

func optionalNonNegativeDecimal(raw map[string]json.RawMessage, key, msg string) (*decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil, nil
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return nil, &validationError{Field: key, Msg: "must be a number"}
	}
	if d.IsNegative() {
		return nil, &validationError{Field: key, Msg: msg}
	}
	return &d, nil
}

func optionalDiscountPct(raw map[string]json.RawMessage) (*decimal.Decimal, *validationError) {
	v, ok := raw["common_stock_discount_pct"]
	if !ok || isNull(v) {
		return nil, nil
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return nil, &validationError{Field: "common_stock_discount_pct", Msg: "must be a number"}
	}
	lower := decimal.Zero
	upper := decimal.NewFromInt(100)
	if d.LessThan(lower) || d.GreaterThan(upper) {
		return nil, &validationError{
			Field: "common_stock_discount_pct",
			Msg:   "Common-stock discount must be between 0 and 100",
		}
	}
	return &d, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
