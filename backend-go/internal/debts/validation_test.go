package debts

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

func TestRequireOwnerUserIDMissing(t *testing.T) {
	raw := rawJSON(t, map[string]any{})
	_, vErr := requireOwnerUserID(raw)
	if vErr == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestRequireOwnerUserIDExplicitNull(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": nil})
	got, vErr := requireOwnerUserID(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if got != nil {
		t.Fatalf("expected nil for explicit null, got %v", *got)
	}
}

func TestRequireOwnerUserIDValid(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": 7})
	got, vErr := requireOwnerUserID(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if got == nil || *got != 7 {
		t.Fatalf("expected 7, got %v", got)
	}
}

func TestDebtTypeToCategoryCovered(t *testing.T) {
	for dt := range validDebtTypes {
		if _, ok := debtTypeToCategory[dt]; !ok {
			t.Fatalf("debt_type %q has no category mapping", dt)
		}
	}
}
