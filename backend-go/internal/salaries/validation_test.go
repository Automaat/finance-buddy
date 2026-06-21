package salaries

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var testNow = func() time.Time { return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC) }

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
		"date":          "2026-05-01",
		"gross_amount":  10000,
		"contract_type": "UOP",
		"company":       "Acme",
		"owner_user_id": 1,
	}
}

func TestBuildCreateRequestOK(t *testing.T) {
	r, vErr := buildCreateRequest(rawJSON(t, validCreate()), testNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if r.ContractType != "UOP" || r.Company != "Acme" || *r.OwnerUserID != 1 {
		t.Fatalf("unexpected: %+v", r)
	}
	if !r.GrossAmount.Equal(decimal.NewFromInt(10000)) {
		t.Fatalf("gross_amount mismatch: %s", r.GrossAmount)
	}
}

func TestBuildCreateRequestFutureDate(t *testing.T) {
	m := validCreate()
	m["date"] = "2027-01-01"
	_, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr == nil || vErr.Field != "date" {
		t.Fatalf("expected date error, got %+v", vErr)
	}
}

func TestBuildCreateRequestInvalidContract(t *testing.T) {
	m := validCreate()
	m["contract_type"] = "BOGUS"
	_, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr == nil || vErr.Field != "contract_type" {
		t.Fatalf("expected contract_type error, got %+v", vErr)
	}
}

func TestBuildCreateRequestBlankCompany(t *testing.T) {
	m := validCreate()
	m["company"] = "  "
	_, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr == nil || vErr.Field != "company" {
		t.Fatalf("expected company error, got %+v", vErr)
	}
}

func TestBuildCreateRequestNonPositiveGross(t *testing.T) {
	m := validCreate()
	m["gross_amount"] = 0
	_, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr == nil || vErr.Field != "gross_amount" {
		t.Fatalf("expected gross_amount error, got %+v", vErr)
	}
}

func TestBuildCreateRequestMissingOwner(t *testing.T) {
	m := validCreate()
	delete(m, "owner_user_id")
	_, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr == nil || vErr.Field != "owner_user_id" {
		t.Fatalf("expected owner_user_id error, got %+v", vErr)
	}
}

func TestBuildCreateRequestSharedOwner(t *testing.T) {
	m := validCreate()
	m["owner_user_id"] = nil
	r, vErr := buildCreateRequest(rawJSON(t, m), testNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if r.OwnerUserID != nil {
		t.Fatalf("expected shared owner, got %+v", r.OwnerUserID)
	}
}

func TestBuildUpdatePatchEmpty(t *testing.T) {
	p, vErr := buildUpdatePatch(map[string]json.RawMessage{}, testNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.Date != nil || p.GrossAmount != nil || p.ContractType != nil || p.Company != nil || p.OwnerUserIDSet {
		t.Fatalf("expected zero patch, got %+v", p)
	}
}

func TestBuildUpdatePatchClearsOwner(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": nil})
	p, vErr := buildUpdatePatch(raw, testNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if !p.OwnerUserIDSet || p.OwnerUserID != nil {
		t.Fatalf("expected owner cleared, got %+v", p)
	}
}

func TestBuildUpdatePatchSetsOwner(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": 42})
	p, vErr := buildUpdatePatch(raw, testNow)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if !p.OwnerUserIDSet || p.OwnerUserID == nil || *p.OwnerUserID != 42 {
		t.Fatalf("expected owner=42, got %+v", p)
	}
}

func TestBuildUpdatePatchFutureDateFails(t *testing.T) {
	raw := rawJSON(t, map[string]any{"date": "2027-01-01"})
	_, vErr := buildUpdatePatch(raw, testNow)
	if vErr == nil || vErr.Field != "date" {
		t.Fatalf("expected date error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchNonPositiveGross(t *testing.T) {
	raw := rawJSON(t, map[string]any{"gross_amount": -1})
	_, vErr := buildUpdatePatch(raw, testNow)
	if vErr == nil || vErr.Field != "gross_amount" {
		t.Fatalf("expected gross_amount error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchInvalidContract(t *testing.T) {
	raw := rawJSON(t, map[string]any{"contract_type": "WAT"})
	_, vErr := buildUpdatePatch(raw, testNow)
	if vErr == nil || vErr.Field != "contract_type" {
		t.Fatalf("expected contract_type error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchBlankCompany(t *testing.T) {
	raw := rawJSON(t, map[string]any{"company": "  "})
	_, vErr := buildUpdatePatch(raw, testNow)
	if vErr == nil || vErr.Field != "company" {
		t.Fatalf("expected company error, got %+v", vErr)
	}
}

func TestCurrentSalaryFromRecentUsesMostRecentRecord(t *testing.T) {
	recent := map[int][]SalaryRecord{
		2: []SalaryRecord{
			{GrossAmount: decimal.NewFromInt(12000)},
			{GrossAmount: decimal.NewFromInt(10000)},
		},
		3: []SalaryRecord{},
	}
	got := currentSalaryFromRecent(recent)
	if !got[2].Equal(decimal.NewFromInt(12000)) {
		t.Fatalf("current salary = %s, want 12000", got[2])
	}
	if _, ok := got[3]; ok {
		t.Fatalf("owner 3 should be absent when there are no recent records")
	}
}

func TestHasPreviousSalary(t *testing.T) {
	if hasPreviousSalary(map[int][]SalaryRecord{
		1: []SalaryRecord{{GrossAmount: decimal.NewFromInt(1)}},
	}) {
		t.Fatal("single recent record should not have a previous salary")
	}
	if !hasPreviousSalary(map[int][]SalaryRecord{
		1: []SalaryRecord{
			{GrossAmount: decimal.NewFromInt(2)},
			{GrossAmount: decimal.NewFromInt(1)},
		},
	}) {
		t.Fatal("two recent records should have a previous salary")
	}
}

func TestRequireDateInvalid(t *testing.T) {
	_, vErr := requireDate(rawJSON(t, map[string]any{"d": "31-01-2026"}), "d")
	if vErr == nil || vErr.Field != "d" {
		t.Fatalf("expected error, got %+v", vErr)
	}
}
