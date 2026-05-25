package holdings

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type validationError struct {
	Field string
	Msg   string
}

func buildSecurityInput(raw map[string]json.RawMessage) (Security, *validationError) {
	var s Security
	s.Currency = "PLN"
	if v, ok := raw["symbol"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Symbol); err != nil {
			return s, &validationError{Field: "symbol", Msg: "must be a string"}
		}
	}
	if s.Symbol = strings.TrimSpace(s.Symbol); s.Symbol == "" {
		return s, &validationError{Field: "symbol", Msg: "required"}
	}
	if len(s.Symbol) > 32 {
		return s, &validationError{Field: "symbol", Msg: "max 32 chars"}
	}
	if v, ok := raw["isin"]; ok && string(v) != "null" {
		var isin string
		if err := json.Unmarshal(v, &isin); err != nil {
			return s, &validationError{Field: "isin", Msg: "must be a string"}
		}
		if isin = strings.TrimSpace(isin); isin != "" {
			if len(isin) != 12 {
				return s, &validationError{Field: "isin", Msg: "must be 12 chars"}
			}
			s.ISIN = &isin
		}
	}
	if v, ok := raw["name"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Name); err != nil {
			return s, &validationError{Field: "name", Msg: "must be a string"}
		}
	}
	if s.Name = strings.TrimSpace(s.Name); s.Name == "" {
		return s, &validationError{Field: "name", Msg: "required"}
	}
	if len(s.Name) > 200 {
		return s, &validationError{Field: "name", Msg: "max 200 chars"}
	}
	if v, ok := raw["asset_type"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.AssetType); err != nil {
			return s, &validationError{Field: "asset_type", Msg: "must be a string"}
		}
	}
	if !validAssetType(s.AssetType) {
		return s, &validationError{Field: "asset_type", Msg: "must be one of stock|etf|bond|fund"}
	}
	if v, ok := raw["currency"]; ok && string(v) != "null" {
		var c string
		if err := json.Unmarshal(v, &c); err != nil {
			return s, &validationError{Field: "currency", Msg: "must be a string"}
		}
		if c = strings.TrimSpace(c); c != "" {
			if len(c) != 3 {
				return s, &validationError{Field: "currency", Msg: "must be 3 chars"}
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

func buildLotInput(raw map[string]json.RawMessage) (Lot, *validationError) {
	var l Lot
	if vErr := requireInt(raw, "account_id", &l.AccountID); vErr != nil {
		return l, vErr
	}
	if l.AccountID <= 0 {
		return l, &validationError{Field: "account_id", Msg: "must be positive"}
	}
	if vErr := requireInt(raw, "security_id", &l.SecurityID); vErr != nil {
		return l, vErr
	}
	if l.SecurityID <= 0 {
		return l, &validationError{Field: "security_id", Msg: "must be positive"}
	}
	var sideStr string
	if vErr := requireString(raw, "side", &sideStr); vErr != nil {
		return l, vErr
	}
	if !IsValidSide(sideStr) {
		return l, &validationError{Field: "side", Msg: "must be buy or sell"}
	}
	l.Side = Side(sideStr)
	qty, vErr := requireDecimal(raw, "quantity")
	if vErr != nil {
		return l, vErr
	}
	if qty.Sign() <= 0 {
		return l, &validationError{Field: "quantity", Msg: "must be positive"}
	}
	l.Quantity = qty
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return l, vErr
	}
	if price.Sign() < 0 {
		return l, &validationError{Field: "price", Msg: "must not be negative"}
	}
	l.Price = price
	if v, ok := raw["fee"]; ok && string(v) != "null" {
		fee, ferr := decimal.NewFromString(strings.Trim(string(v), `"`))
		if ferr != nil {
			return l, &validationError{Field: "fee", Msg: "must be a number"}
		}
		if fee.Sign() < 0 {
			return l, &validationError{Field: "fee", Msg: "must not be negative"}
		}
		l.Fee = fee
	}
	dateStr, vErr := requireString2(raw, "date")
	if vErr != nil {
		return l, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return l, &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	l.Date = parsed
	return l, nil
}

func buildQuoteInput(raw map[string]json.RawMessage, securityID int) (PriceQuote, *validationError) {
	q := PriceQuote{SecurityID: securityID, Source: "manual"}
	dateStr, vErr := requireString2(raw, "date")
	if vErr != nil {
		return q, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return q, &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	q.Date = parsed
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return q, vErr
	}
	if price.Sign() < 0 {
		return q, &validationError{Field: "price", Msg: "must not be negative"}
	}
	q.Price = price
	if v, ok := raw["source"]; ok && string(v) != "null" {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return q, &validationError{Field: "source", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			if len(s) > 40 {
				return q, &validationError{Field: "source", Msg: "max 40 chars"}
			}
			q.Source = s
		}
	}
	return q, nil
}

func requireInt(raw map[string]json.RawMessage, key string, dest *int) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be integer"}
	}
	return nil
}

func requireString(raw map[string]json.RawMessage, key string, dest *string) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	if *dest == "" {
		return &validationError{Field: key, Msg: "cannot be empty"}
	}
	return nil
}

func requireString2(raw map[string]json.RawMessage, key string) (string, *validationError) {
	var s string
	if vErr := requireString(raw, key, &s); vErr != nil {
		return "", vErr
	}
	return s, nil
}

func requireDecimal(raw map[string]json.RawMessage, key string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return decimal.Zero, &validationError{Field: key, Msg: "required"}
	}
	d, err := decimal.NewFromString(strings.Trim(string(v), `"`))
	if err != nil {
		return decimal.Zero, &validationError{Field: key, Msg: "must be a number"}
	}
	return d, nil
}
