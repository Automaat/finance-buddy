package holdings

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

func buildSecurityInput(raw map[string]json.RawMessage) (Security, *httputil.ValidationError) {
	var s Security
	s.Currency = "PLN"
	if v, ok := raw["symbol"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Symbol); err != nil {
			return s, &httputil.ValidationError{Field: "symbol", Msg: "must be a string"}
		}
	}
	if s.Symbol = strings.TrimSpace(s.Symbol); s.Symbol == "" {
		return s, &httputil.ValidationError{Field: "symbol", Msg: "required"}
	}
	if len(s.Symbol) > 32 {
		return s, &httputil.ValidationError{Field: "symbol", Msg: "max 32 chars"}
	}
	if v, ok := raw["isin"]; ok && string(v) != "null" {
		var isin string
		if err := json.Unmarshal(v, &isin); err != nil {
			return s, &httputil.ValidationError{Field: "isin", Msg: "must be a string"}
		}
		if isin = strings.TrimSpace(isin); isin != "" {
			if len(isin) != 12 {
				return s, &httputil.ValidationError{Field: "isin", Msg: "must be 12 chars"}
			}
			s.ISIN = &isin
		}
	}
	if v, ok := raw["name"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Name); err != nil {
			return s, &httputil.ValidationError{Field: "name", Msg: "must be a string"}
		}
	}
	if s.Name = strings.TrimSpace(s.Name); s.Name == "" {
		return s, &httputil.ValidationError{Field: "name", Msg: "required"}
	}
	if len(s.Name) > 200 {
		return s, &httputil.ValidationError{Field: "name", Msg: "max 200 chars"}
	}
	if v, ok := raw["asset_type"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.AssetType); err != nil {
			return s, &httputil.ValidationError{Field: "asset_type", Msg: "must be a string"}
		}
	}
	if !validAssetType(s.AssetType) {
		return s, &httputil.ValidationError{Field: "asset_type", Msg: "must be one of stock|etf|bond|fund"}
	}
	if v, ok := raw["currency"]; ok && string(v) != "null" {
		var c string
		if err := json.Unmarshal(v, &c); err != nil {
			return s, &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
		}
		if c = strings.TrimSpace(c); c != "" {
			if len(c) != 3 {
				return s, &httputil.ValidationError{Field: "currency", Msg: "must be 3 chars"}
			}
			s.Currency = strings.ToUpper(c)
		}
	}
	return s, nil
}

func validAssetType(t string) bool {
	switch t {
	case "stock", "etf", "bond", "fund":
		return true
	}
	return false
}

func buildLotInput(raw map[string]json.RawMessage) (Lot, *httputil.ValidationError) {
	var l Lot
	accountID, vErr := validation.RequiredInt(raw, "account_id", "required", "must be integer")
	if vErr != nil {
		return l, vErr
	}
	l.AccountID = accountID
	if l.AccountID <= 0 {
		return l, &httputil.ValidationError{Field: "account_id", Msg: "must be positive"}
	}
	securityID, vErr := validation.RequiredInt(raw, "security_id", "required", "must be integer")
	if vErr != nil {
		return l, vErr
	}
	l.SecurityID = securityID
	if l.SecurityID <= 0 {
		return l, &httputil.ValidationError{Field: "security_id", Msg: "must be positive"}
	}
	sideStr, vErr := validation.RequiredString(raw, "side", "required", "cannot be empty")
	if vErr != nil {
		return l, vErr
	}
	if !IsValidSide(sideStr) {
		return l, &httputil.ValidationError{Field: "side", Msg: "must be buy or sell"}
	}
	l.Side = Side(sideStr)
	qty, vErr := requireDecimal(raw, "quantity")
	if vErr != nil {
		return l, vErr
	}
	if qty.Sign() <= 0 {
		return l, &httputil.ValidationError{Field: "quantity", Msg: "must be positive"}
	}
	l.Quantity = qty
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return l, vErr
	}
	if price.Sign() < 0 {
		return l, &httputil.ValidationError{Field: "price", Msg: "must not be negative"}
	}
	l.Price = price
	if fee, vErr := validation.OptionalDecimalStringOrNumber(raw, "fee"); vErr != nil {
		return l, vErr
	} else if fee != nil {
		if fee.Sign() < 0 {
			return l, &httputil.ValidationError{Field: "fee", Msg: "must not be negative"}
		}
		l.Fee = *fee
	}
	dateStr, vErr := validation.RequiredString(raw, "date", "required", "cannot be empty")
	if vErr != nil {
		return l, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return l, &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	l.Date = parsed
	return l, nil
}

func buildQuoteInput(raw map[string]json.RawMessage, securityID int) (PriceQuote, *httputil.ValidationError) {
	q := PriceQuote{SecurityID: securityID, Source: "manual"}
	dateStr, vErr := validation.RequiredString(raw, "date", "required", "cannot be empty")
	if vErr != nil {
		return q, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return q, &httputil.ValidationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	q.Date = parsed
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return q, vErr
	}
	if price.Sign() < 0 {
		return q, &httputil.ValidationError{Field: "price", Msg: "must not be negative"}
	}
	q.Price = price
	if v, ok := raw["source"]; ok && string(v) != "null" {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return q, &httputil.ValidationError{Field: "source", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			if len(s) > 40 {
				return q, &httputil.ValidationError{Field: "source", Msg: "max 40 chars"}
			}
			q.Source = s
		}
	}
	return q, nil
}

func buildDividendInput(raw map[string]json.RawMessage) (Dividend, *httputil.ValidationError) {
	var d Dividend
	d.Currency = "PLN"
	accountID, vErr := validation.RequiredInt(raw, "account_id", "required", "must be integer")
	if vErr != nil {
		return d, vErr
	}
	d.AccountID = accountID
	if d.AccountID <= 0 {
		return d, &httputil.ValidationError{Field: "account_id", Msg: "must be positive"}
	}
	securityID, vErr := validation.RequiredInt(raw, "security_id", "required", "must be integer")
	if vErr != nil {
		return d, vErr
	}
	d.SecurityID = securityID
	if d.SecurityID <= 0 {
		return d, &httputil.ValidationError{Field: "security_id", Msg: "must be positive"}
	}
	dateStr, vErr := validation.RequiredString(raw, "pay_date", "required", "cannot be empty")
	if vErr != nil {
		return d, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return d, &httputil.ValidationError{Field: "pay_date", Msg: "must be YYYY-MM-DD"}
	}
	d.PayDate = parsed
	gross, vErr := requireDecimal(raw, "gross_amount")
	if vErr != nil {
		return d, vErr
	}
	if gross.Sign() <= 0 {
		return d, &httputil.ValidationError{Field: "gross_amount", Msg: "must be positive"}
	}
	d.GrossAmount = gross
	if tax, vErr := validation.OptionalDecimalStringOrNumber(raw, "withholding_tax"); vErr != nil {
		return d, vErr
	} else if tax != nil {
		if tax.Sign() < 0 {
			return d, &httputil.ValidationError{Field: "withholding_tax", Msg: "must not be negative"}
		}
		if tax.GreaterThan(gross) {
			return d, &httputil.ValidationError{Field: "withholding_tax", Msg: "must not exceed gross amount"}
		}
		d.WithholdingTax = *tax
	}
	if v, ok := raw["currency"]; ok && string(v) != "null" {
		var c string
		if err := json.Unmarshal(v, &c); err != nil {
			return d, &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
		}
		if c = strings.TrimSpace(c); c != "" {
			if len(c) != 3 {
				return d, &httputil.ValidationError{Field: "currency", Msg: "must be 3 chars"}
			}
			d.Currency = strings.ToUpper(c)
		}
	}
	return d, nil
}

func requireDecimal(raw map[string]json.RawMessage, key string) (decimal.Decimal, *httputil.ValidationError) {
	return validation.RequiredDecimalStringOrNumber(raw, key, "required")
}
