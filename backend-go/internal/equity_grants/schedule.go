package equitygrants

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
)

// optionalCustomSchedule reads the optional vest_custom_schedule key from
// a PATCH/POST body. Returns nil entries when the key is absent or null.
func optionalCustomSchedule(raw map[string]json.RawMessage) ([]CustomScheduleEntry, *httputil.ValidationError) {
	v, ok := raw["vest_custom_schedule"]
	if !ok || validation.IsNull(v) {
		return nil, nil
	}
	return parseCustomSchedule(v)
}

// parseCustomSchedule decodes a vest_custom_schedule JSON array into the
// typed CustomScheduleEntry slice, validating each row. Output is sorted
// by month so downstream vesting math can walk it linearly.
func parseCustomSchedule(v json.RawMessage) ([]CustomScheduleEntry, *httputil.ValidationError) {
	var entries []map[string]any
	if err := json.Unmarshal(v, &entries); err != nil {
		return nil, &httputil.ValidationError{Field: "vest_custom_schedule", Msg: "must be an array of objects"}
	}
	out := make([]CustomScheduleEntry, 0, len(entries))
	for _, entry := range entries {
		monthVal, hasMonth := entry["month"]
		pctVal, hasPct := entry["pct"]
		if !hasMonth || !hasPct {
			return nil, &httputil.ValidationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule entries require 'month' and 'pct'",
			}
		}
		month, mErr := toInt(monthVal)
		if mErr != nil || month < 0 {
			return nil, &httputil.ValidationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule month must be a non-negative integer",
			}
		}
		pct, pErr := toFloat(pctVal)
		if pErr != nil || pct < 0 {
			return nil, &httputil.ValidationError{
				Field: "vest_custom_schedule",
				Msg:   "Custom schedule pct must be a non-negative number",
			}
		}
		out = append(out, CustomScheduleEntry{Month: month, Pct: pct})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out, nil
}

// toInt coerces an arbitrary JSON-decoded value (float64 from json.Unmarshal,
// or the rare native int) into an int. Used by parseCustomSchedule which
// works through the untyped map[string]any path.
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

// toFloat coerces an arbitrary JSON-decoded value into a float64.
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
