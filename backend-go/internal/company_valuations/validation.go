package companyvaluations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func validateCreate(req *createRequest) *validationError {
	company := strings.TrimSpace(req.Company)
	if company == "" {
		return &validationError{Field: "company", Msg: "Company cannot be empty"}
	}
	req.Company = company

	if time.Time(req.Date).IsZero() {
		return &validationError{Field: "date", Msg: "Field required"}
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}
	if _, ok := validCurrencies[currency]; !ok {
		return &validationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
	}
	req.Currency = currency

	if req.FMVPerShare < 0 {
		return &validationError{Field: "fmv_per_share", Msg: "FMV per share must be non-negative"}
	}
	if req.FMVLow != nil && *req.FMVLow < 0 {
		return &validationError{Field: "fmv_low", Msg: "FMV range values must be non-negative"}
	}
	if req.FMVHigh != nil && *req.FMVHigh < 0 {
		return &validationError{Field: "fmv_high", Msg: "FMV range values must be non-negative"}
	}
	if req.FMVLow != nil && *req.FMVLow > req.FMVPerShare {
		return &validationError{Field: "fmv_low", Msg: "fmv_low cannot exceed fmv_per_share"}
	}
	if req.FMVHigh != nil && *req.FMVHigh < req.FMVPerShare {
		return &validationError{Field: "fmv_high", Msg: "fmv_high cannot be below fmv_per_share"}
	}
	if req.CommonStockDiscountPct != nil {
		d := *req.CommonStockDiscountPct
		if d < 0 || d > 100 {
			return &validationError{
				Field: "common_stock_discount_pct",
				Msg:   "Common-stock discount must be between 0 and 100",
			}
		}
	}
	if _, ok := validSources[req.Source]; !ok {
		return &validationError{Field: "source", Msg: fmt.Sprintf("invalid source %q", req.Source)}
	}
	return nil
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
		if err := decodeNonNegativeDecimal(raw, f.key, f.field, f.msg, f.dest); err != nil {
			return err
		}
	}
	if v, ok := raw["common_stock_discount_pct"]; ok && !isNull(v) {
		var x float64
		if err := json.Unmarshal(v, &x); err != nil {
			return &validationError{Field: "common_stock_discount_pct", Msg: "must be a number"}
		}
		if x < 0 || x > 100 {
			return &validationError{
				Field: "common_stock_discount_pct",
				Msg:   "Common-stock discount must be between 0 and 100",
			}
		}
		d := decimal.NewFromFloat(x)
		p.CommonStockDiscountPct = &d
	}
	return nil
}

func decodeNonNegativeDecimal(
	raw map[string]json.RawMessage, key, field, msg string, dest **decimal.Decimal,
) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &validationError{Field: field, Msg: "must be a number"}
	}
	if f < 0 {
		return &validationError{Field: field, Msg: msg}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
