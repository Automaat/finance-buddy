package retirement

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

type limitRequest struct {
	Year           int
	AccountWrapper string
	OwnerUserID    *int
	LimitAmount    decimal.Decimal
	Notes          *string
}

var validWrappers = map[string]struct{}{
	"IKE": {}, "IKZE": {}, "PPK": {},
}

func buildLimitRequest(raw map[string]json.RawMessage, now func() time.Time) (limitRequest, *httputil.ValidationError) {
	var r limitRequest
	currentYear := now().UTC().Year()
	year, vErr := requireIntRange(raw, "year", 2000, currentYear+10,
		fmt.Sprintf("Year must be between 2000 and %d", currentYear+10))
	if vErr != nil {
		return r, vErr
	}
	r.Year = year

	wrap, vErr := validation.RequiredEnumString(raw, "account_wrapper", validWrappers)
	if vErr != nil {
		return r, vErr
	}
	r.AccountWrapper = wrap

	owner, vErr := validation.RequiredIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = owner

	amt, vErr := requirePositiveDecimal(raw, "limit_amount", "Limit amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.LimitAmount = amt

	if v, ok := raw["notes"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &httputil.ValidationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

type generateRequest struct {
	OwnerUserID    *int
	Month          int
	Year           int
	IncludeWelcome bool
	IncludeAnnual  bool
}

func buildGenerateRequest(raw map[string]json.RawMessage, now func() time.Time) (generateRequest, *httputil.ValidationError) {
	var r generateRequest
	// PPK contributions are always generated for a specific person — a null
	// (jointly owned) owner is rejected up front rather than failing later
	// with a confusing 404.
	owner, vErr := requireInt(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = &owner

	month, vErr := requireIntRange(raw, "month", 1, 12, "Month must be between 1 and 12")
	if vErr != nil {
		return r, vErr
	}
	r.Month = month

	currentYear := now().UTC().Year()
	year, vErr := requireIntRange(raw, "year", 2019, currentYear+1,
		fmt.Sprintf("Year must be between 2019 and %d", currentYear+1))
	if vErr != nil {
		return r, vErr
	}
	r.Year = year

	welcome, vErr := optionalBool(raw, "include_welcome_subsidy")
	if vErr != nil {
		return r, vErr
	}
	r.IncludeWelcome = welcome
	annual, vErr := optionalBool(raw, "include_annual_subsidy")
	if vErr != nil {
		return r, vErr
	}
	r.IncludeAnnual = annual
	return r, nil
}

// optionalBool reads a boolean key. Missing/null returns false; a present
// non-bool value (e.g. "true" sent as a string) is rejected so the client
// gets a 422 instead of silently degrading to false.
func optionalBool(raw map[string]json.RawMessage, key string) (bool, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return false, nil
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, &httputil.ValidationError{Field: key, Msg: "must be a boolean"}
	}
	return b, nil
}

// requireInt reads an integer key that must be present and non-null.
func requireInt(raw map[string]json.RawMessage, key string) (int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	return n, nil
}

func requireIntRange(raw map[string]json.RawMessage, key string, lo, hi int, msg string) (int, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return 0, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &httputil.ValidationError{Field: key, Msg: "must be an integer"}
	}
	if n < lo || n > hi {
		return 0, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return n, nil
}

func requirePositiveDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *httputil.ValidationError) {
	v, ok := raw[key]
	if !ok || validation.IsNull(v) {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "Field required"}
	}
	d, err := validation.RawDecimal(v)
	if err != nil {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: "must be a number"}
	}
	if !d.IsPositive() {
		return decimal.Decimal{}, &httputil.ValidationError{Field: key, Msg: msg}
	}
	return d, nil
}
