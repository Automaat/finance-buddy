package equitygrants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// createRequest is the parsed-and-validated input ready for Store.Create.
type createRequest struct {
	GrantDate              time.Time
	Type                   string
	Company                string
	Owner                  string
	TotalShares            int
	StrikePrice            *decimal.Decimal
	Currency               string
	VestStartDate          time.Time
	VestCliffMonths        int
	VestTotalMonths        int
	VestFrequency          string
	VestCustomSchedule     []CustomScheduleEntry
	RequiresLiquidityEvent bool
	LiquidityEventDate     *time.Time
	TaxTreatment           string
	Notes                  *string
}

func requestToGrant(req createRequest) *EquityGrant {
	return &EquityGrant{
		GrantDate:              req.GrantDate,
		Type:                   req.Type,
		Company:                req.Company,
		Owner:                  req.Owner,
		TotalShares:            req.TotalShares,
		StrikePrice:            req.StrikePrice,
		Currency:               req.Currency,
		VestStartDate:          req.VestStartDate,
		VestCliffMonths:        req.VestCliffMonths,
		VestTotalMonths:        req.VestTotalMonths,
		VestFrequency:          req.VestFrequency,
		VestCustomSchedule:     req.VestCustomSchedule,
		RequiresLiquidityEvent: req.RequiresLiquidityEvent,
		LiquidityEventDate:     req.LiquidityEventDate,
		TaxTreatment:           req.TaxTreatment,
		Notes:                  req.Notes,
	}
}

// buildCreateRequest validates the POST body. Numbers go through
// decimal.NewFromString on the raw token to preserve Numeric precision.
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	if vErr := requireGrantBasics(raw, &r); vErr != nil {
		return r, vErr
	}
	if vErr := requireGrantVesting(raw, &r); vErr != nil {
		return r, vErr
	}
	if vErr := optionalGrantLiquidity(raw, &r); vErr != nil {
		return r, vErr
	}
	if vErr := optionalGrantTaxNotes(raw, &r); vErr != nil {
		return r, vErr
	}
	if r.VestCliffMonths > r.VestTotalMonths {
		return r, &validationError{
			Field: "vest_cliff_months",
			Msg:   "Cliff months cannot exceed total vesting months",
		}
	}
	if r.Type == "option" && r.StrikePrice == nil {
		return r, &validationError{
			Field: "strike_price", Msg: "Stock options require a strike price",
		}
	}
	return r, nil
}

func requireGrantBasics(raw map[string]json.RawMessage, r *createRequest) *validationError {
	t, vErr := requireDate(raw, "grant_date")
	if vErr != nil {
		return vErr
	}
	r.GrantDate = t

	grantType, vErr := requireEnumString(raw, "type", validGrantTypes)
	if vErr != nil {
		return vErr
	}
	r.Type = grantType

	company, vErr := requireString(raw, "company", "Company cannot be empty")
	if vErr != nil {
		return vErr
	}
	r.Company = company

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return vErr
	}
	r.Owner = owner

	totalShares, vErr := requirePositiveInt(raw, "total_shares", "Total shares must be greater than 0")
	if vErr != nil {
		return vErr
	}
	r.TotalShares = totalShares

	strike, vErr := optionalNonNegativeDecimal(raw, "strike_price", "Strike price must be non-negative")
	if vErr != nil {
		return vErr
	}
	r.StrikePrice = strike

	currency, vErr := optionalCurrency(raw, "USD")
	if vErr != nil {
		return vErr
	}
	r.Currency = currency
	return nil
}

func requireGrantVesting(raw map[string]json.RawMessage, r *createRequest) *validationError {
	vsd, vErr := requireDate(raw, "vest_start_date")
	if vErr != nil {
		return vErr
	}
	r.VestStartDate = vsd

	cliff, vErr := optionalNonNegativeInt(raw, "vest_cliff_months", "Cliff months must be non-negative", 0)
	if vErr != nil {
		return vErr
	}
	r.VestCliffMonths = cliff

	total, vErr := requirePositiveInt(raw, "vest_total_months", "Total vesting months must be greater than 0")
	if vErr != nil {
		return vErr
	}
	r.VestTotalMonths = total

	freq, vErr := requireEnumString(raw, "vest_frequency", validFrequencies)
	if vErr != nil {
		return vErr
	}
	r.VestFrequency = freq

	schedule, vErr := optionalCustomSchedule(raw)
	if vErr != nil {
		return vErr
	}
	r.VestCustomSchedule = schedule
	return nil
}

func optionalGrantLiquidity(raw map[string]json.RawMessage, r *createRequest) *validationError {
	if v, ok := raw["requires_liquidity_event"]; ok && !isNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return &validationError{Field: "requires_liquidity_event", Msg: "must be a boolean"}
		}
		r.RequiresLiquidityEvent = b
	}
	if v, ok := raw["liquidity_event_date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "liquidity_event_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &validationError{Field: "liquidity_event_date", Msg: "must be YYYY-MM-DD"}
		}
		r.LiquidityEventDate = &t
	}
	return nil
}

func optionalGrantTaxNotes(raw map[string]json.RawMessage, r *createRequest) *validationError {
	tax, vErr := optionalEnumString(raw, "tax_treatment", validTaxTreatments, "capital_gains_19")
	if vErr != nil {
		return vErr
	}
	r.TaxTreatment = tax

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return nil
}

// buildUpdatePatch reads the PATCH body. Null = no-op for scalar fields;
// vest_custom_schedule explicit-null clears the JSON column.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if vErr := patchStrings(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchNumbersAndBools(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchDates(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchSchedule(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

func patchStrings(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	if vErr := patchEnumString(raw, "type", validGrantTypes, &p.Type); vErr != nil {
		return vErr
	}
	if vErr := patchNonEmptyString(raw, "company", "Company cannot be empty", &p.Company); vErr != nil {
		return vErr
	}
	if vErr := patchNonEmptyString(raw, "owner", "Owner cannot be empty", &p.Owner); vErr != nil {
		return vErr
	}
	if vErr := patchCurrency(raw, &p.Currency); vErr != nil {
		return vErr
	}
	if vErr := patchEnumString(raw, "vest_frequency", validFrequencies, &p.VestFrequency); vErr != nil {
		return vErr
	}
	if vErr := patchEnumString(raw, "tax_treatment", validTaxTreatments, &p.TaxTreatment); vErr != nil {
		return vErr
	}
	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	return nil
}

func patchNumbersAndBools(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	if vErr := patchPositiveInt(raw, "total_shares", "Total shares must be greater than 0", &p.TotalShares); vErr != nil {
		return vErr
	}
	if vErr := patchNonNegativeIntPtr(raw, "vest_cliff_months", "Cliff months must be non-negative", &p.VestCliffMonths); vErr != nil {
		return vErr
	}
	if vErr := patchPositiveInt(raw, "vest_total_months", "Total vesting months must be greater than 0", &p.VestTotalMonths); vErr != nil {
		return vErr
	}
	if v, ok := raw["strike_price"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return &validationError{Field: "strike_price", Msg: "must be a number"}
		}
		if d.IsNegative() {
			return &validationError{Field: "strike_price", Msg: "Strike price must be non-negative"}
		}
		p.StrikePrice = &d
	}
	if v, ok := raw["requires_liquidity_event"]; ok && !isNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return &validationError{Field: "requires_liquidity_event", Msg: "must be a boolean"}
		}
		p.RequiresLiquidityEvent = &b
	}
	return nil
}

func patchDates(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	for _, f := range [...]struct {
		key  string
		dest **time.Time
	}{
		{"grant_date", &p.GrantDate},
		{"vest_start_date", &p.VestStartDate},
		{"liquidity_event_date", &p.LiquidityEventDate},
	} {
		v, ok := raw[f.key]
		if !ok || isNull(v) {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &validationError{Field: f.key, Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return &validationError{Field: f.key, Msg: "must be YYYY-MM-DD"}
		}
		*f.dest = &t
	}
	return nil
}

func patchSchedule(raw map[string]json.RawMessage, p *UpdatePatch) *validationError {
	v, ok := raw["vest_custom_schedule"]
	if !ok {
		return nil
	}
	p.VestCustomScheduleSet = true
	if isNull(v) {
		return nil
	}
	schedule, vErr := parseCustomSchedule(v)
	if vErr != nil {
		return vErr
	}
	p.VestCustomSchedule = schedule
	return nil
}

// --- shared decoders ---

func requireString(raw map[string]json.RawMessage, key, emptyMsg string) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
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

func requirePositiveInt(raw map[string]json.RawMessage, key, msg string) (int, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	if n <= 0 {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return n, nil
}

func optionalNonNegativeInt(raw map[string]json.RawMessage, key, msg string, fallback int) (int, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return fallback, nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	if n < 0 {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return n, nil
}

func requireEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

func optionalEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, fallback string) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return fallback, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

func optionalCurrency(raw map[string]json.RawMessage, fallback string) (string, *validationError) {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return fallback, nil
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

func optionalCustomSchedule(raw map[string]json.RawMessage) ([]CustomScheduleEntry, *validationError) {
	v, ok := raw["vest_custom_schedule"]
	if !ok || isNull(v) {
		return nil, nil
	}
	return parseCustomSchedule(v)
}

func parseCustomSchedule(v json.RawMessage) ([]CustomScheduleEntry, *validationError) {
	var entries []map[string]any
	if err := json.Unmarshal(v, &entries); err != nil {
		return nil, &validationError{Field: "vest_custom_schedule", Msg: "must be an array of objects"}
	}
	out := make([]CustomScheduleEntry, 0, len(entries))
	for _, entry := range entries {
		monthVal, hasMonth := entry["month"]
		pctVal, hasPct := entry["pct"]
		if !hasMonth || !hasPct {
			return nil, &validationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule entries require 'month' and 'pct'",
			}
		}
		month, mErr := toInt(monthVal)
		if mErr != nil || month < 0 {
			return nil, &validationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule month must be a non-negative integer",
			}
		}
		pct, pErr := toFloat(pctVal)
		if pErr != nil || pct < 0 {
			return nil, &validationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule pct must be a non-negative number",
			}
		}
		out = append(out, CustomScheduleEntry{Month: month, Pct: pct})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out, nil
}

func toInt(v any) (int, error) {
	switch x := v.(type) {
	case float64:
		return int(x), nil
	case int:
		return x, nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

// --- PATCH helpers ---

func patchEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, dest **string) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	*dest = &s
	return nil
}

func patchNonEmptyString(raw map[string]json.RawMessage, key, emptyMsg string, dest **string) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return &validationError{Field: key, Msg: emptyMsg}
	}
	*dest = &s
	return nil
}

func patchCurrency(raw map[string]json.RawMessage, dest **string) *validationError {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &validationError{Field: "currency", Msg: "must be a string"}
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	if _, ok := validCurrencies[s]; !ok {
		return &validationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
	}
	*dest = &s
	return nil
}

func patchPositiveInt(raw map[string]json.RawMessage, key, msg string, dest **int) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return &validationError{Field: key, Msg: "must be an integer"}
	}
	if n <= 0 {
		return &validationError{Field: key, Msg: msg}
	}
	*dest = &n
	return nil
}

func patchNonNegativeIntPtr(raw map[string]json.RawMessage, key, msg string, dest **int) *validationError {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return &validationError{Field: key, Msg: "must be an integer"}
	}
	if n < 0 {
		return &validationError{Field: key, Msg: msg}
	}
	*dest = &n
	return nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
