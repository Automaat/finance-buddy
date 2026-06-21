package goals

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

func validateCreate(req *createRequest) *httputil.ValidationError {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
	}
	req.Name = name
	if time.Time(req.TargetDate).IsZero() {
		return &httputil.ValidationError{Field: "target_date", Msg: "Field required"}
	}
	if req.TargetAmount <= 0 {
		return &httputil.ValidationError{Field: "target_amount", Msg: "Target amount must be greater than 0"}
	}
	if req.CurrentAmount < 0 {
		return &httputil.ValidationError{Field: "current_amount", Msg: "Current amount must be non-negative"}
	}
	if req.MonthlyContribution < 0 {
		return &httputil.ValidationError{
			Field: "monthly_contribution",
			Msg:   "Monthly contribution must be non-negative",
		}
	}
	if req.Category != nil {
		if _, ok := validCategories[*req.Category]; !ok {
			return &httputil.ValidationError{
				Field: "category",
				Msg:   fmt.Sprintf("invalid category %q", *req.Category),
			}
		}
	}
	return nil
}

// buildUpdatePatch reads a raw JSON object and decides, per field, whether
// it was omitted, explicitly null, or set to a value. This is the Go analog
// of Pydantic's model_fields_set used in update_goal.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if vErr := patchScalarFields(raw, &p); vErr != nil {
		return p, vErr
	}
	if vErr := patchNullableRefs(raw, &p); vErr != nil {
		return p, vErr
	}
	return p, nil
}

// patchScalarFields handles non-nullable-but-omittable update fields.
// Matches Python's GoalUpdate validators: explicit null is treated as
// "no-op" (the validator returns None and the service skips that field).
func patchScalarFields(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if v, ok := raw["name"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return &httputil.ValidationError{Field: "name", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return &httputil.ValidationError{Field: "name", Msg: "Name cannot be empty"}
		}
		p.Name = &s
	}
	if vErr := patchPositiveAmount(raw, "target_amount", &p.TargetAmount, "Target amount must be greater than 0"); vErr != nil {
		return vErr
	}
	if t, vErr := validation.OptionalDate(raw, "target_date"); vErr != nil {
		return vErr
	} else if t != nil {
		p.TargetDate = t
	}
	if vErr := patchNonNegativeAmount(raw, "current_amount", &p.CurrentAmount, "Current amount must be non-negative"); vErr != nil {
		return vErr
	}
	if vErr := patchNonNegativeAmount(raw, "monthly_contribution", &p.MonthlyContribution, "Monthly contribution must be non-negative"); vErr != nil {
		return vErr
	}
	if v, ok := raw["is_completed"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return &httputil.ValidationError{Field: "is_completed", Msg: "must be a boolean"}
		}
		p.IsCompleted = &b
	}
	return nil
}

// patchNullableRefs handles the two fields where the caller can explicitly
// send null to clear the link (account_id, category).
func patchNullableRefs(raw map[string]json.RawMessage, p *UpdatePatch) *httputil.ValidationError {
	if _, ok := raw["account_id"]; ok {
		p.AccountIDSet = true
		accountID, vErr := validation.OptionalInt(raw, "account_id", "must be an integer")
		if vErr != nil {
			return vErr
		}
		p.AccountID = accountID
	}
	if v, ok := raw["category"]; ok {
		p.CategorySet = true
		if !validation.IsNull(v) {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return &httputil.ValidationError{Field: "category", Msg: "must be a string"}
			}
			if _, valid := validCategories[s]; !valid {
				return &httputil.ValidationError{
					Field: "category",
					Msg:   fmt.Sprintf("invalid category %q", s),
				}
			}
			p.Category = &s
		}
	}
	return nil
}

func patchPositiveAmount(raw map[string]json.RawMessage, field string, dest **decimal.Decimal, msg string) *httputil.ValidationError {
	v, ok := raw[field]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &httputil.ValidationError{Field: field, Msg: "must be a number"}
	}
	if f <= 0 {
		return &httputil.ValidationError{Field: field, Msg: msg}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}

func patchNonNegativeAmount(raw map[string]json.RawMessage, field string, dest **decimal.Decimal, msg string) *httputil.ValidationError {
	v, ok := raw[field]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &httputil.ValidationError{Field: field, Msg: "must be a number"}
	}
	if f < 0 {
		return &httputil.ValidationError{Field: field, Msg: msg}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}
