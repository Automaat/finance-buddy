package bonds

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// validateCreate normalizes and rejects bad inputs on the create path.
func validateCreate(req *createRequest) *httputil.ValidationError {
	t := BondType(strings.TrimSpace(strings.ToUpper(req.Type)))
	if !t.IsValid() {
		return &httputil.ValidationError{Field: "type", Msg: fmt.Sprintf("invalid bond type %q", req.Type)}
	}
	req.Type = string(t)

	series := strings.TrimSpace(req.Series)
	if series == "" {
		return &httputil.ValidationError{Field: "series", Msg: "Series cannot be empty"}
	}
	req.Series = series

	if req.FaceValue <= 0 {
		return &httputil.ValidationError{Field: "face_value", Msg: "Face value must be greater than 0"}
	}
	if time.Time(req.PurchaseDate).IsZero() {
		return &httputil.ValidationError{Field: "purchase_date", Msg: "Field required"}
	}
	if req.FirstYearRate < 0 || req.FirstYearRate > 100 {
		return &httputil.ValidationError{Field: "first_year_rate", Msg: "First year rate must be between 0 and 100"}
	}
	if req.Margin < 0 || req.Margin > 100 {
		return &httputil.ValidationError{Field: "margin", Msg: "Margin must be between 0 and 100"}
	}
	return nil
}

// buildUpdatePatch reads a raw JSON object and decides, per field, whether
// it was omitted, explicitly null, or set to a value. Mirrors the goals
// package's Pydantic-shaped sparse-update behavior.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *httputil.ValidationError) {
	var p UpdatePatch
	if v, ok := raw["type"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "type", Msg: "must be a string"}
		}
		t := BondType(strings.TrimSpace(strings.ToUpper(s)))
		if !t.IsValid() {
			return p, &httputil.ValidationError{Field: "type", Msg: fmt.Sprintf("invalid bond type %q", s)}
		}
		p.Type = &t
	}
	if v, ok := raw["series"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "series", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &httputil.ValidationError{Field: "series", Msg: "Series cannot be empty"}
		}
		p.Series = &s
	}
	if vErr := patchPositiveAmount(raw, "face_value", &p.FaceValue, "Face value must be greater than 0"); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["purchase_date"]; ok && !validation.IsNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &httputil.ValidationError{Field: "purchase_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &httputil.ValidationError{Field: "purchase_date", Msg: "must be YYYY-MM-DD"}
		}
		p.PurchaseDate = &t
	}
	if v, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		if !validation.IsNull(v) {
			var id int
			if err := json.Unmarshal(v, &id); err != nil {
				return p, &httputil.ValidationError{Field: "owner_user_id", Msg: "must be an integer"}
			}
			p.OwnerUserID = &id
		}
	}
	if v, ok := raw["account_id"]; ok {
		p.AccountIDSet = true
		if !validation.IsNull(v) {
			var id int
			if err := json.Unmarshal(v, &id); err != nil {
				return p, &httputil.ValidationError{Field: "account_id", Msg: "must be an integer"}
			}
			p.AccountID = &id
		}
	}
	if vErr := patchRatePercent(raw, "first_year_rate", &p.FirstYearRate); vErr != nil {
		return p, vErr
	}
	if vErr := patchRatePercent(raw, "margin", &p.Margin); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["capitalize"]; ok && !validation.IsNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return p, &httputil.ValidationError{Field: "capitalize", Msg: "must be a boolean"}
		}
		p.Capitalize = &b
	}
	return p, nil
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

func patchRatePercent(raw map[string]json.RawMessage, field string, dest **decimal.Decimal) *httputil.ValidationError {
	v, ok := raw[field]
	if !ok || validation.IsNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &httputil.ValidationError{Field: field, Msg: "must be a number"}
	}
	if f < 0 || f > 100 {
		return &httputil.ValidationError{Field: field, Msg: fmt.Sprintf("%s must be between 0 and 100", field)}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}
