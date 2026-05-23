package allocation

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// validAllocationCategories are the asset-side categories users can target.
// Mirrors accounts.validCategories minus the liability ones (mortgage,
// installment) — targeting a liability percentage doesn't make sense.
var validAllocationCategories = map[string]struct{}{
	"bank": {}, "saving_account": {}, "stock": {}, "bond": {}, "gold": {},
	"real_estate": {}, "ppk": {}, "fund": {}, "etf": {}, "vehicle": {},
}

func validateCreate(req *createRequest) *validationError {
	if _, ok := validAllocationCategories[req.Category]; !ok {
		return &validationError{Field: "category", Msg: fmt.Sprintf("invalid category %q", req.Category)}
	}
	if req.TargetPct < 0 || req.TargetPct > 100 {
		return &validationError{Field: "target_pct", Msg: "Target percentage must be between 0 and 100"}
	}
	return nil
}

// buildUpdatePatch reads a raw JSON object and decides, per field, whether
// it was omitted vs set. Only target_pct is mutable on PUT; changing scope
// (owner_user_id / category) is delete + create.
func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if v, ok := raw["target_pct"]; ok && !isNull(v) {
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			return p, &validationError{Field: "target_pct", Msg: "must be a number"}
		}
		if f < 0 || f > 100 {
			return p, &validationError{Field: "target_pct", Msg: "Target percentage must be between 0 and 100"}
		}
		d := decimal.NewFromFloat(f)
		p.TargetPct = &d
	}
	return p, nil
}

// validateReplaceBatch enforces the sum-to-100 invariant + per-category
// uniqueness within one bulk-replace payload.
func validateReplaceBatch(items []replaceItem) *validationError {
	seen := map[string]struct{}{}
	sum := 0.0
	for i, it := range items {
		if _, ok := validAllocationCategories[it.Category]; !ok {
			return &validationError{
				Field: fmt.Sprintf("targets[%d].category", i),
				Msg:   fmt.Sprintf("invalid category %q", it.Category),
			}
		}
		if _, dup := seen[it.Category]; dup {
			return &validationError{
				Field: fmt.Sprintf("targets[%d].category", i),
				Msg:   fmt.Sprintf("duplicate category %q in payload", it.Category),
			}
		}
		seen[it.Category] = struct{}{}
		if it.TargetPct < 0 || it.TargetPct > 100 {
			return &validationError{
				Field: fmt.Sprintf("targets[%d].target_pct", i),
				Msg:   "Target percentage must be between 0 and 100",
			}
		}
		sum += it.TargetPct
	}
	if len(items) == 0 {
		return nil
	}
	if absFloat(sum-100) > 0.01 {
		return &validationError{
			Field: "targets",
			Msg:   fmt.Sprintf("Target percentages must sum to 100 (got %.2f)", sum),
		}
	}
	return nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
