package equitygrants

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// createRequest is the parsed-and-validated input ready for Store.Create.
type createRequest struct {
	GrantDate              time.Time
	Type                   string
	Company                string
	OwnerUserID            *int
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
		OwnerUserID:            req.OwnerUserID,
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
func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
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
		return r, &httputil.ValidationError{
			Field: "vest_cliff_months",
			Msg:   "Cliff months cannot exceed total vesting months",
		}
	}
	if r.Type == "option" && r.StrikePrice == nil {
		return r, &httputil.ValidationError{
			Field: "strike_price", Msg: "Stock options require a strike price",
		}
	}
	return r, nil
}

func requireGrantBasics(raw map[string]json.RawMessage, r *createRequest) *httputil.ValidationError {
	t, vErr := validation.RequiredDate(raw, "grant_date")
	if vErr != nil {
		return vErr
	}
	r.GrantDate = t

	grantType, vErr := validation.RequiredEnumString(raw, "type", validGrantTypes)
	if vErr != nil {
		return vErr
	}
	r.Type = grantType

	company, vErr := validation.RequiredTrimmedString(raw, "company", "Field required", "Company cannot be empty")
	if vErr != nil {
		return vErr
	}
	r.Company = company

	ownerID, vErr := validation.RequiredIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return vErr
	}
	r.OwnerUserID = ownerID

	totalShares, vErr := requirePositiveInt(raw, "total_shares", "Total shares must be greater than 0")
	if vErr != nil {
		return vErr
	}
	r.TotalShares = totalShares

	strike, vErr := validation.OptionalNonNegativeDecimal(raw, "strike_price", "Strike price must be non-negative")
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

func requireGrantVesting(raw map[string]json.RawMessage, r *createRequest) *httputil.ValidationError {
	vsd, vErr := validation.RequiredDate(raw, "vest_start_date")
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

	freq, vErr := validation.RequiredEnumString(raw, "vest_frequency", validFrequencies)
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

func optionalGrantLiquidity(raw map[string]json.RawMessage, r *createRequest) *httputil.ValidationError {
	if v, ok := raw["requires_liquidity_event"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return &httputil.ValidationError{Field: "requires_liquidity_event", Msg: "must be a boolean"}
		}
		r.RequiresLiquidityEvent = b
	}
	if t, vErr := validation.OptionalDate(raw, "liquidity_event_date"); vErr != nil {
		return vErr
	} else if t != nil {
		r.LiquidityEventDate = t
	}
	return nil
}

func optionalGrantTaxNotes(raw map[string]json.RawMessage, r *createRequest) *httputil.ValidationError {
	tax, vErr := optionalEnumString(raw, "tax_treatment", validTaxTreatments, "capital_gains_19")
	if vErr != nil {
		return vErr
	}
	r.TaxTreatment = tax

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return nil
}

// buildUpdatePatch reads the PATCH body. Null = no-op for scalar fields;
// vest_custom_schedule explicit-null clears the JSON column.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
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

func patchStrings(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if vErr := patchEnumString(raw, "type", validGrantTypes, &p.Type); vErr != nil {
		return vErr
	}
	if vErr := patchNonEmptyString(raw, "company", "Company cannot be empty", &p.Company); vErr != nil {
		return vErr
	}
	if vErr := patchOwnerUserID(raw, p); vErr != nil {
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
	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		p.Notes = &s
	}
	return nil
}

func patchNumbersAndBools(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if vErr := patchPositiveInt(raw, "total_shares", "Total shares must be greater than 0", &p.TotalShares); vErr != nil {
		return vErr
	}
	if vErr := patchNonNegativeIntPtr(raw, "vest_cliff_months", "Cliff months must be non-negative", &p.VestCliffMonths); vErr != nil {
		return vErr
	}
	if vErr := patchPositiveInt(raw, "vest_total_months", "Total vesting months must be greater than 0", &p.VestTotalMonths); vErr != nil {
		return vErr
	}
	if d, vErr := validation.OptionalNonNegativeDecimal(raw, "strike_price", "Strike price must be non-negative"); vErr != nil {
		return vErr
	} else if d != nil {
		p.StrikePrice = d
	}
	if v, ok := raw["requires_liquidity_event"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return &httputil.ValidationError{Field: "requires_liquidity_event", Msg: "must be a boolean"}
		}
		p.RequiresLiquidityEvent = &b
	}
	return nil
}

func patchDates(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	for _, f := range [...]struct {
		key  string
		dest **time.Time
	}{
		{"grant_date", &p.GrantDate},
		{"vest_start_date", &p.VestStartDate},
		{"liquidity_event_date", &p.LiquidityEventDate},
	} {
		t, vErr := validation.OptionalDate(raw, f.key)
		if vErr != nil {
			return vErr
		}
		if t != nil {
			*f.dest = t
		}
	}
	return nil
}

// patchSchedule reads vest_custom_schedule. Matches Python's update_equity_grant
// behavior: explicit null is treated the same as omitted (no-op); only an
// actual JSON array reassigns the field.
func patchSchedule(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	v, ok := raw["vest_custom_schedule"]
	if !ok || validation.IsNull(v) {
		return nil
	}
	schedule, vErr := parseCustomSchedule(v)
	if vErr != nil {
		return vErr
	}
	p.VestCustomScheduleSet = true
	p.VestCustomSchedule = schedule
	return nil
}

// --- shared decoders ---

func requirePositiveInt(raw map[string]json.RawMessage, key, msg string) (int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	if n <= 0 {
		return 0, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return n, nil
}

func optionalNonNegativeInt(raw map[string]json.RawMessage, key, msg string, fallback int) (int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return fallback, nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	if n < 0 {
		return 0, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return n, nil
}

func optionalEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, fallback string) (string, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return fallback, nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

func optionalCurrency(raw map[string]json.RawMessage, fallback string) (string, *httputil.ValidationError) {
	v, ok := raw["currency"]
	if !ok || validation.IsNull(v) {
		return fallback, nil
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

// --- PATCH helpers ---

func patchEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}, dest **string) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return &httputil.ValidationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	*dest = &s
	return nil
}

func patchNonEmptyString(raw map[string]json.RawMessage, key, emptyMsg string, dest **string) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return &httputil.ValidationError{Field: key, Msg: emptyMsg}
	}
	*dest = &s
	return nil
}

func patchCurrency(raw map[string]json.RawMessage, dest **string) *httputil.ValidationError {
	v, ok := raw["currency"]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return &httputil.ValidationError{Field: "currency", Msg: "must be a string"}
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	if _, ok := validCurrencies[s]; !ok {
		return &httputil.ValidationError{Field: "currency", Msg: "Currency must be one of [CHF, EUR, GBP, PLN, USD]"}
	}
	*dest = &s
	return nil
}

func patchPositiveInt(raw map[string]json.RawMessage, key, msg string, dest **int) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	if n <= 0 {
		return &httputil.ValidationError{Field: key, Msg: msg}
	}
	*dest = &n
	return nil
}

func patchNonNegativeIntPtr(raw map[string]json.RawMessage, key, msg string, dest **int) *httputil.ValidationError {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	if n < 0 {
		return &httputil.ValidationError{Field: key, Msg: msg}
	}
	*dest = &n
	return nil
}

// patchOwnerUserID reads owner_user_id from a PATCH body: present marks the
// field set; explicit null clears it (jointly owned).
func patchOwnerUserID(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	v, ok := raw["owner_user_id"]
	if !ok {
		return nil
	}
	p.OwnerUserIDSet = true
	if validation.IsNull(v) {
		p.OwnerUserID = nil
		return nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return &httputil.ValidationError{Field: "owner_user_id", Msg: "must be an integer"}
	}
	p.OwnerUserID = &n
	return nil
}
