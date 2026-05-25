package goals

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
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

func validReq(t *testing.T) *createRequest {
	t.Helper()
	return &createRequest{
		Name:                "Wakacje",
		TargetAmount:        10000,
		TargetDate:          wire.IsoDate(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)),
		CurrentAmount:       1000,
		MonthlyContribution: 500,
	}
}

func TestValidateCreateOK(t *testing.T) {
	req := validReq(t)
	if err := validateCreate(req); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if req.Name != "Wakacje" {
		t.Fatalf("name not trimmed: %q", req.Name)
	}
}

func TestValidateCreateTrimsName(t *testing.T) {
	req := validReq(t)
	req.Name = "  Wakacje  "
	if err := validateCreate(req); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if req.Name != "Wakacje" {
		t.Fatalf("name should be trimmed, got %q", req.Name)
	}
}

func TestValidateCreateBlankName(t *testing.T) {
	req := validReq(t)
	req.Name = "  "
	if err := validateCreate(req); err == nil || err.Field != "name" {
		t.Fatalf("expected name error, got %+v", err)
	}
}

func TestValidateCreateMissingDate(t *testing.T) {
	req := validReq(t)
	req.TargetDate = wire.IsoDate(time.Time{})
	if err := validateCreate(req); err == nil || err.Field != "target_date" {
		t.Fatalf("expected target_date error, got %+v", err)
	}
}

func TestValidateCreateNonPositiveTarget(t *testing.T) {
	req := validReq(t)
	req.TargetAmount = 0
	if err := validateCreate(req); err == nil || err.Field != "target_amount" {
		t.Fatalf("expected target_amount error, got %+v", err)
	}
}

func TestValidateCreateNegativeCurrent(t *testing.T) {
	req := validReq(t)
	req.CurrentAmount = -1
	if err := validateCreate(req); err == nil || err.Field != "current_amount" {
		t.Fatalf("expected current_amount error, got %+v", err)
	}
}

func TestValidateCreateNegativeMonthly(t *testing.T) {
	req := validReq(t)
	req.MonthlyContribution = -1
	if err := validateCreate(req); err == nil || err.Field != "monthly_contribution" {
		t.Fatalf("expected monthly_contribution error, got %+v", err)
	}
}

func TestValidateCreateInvalidCategory(t *testing.T) {
	req := validReq(t)
	bogus := "bogus"
	req.Category = &bogus
	if err := validateCreate(req); err == nil || err.Field != "category" {
		t.Fatalf("expected category error, got %+v", err)
	}
}

func TestValidateCreateValidCategory(t *testing.T) {
	req := validReq(t)
	cat := "bank"
	req.Category = &cat
	if err := validateCreate(req); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func TestBuildUpdatePatchEmpty(t *testing.T) {
	p, vErr := buildUpdatePatch(map[string]json.RawMessage{})
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if p.Name != nil || p.TargetAmount != nil || p.AccountIDSet || p.CategorySet {
		t.Fatalf("expected zero patch, got %+v", p)
	}
}

func TestBuildUpdatePatchBlankName(t *testing.T) {
	raw := rawJSON(t, map[string]any{"name": "  "})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchClearsAccountID(t *testing.T) {
	raw := rawJSON(t, map[string]any{"account_id": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.AccountIDSet || p.AccountID != nil {
		t.Fatalf("expected AccountIDSet=true, value nil; got %+v", p)
	}
}

func TestBuildUpdatePatchSetsAccountID(t *testing.T) {
	raw := rawJSON(t, map[string]any{"account_id": 42})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.AccountIDSet || p.AccountID == nil || *p.AccountID != 42 {
		t.Fatalf("expected AccountID=42, got %+v", p)
	}
}

func TestBuildUpdatePatchClearsCategory(t *testing.T) {
	raw := rawJSON(t, map[string]any{"category": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.CategorySet || p.Category != nil {
		t.Fatalf("expected CategorySet=true, value nil; got %+v", p)
	}
}

func TestBuildUpdatePatchInvalidCategory(t *testing.T) {
	raw := rawJSON(t, map[string]any{"category": "bogus"})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "category" {
		t.Fatalf("expected category error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchNonPositiveTarget(t *testing.T) {
	raw := rawJSON(t, map[string]any{"target_amount": 0})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "target_amount" {
		t.Fatalf("expected target_amount error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchNegativeCurrent(t *testing.T) {
	raw := rawJSON(t, map[string]any{"current_amount": -1})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "current_amount" {
		t.Fatalf("expected current_amount error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchTargetDateInvalid(t *testing.T) {
	raw := rawJSON(t, map[string]any{"target_date": "31-01-2026"})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "target_date" {
		t.Fatalf("expected target_date error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchTargetDateValid(t *testing.T) {
	raw := rawJSON(t, map[string]any{"target_date": "2027-12-31"})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	want := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	if p.TargetDate == nil || !p.TargetDate.Equal(want) {
		t.Fatalf("date mismatch: %+v want %s", p.TargetDate, want)
	}
}

func TestBuildUpdatePatchSetsTargetAmount(t *testing.T) {
	raw := rawJSON(t, map[string]any{"target_amount": 1234.5})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.TargetAmount == nil || !p.TargetAmount.Equal(decimal.NewFromFloat(1234.5)) {
		t.Fatalf("expected 1234.5, got %+v", p.TargetAmount)
	}
}

func TestBuildUpdatePatchSetsMonthlyContribution(t *testing.T) {
	raw := rawJSON(t, map[string]any{"monthly_contribution": 500})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if p.MonthlyContribution == nil || !p.MonthlyContribution.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("expected 500, got %+v", p.MonthlyContribution)
	}
}
