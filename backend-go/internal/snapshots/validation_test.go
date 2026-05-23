package snapshots

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func rawJSON(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", k, err)
		}
		out[k] = b
	}
	return out
}

func TestBuildCreateRequestMinimal(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"date": "2026-01-31",
		"values": []map[string]any{
			{"account_id": 1, "value": 100.5},
		},
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if r.Date.Format("2006-01-02") != "2026-01-31" {
		t.Fatalf("date mismatch: %s", r.Date)
	}
	if len(r.Values) != 1 || r.Values[0].AccountID == nil || *r.Values[0].AccountID != 1 {
		t.Fatalf("values mismatch: %+v", r.Values)
	}
	if !r.Values[0].Value.Equal(decimal.RequireFromString("100.5")) {
		t.Fatalf("value mismatch: %s", r.Values[0].Value)
	}
}

func TestBuildCreateRequestMissingDate(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"values": []map[string]any{{"account_id": 1, "value": 1}},
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "date" {
		t.Fatalf("expected date error, got %+v", vErr)
	}
}

func TestBuildCreateRequestInvalidDate(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"date":   "31-01-2026",
		"values": []map[string]any{{"account_id": 1, "value": 1}},
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "date" {
		t.Fatalf("expected date error, got %+v", vErr)
	}
}

func TestBuildCreateRequestEmptyValues(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"date":   "2026-01-31",
		"values": []map[string]any{},
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "values" {
		t.Fatalf("expected values error, got %+v", vErr)
	}
}

func TestBuildCreateRequestMissingValues(t *testing.T) {
	raw := rawJSON(t, map[string]any{"date": "2026-01-31"})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "values" {
		t.Fatalf("expected values error, got %+v", vErr)
	}
}

func TestParseValueEntryRequiresOneRef(t *testing.T) {
	entry := rawJSON(t, map[string]any{"value": 1})
	_, vErr := parseValueEntry(entry)
	if vErr == nil || vErr.Field != "values" {
		t.Fatalf("expected mutual-ref error, got %+v", vErr)
	}
}

func TestParseValueEntryRejectsBothRefs(t *testing.T) {
	entry := rawJSON(t, map[string]any{"asset_id": 1, "account_id": 2, "value": 5})
	_, vErr := parseValueEntry(entry)
	if vErr == nil || vErr.Field != "values" {
		t.Fatalf("expected mutual-ref error, got %+v", vErr)
	}
}

func TestParseValueEntryRequiresValue(t *testing.T) {
	entry := rawJSON(t, map[string]any{"account_id": 1})
	_, vErr := parseValueEntry(entry)
	if vErr == nil || vErr.Field != "value" {
		t.Fatalf("expected value error, got %+v", vErr)
	}
}

func TestParseValueEntryInvalidNumber(t *testing.T) {
	entry := map[string]json.RawMessage{
		"account_id": json.RawMessage("1"),
		"value":      json.RawMessage(`"not-a-number"`),
	}
	_, vErr := parseValueEntry(entry)
	if vErr == nil || vErr.Field != "value" {
		t.Fatalf("expected value parse error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchEmptyIsNoop(t *testing.T) {
	p, vErr := buildUpdatePatch(map[string]json.RawMessage{})
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if p.Date != nil || p.NotesSet || p.ValuesSet {
		t.Fatalf("expected zero patch, got %+v", p)
	}
}

func TestBuildUpdatePatchClearsNotes(t *testing.T) {
	raw := rawJSON(t, map[string]any{"notes": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.NotesSet || p.Notes != nil {
		t.Fatalf("expected NotesSet=true, value nil; got %+v", p)
	}
}

func TestBuildUpdatePatchSetsNotes(t *testing.T) {
	raw := rawJSON(t, map[string]any{"notes": "year end"})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.NotesSet || p.Notes == nil || *p.Notes != "year end" {
		t.Fatalf("notes mismatch: %+v", p)
	}
}

func TestBuildUpdatePatchValuesReplaces(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"values": []map[string]any{{"account_id": 7, "value": 99}},
	})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.ValuesSet || len(p.Values) != 1 {
		t.Fatalf("expected one replacement value, got %+v", p)
	}
}

func TestBuildUpdatePatchEmptyValuesArrayFails(t *testing.T) {
	raw := rawJSON(t, map[string]any{"values": []map[string]any{}})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "values" {
		t.Fatalf("expected values error, got %+v", vErr)
	}
}

func TestOptionalIntParses(t *testing.T) {
	raw := rawJSON(t, map[string]any{"x": 42})
	n, vErr := optionalInt(raw, "x")
	if vErr != nil || n == nil || *n != 42 {
		t.Fatalf("expected 42, got %+v err=%+v", n, vErr)
	}
}

func TestOptionalIntMissingReturnsNil(t *testing.T) {
	n, vErr := optionalInt(map[string]json.RawMessage{}, "x")
	if vErr != nil || n != nil {
		t.Fatalf("expected nil/nil, got %+v err=%+v", n, vErr)
	}
}

func TestOptionalIntInvalidFails(t *testing.T) {
	raw := rawJSON(t, map[string]any{"x": "abc"})
	_, vErr := optionalInt(raw, "x")
	if vErr == nil || vErr.Field != "x" {
		t.Fatalf("expected x error, got %+v", vErr)
	}
}
