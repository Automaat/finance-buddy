package companyvaluations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// buildCreateRequest is the POST body parser + validator.
//
// JSON numbers are read as raw bytes and converted via decimal.NewFromString
// to preserve Numeric column precision (going through float64 introduces
// IEEE754 rounding that diverges from Python's Decimal(str(...)) round-trip).
// Required fields surface 422 "Field required" when missing — Pydantic
// behavior for the required-field case.
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest

	company, vErr := validation.RequiredTrimmedString(raw, "company", "Field required", "Company cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Company = company

	t, vErr := validation.RequiredDate(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	currency, vErr := optionalCurrency(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Currency = currency

	fmv, vErr := validation.RequiredNonNegativeDecimal(
		raw,
		"fmv_per_share",
		"Field required",
		"FMV per share must be non-negative",
	)
	if vErr != nil {
		return r, vErr
	}
	r.FMVPerShare = fmv

	r.FMVLow, vErr = validation.OptionalNonNegativeDecimal(raw, "fmv_low", "FMV range values must be non-negative")
	if vErr != nil {
		return r, vErr
	}
	r.FMVHigh, vErr = validation.OptionalNonNegativeDecimal(raw, "fmv_high", "FMV range values must be non-negative")
	if vErr != nil {
		return r, vErr
	}
	if r.FMVLow != nil && r.FMVLow.GreaterThan(r.FMVPerShare) {
		return r, &httputil.ValidationError{Field: "fmv_low", Msg: "fmv_low cannot exceed fmv_per_share"}
	}
	if r.FMVHigh != nil && r.FMVHigh.LessThan(r.FMVPerShare) {
		return r, &httputil.ValidationError{Field: "fmv_high", Msg: "fmv_high cannot be below fmv_per_share"}
	}

	r.CommonStockDiscountPct, vErr = optionalDiscountPct(raw)
	if vErr != nil {
		return r, vErr
	}

	source, vErr := validation.RequiredTrimmedString(raw, "source", "Field required", "Source cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	if _, ok := validSources[source]; !ok {
		return r, &httputil.ValidationError{Field: "source", Msg: fmt.Sprintf("invalid source %q", source)}
	}
	r.Source = source

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

// buildUpdatePatch follows Python's GoalUpdate convention: a field set to
// null is treated as "no-op", matching how Pydantic's nullable validators
// behave on None input.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if vErr := patchStrings(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchNumbers(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

func patchStrings(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
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
	if v, ok := raw["currency"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
		}
		s = strings.ToUpper(strings.TrimSpace(s))
		if _, valid := validCurrencies[s]; !valid {
			return &httputil.ValidationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
		}
		p.Currency = &s
	}
	if v, ok := raw["source"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "source", Msg: "must be a string"}
		}
		if _, valid := validSources[s]; !valid {
			return &httputil.ValidationError{Field: "source", Msg: fmt.Sprintf("invalid source %q", s)}
		}
		p.Source = &s
	}
	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	if v, ok := raw["date"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
		}
		p.Date = &t
	}
	return nil
}

func patchNumbers(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	for _, f := range [...]struct {
		key, field, msg string
		dest            **decimal.Decimal
	}{
		{"fmv_per_share", "fmv_per_share", "FMV values must be non-negative", &p.FMVPerShare},
		{"fmv_low", "fmv_low", "FMV values must be non-negative", &p.FMVLow},
		{"fmv_high", "fmv_high", "FMV values must be non-negative", &p.FMVHigh},
	} {
		d, vErr := validation.OptionalNonNegativeDecimal(raw, f.key, f.msg)
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

func optionalCurrency(raw map[string]json.RawMessage) (string, *httputil.ValidationError) {
	v, ok := raw["currency"]
	if !ok || validation.IsNull(v) {
		return "USD", nil // Python default
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

func optionalDiscountPct(raw map[string]json.RawMessage) (*decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw["common_stock_discount_pct"]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	d, err := validation.RawDecimal(v)
	if err != nil {
		return nil, &httputil.ValidationError{Field: "common_stock_discount_pct", Msg: "must be a number"}
	}
	lower := decimal.Zero
	upper := decimal.NewFromInt(100)
	if d.LessThan(lower) || d.GreaterThan(upper) {
		return nil, &httputil.ValidationError{
			Field: "common_stock_discount_pct",
			Msg:   "Common-stock discount must be between 0 and 100",
		}
	}
	return &d, nil
}
