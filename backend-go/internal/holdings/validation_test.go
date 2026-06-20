package holdings

import (
	"encoding/json"
	"testing"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

func TestBuildLotInputRejectsQuotedRequiredDecimal(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id":  json.RawMessage(`1`),
		"security_id": json.RawMessage(`2`),
		"side":        json.RawMessage(`"buy"`),
		"quantity":    json.RawMessage(`"10.5"`),
		"price":       json.RawMessage(`100`),
		"date":        json.RawMessage(`"2026-01-02"`),
	}

	_, vErr := buildLotInput(raw)
	assertValidation(t, vErr, "quantity", "must be a number")
}

func TestBuildLotInputRejectsQuotedOptionalDecimal(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id":  json.RawMessage(`1`),
		"security_id": json.RawMessage(`2`),
		"side":        json.RawMessage(`"buy"`),
		"quantity":    json.RawMessage(`10.5`),
		"price":       json.RawMessage(`100`),
		"fee":         json.RawMessage(`"1.25"`),
		"date":        json.RawMessage(`"2026-01-02"`),
	}

	_, vErr := buildLotInput(raw)
	assertValidation(t, vErr, "fee", "must be a number")
}

func TestBuildDividendInputRejectsQuotedOptionalDecimal(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id":      json.RawMessage(`1`),
		"security_id":     json.RawMessage(`2`),
		"pay_date":        json.RawMessage(`"2026-01-02"`),
		"gross_amount":    json.RawMessage(`100`),
		"withholding_tax": json.RawMessage(`"15"`),
	}

	_, vErr := buildDividendInput(raw)
	assertValidation(t, vErr, "withholding_tax", "must be a number")
}

func assertValidation(t *testing.T, got *httputil.ValidationError, field, msg string) {
	t.Helper()
	if got == nil {
		t.Fatal("vErr is nil")
	}
	if got.Field != field || got.Msg != msg {
		t.Fatalf("vErr = %#v, want field=%q msg=%q", got, field, msg)
	}
}
