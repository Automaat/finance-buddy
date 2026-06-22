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
	year, vErr := validation.RequiredIntRange(raw, "year", 2000, currentYear+10,
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

	amt, vErr := validation.RequiredPositiveDecimal(raw, "limit_amount", "Field required", "Limit amount must be greater than 0")
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
	owner, vErr := validation.RequiredInt(raw, "owner_user_id", "Field required", "must be an integer")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = &owner

	month, vErr := validation.RequiredIntRange(raw, "month", 1, 12, "Month must be between 1 and 12")
	if vErr != nil {
		return r, vErr
	}
	r.Month = month

	currentYear := now().UTC().Year()
	year, vErr := validation.RequiredIntRange(raw, "year", 2019, currentYear+1,
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
	return validation.OptionalBoolDefault(raw, key, false)
}
