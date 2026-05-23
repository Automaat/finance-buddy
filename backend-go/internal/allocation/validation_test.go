package allocation

import (
	"encoding/json"
	"testing"
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

func TestValidateCreateOK(t *testing.T) {
	if err := validateCreate(&createRequest{Category: "stock", TargetPct: 60}); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func TestValidateCreateUnknownCategory(t *testing.T) {
	err := validateCreate(&createRequest{Category: "crypto", TargetPct: 10})
	if err == nil || err.Field != "category" {
		t.Fatalf("expected category error, got %+v", err)
	}
}

func TestValidateCreateNegativePct(t *testing.T) {
	err := validateCreate(&createRequest{Category: "stock", TargetPct: -1})
	if err == nil || err.Field != "target_pct" {
		t.Fatalf("expected target_pct error, got %+v", err)
	}
}

func TestValidateCreatePctOver100(t *testing.T) {
	err := validateCreate(&createRequest{Category: "stock", TargetPct: 100.5})
	if err == nil || err.Field != "target_pct" {
		t.Fatalf("expected target_pct error, got %+v", err)
	}
}

func TestValidateCreateLiabilityCategoryRejected(t *testing.T) {
	err := validateCreate(&createRequest{Category: "mortgage", TargetPct: 50})
	if err == nil || err.Field != "category" {
		t.Fatalf("liability category should be rejected, got %+v", err)
	}
}

func TestReplaceBatchSumsTo100(t *testing.T) {
	err := validateReplaceBatch([]replaceItem{
		{Category: "stock", TargetPct: 60},
		{Category: "bond", TargetPct: 30},
		{Category: "gold", TargetPct: 10},
	})
	if err != nil {
		t.Fatalf("expected ok, got %+v", err)
	}
}

func TestReplaceBatchSumMismatch(t *testing.T) {
	err := validateReplaceBatch([]replaceItem{
		{Category: "stock", TargetPct: 60},
		{Category: "bond", TargetPct: 30},
	})
	if err == nil || err.Field != "targets" {
		t.Fatalf("expected sum error, got %+v", err)
	}
}

func TestReplaceBatchDuplicateCategory(t *testing.T) {
	err := validateReplaceBatch([]replaceItem{
		{Category: "stock", TargetPct: 60},
		{Category: "stock", TargetPct: 40},
	})
	if err == nil {
		t.Fatalf("expected duplicate error, got nil")
	}
}

func TestReplaceBatchEmptyOK(t *testing.T) {
	if err := validateReplaceBatch([]replaceItem{}); err != nil {
		t.Fatalf("empty payload should clear targets, got %+v", err)
	}
}

func TestBuildUpdatePatchTargetPct(t *testing.T) {
	patch, err := buildUpdatePatch(rawJSON(t, map[string]any{"target_pct": 42.5}))
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if patch.TargetPct == nil {
		t.Fatalf("target_pct not parsed")
	}
	got, _ := patch.TargetPct.Float64()
	if got != 42.5 {
		t.Fatalf("target_pct = %v, want 42.5", got)
	}
}

func TestBuildUpdatePatchOmittedNoChange(t *testing.T) {
	patch, err := buildUpdatePatch(rawJSON(t, map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if patch.TargetPct != nil {
		t.Fatalf("target_pct should be nil when omitted")
	}
}

func TestBuildUpdatePatchInvalidRange(t *testing.T) {
	if _, err := buildUpdatePatch(rawJSON(t, map[string]any{"target_pct": 150})); err == nil {
		t.Fatalf("expected range error")
	}
}
