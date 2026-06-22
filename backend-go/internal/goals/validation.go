package goals

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	if s, vErr := validation.OptionalTrimmedString(raw, "name", "Name cannot be empty"); vErr != nil {
		return vErr
	} else if s != nil {
		p.Name = s
	}
	if d, vErr := validation.OptionalPositiveDecimal(raw, "target_amount", "Target amount must be greater than 0"); vErr != nil {
		return vErr
	} else if d != nil {
		p.TargetAmount = d
	}
	if t, vErr := validation.OptionalDate(raw, "target_date"); vErr != nil {
		return vErr
	} else if t != nil {
		p.TargetDate = t
	}
	if d, vErr := validation.OptionalNonNegativeDecimal(raw, "current_amount", "Current amount must be non-negative"); vErr != nil {
		return vErr
	} else if d != nil {
		p.CurrentAmount = d
	}
	if d, vErr := validation.OptionalNonNegativeDecimal(
		raw,
		"monthly_contribution",
		"Monthly contribution must be non-negative",
	); vErr != nil {
		return vErr
	} else if d != nil {
		p.MonthlyContribution = d
	}
	if b, vErr := validation.OptionalBool(raw, "is_completed"); vErr != nil {
		return vErr
	} else if b != nil {
		p.IsCompleted = b
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
