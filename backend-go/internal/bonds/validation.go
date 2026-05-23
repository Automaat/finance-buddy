package bonds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// validateCreate normalizes and rejects bad inputs on the create path.
func validateCreate(req *createRequest) *validationError {
	t := BondType(strings.TrimSpace(strings.ToUpper(req.Type)))
	if !t.IsValid() {
		return &validationError{Field: "type", Msg: fmt.Sprintf("invalid bond type %q", req.Type)}
	}
	req.Type = string(t)

	series := strings.TrimSpace(req.Series)
	if series == "" {
		return &validationError{Field: "series", Msg: "Series cannot be empty"}
	}
	req.Series = series

	if req.FaceValue <= 0 {
		return &validationError{Field: "face_value", Msg: "Face value must be greater than 0"}
	}
	if time.Time(req.PurchaseDate).IsZero() {
		return &validationError{Field: "purchase_date", Msg: "Field required"}
	}
	if req.FirstYearRate < 0 || req.FirstYearRate > 100 {
		return &validationError{Field: "first_year_rate", Msg: "First year rate must be between 0 and 100"}
	}
	if req.Margin < 0 || req.Margin > 100 {
		return &validationError{Field: "margin", Msg: "Margin must be between 0 and 100"}
	}
	return nil
}

// buildUpdatePatch reads a raw JSON object and decides, per field, whether
// it was omitted, explicitly null, or set to a value. Mirrors the goals
// package's Pydantic-shaped sparse-update behavior.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if v, ok := raw["type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "type", Msg: "must be a string"}
		}
		t := BondType(strings.TrimSpace(strings.ToUpper(s)))
		if !t.IsValid() {
			return p, &validationError{Field: "type", Msg: fmt.Sprintf("invalid bond type %q", s)}
		}
		p.Type = &t
	}
	if v, ok := raw["series"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "series", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &validationError{Field: "series", Msg: "Series cannot be empty"}
		}
		p.Series = &s
	}
	if vErr := patchPositiveAmount(raw, "face_value", &p.FaceValue, "Face value must be greater than 0"); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["purchase_date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "purchase_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &validationError{Field: "purchase_date", Msg: "must be YYYY-MM-DD"}
		}
		p.PurchaseDate = &t
	}
	if v, ok := raw["owner_user_id"]; ok {
		p.OwnerUserIDSet = true
		if !isNull(v) {
			var id int
			if err := json.Unmarshal(v, &id); err != nil {
				return p, &validationError{Field: "owner_user_id", Msg: "must be an integer"}
			}
			p.OwnerUserID = &id
		}
	}
	if vErr := patchRatePercent(raw, "first_year_rate", &p.FirstYearRate); vErr != nil {
		return p, vErr
	}
	if vErr := patchRatePercent(raw, "margin", &p.Margin); vErr != nil {
		return p, vErr
	}
	if v, ok := raw["capitalize"]; ok && !isNull(v) {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return p, &validationError{Field: "capitalize", Msg: "must be a boolean"}
		}
		p.Capitalize = &b
	}
	return p, nil
}

func patchPositiveAmount(raw map[string]json.RawMessage, field string, dest **decimal.Decimal, msg string) *validationError {
	v, ok := raw[field]
	if !ok || isNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &validationError{Field: field, Msg: "must be a number"}
	}
	if f <= 0 {
		return &validationError{Field: field, Msg: msg}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}

func patchRatePercent(raw map[string]json.RawMessage, field string, dest **decimal.Decimal) *validationError {
	v, ok := raw[field]
	if !ok || isNull(v) {
		return nil
	}
	var f float64
	if err := json.Unmarshal(v, &f); err != nil {
		return &validationError{Field: field, Msg: "must be a number"}
	}
	if f < 0 || f > 100 {
		return &validationError{Field: field, Msg: fmt.Sprintf("%s must be between 0 and 100", field)}
	}
	d := decimal.NewFromFloat(f)
	*dest = &d
	return nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
