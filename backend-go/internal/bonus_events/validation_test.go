package bonusevents

import (
	"encoding/json"
	"testing"
	"time"
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

func validCreate() map[string]any {
	return map[string]any{
		"date":          "2026-01-15",
		"amount":        5000,
		"type":          "annual",
		"company":       "Acme",
		"owner_user_id": 1,
		"contract_type": "UOP",
	}
}

func TestBuildCreateRequestOK(t *testing.T) {
	r, vErr := buildCreateRequest(rawJSON(t, validCreate()))
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if r.Currency != "PLN" {
		t.Fatalf("currency fallback failed: %q", r.Currency)
	}
	if r.Type != "annual" || r.ContractType != "UOP" || r.Company != "Acme" {
		t.Fatalf("fields mismatch: %+v", r)
	}
}

func TestBuildCreateRequestFutureDate(t *testing.T) {
	m := validCreate()
	m["date"] = time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02")
	_, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr == nil || vErr.Field != "date" {
		t.Fatalf("expected date error, got %+v", vErr)
	}
}

func TestBuildCreateRequestNegativeAmount(t *testing.T) {
	m := validCreate()
	m["amount"] = -10
	_, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr == nil || vErr.Field != "amount" {
		t.Fatalf("expected amount error, got %+v", vErr)
	}
}

func TestBuildCreateRequestInvalidType(t *testing.T) {
	m := validCreate()
	m["type"] = "bogus"
	_, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr == nil || vErr.Field != "type" {
		t.Fatalf("expected type error, got %+v", vErr)
	}
}

func TestBuildCreateRequestInvalidContract(t *testing.T) {
	m := validCreate()
	m["contract_type"] = "X"
	_, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr == nil || vErr.Field != "contract_type" {
		t.Fatalf("expected contract_type error, got %+v", vErr)
	}
}

func TestBuildCreateRequestBlankCompany(t *testing.T) {
	m := validCreate()
	m["company"] = "  "
	_, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr == nil || vErr.Field != "company" {
		t.Fatalf("expected company error, got %+v", vErr)
	}
}

func TestBuildCreateRequestNotes(t *testing.T) {
	m := validCreate()
	m["notes"] = "tax refund"
	r, vErr := buildCreateRequest(rawJSON(t, m))
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if r.Notes == nil || *r.Notes != "tax refund" {
		t.Fatalf("notes mismatch: %+v", r.Notes)
	}
}

func TestBuildUpdatePatchEmpty(t *testing.T) {
	p, vErr := buildUpdatePatch(map[string]json.RawMessage{})
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.Date != nil || p.Amount != nil || p.Notes != nil {
		t.Fatalf("expected zero patch, got %+v", p)
	}
}

func TestBuildUpdatePatchNullNotesIsNoop(t *testing.T) {
	raw := rawJSON(t, map[string]any{"notes": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.Notes != nil {
		t.Fatalf("null notes should be no-op, got %+v", p.Notes)
	}
}

func TestBuildUpdatePatchSetsNotes(t *testing.T) {
	raw := rawJSON(t, map[string]any{"notes": "bonus"})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.Notes == nil || *p.Notes != "bonus" {
		t.Fatalf("notes mismatch: %+v", p)
	}
}

func TestBuildUpdatePatchClearsOwnerExplicitly(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if !p.OwnerUserIDSet || p.OwnerUserID != nil {
		t.Fatalf("expected owner cleared, got %+v", p)
	}
}

func TestBuildUpdatePatchInvalidType(t *testing.T) {
	raw := rawJSON(t, map[string]any{"type": "bogus"})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "type" {
		t.Fatalf("expected type error, got %+v", vErr)
	}
}
