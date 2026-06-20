package recurring

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

func TestBuildInputAcceptsQuotedAmount(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id": json.RawMessage(`1`),
		"amount":     json.RawMessage(`"1000.00"`),
		"frequency":  json.RawMessage(`"monthly"`),
		"start_date": json.RawMessage(`"2026-01-01"`),
	}

	got, vErr := buildInput(raw)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	want := decimal.RequireFromString("1000.00")
	if !got.Amount.Equal(want) {
		t.Fatalf("got = %s, want %s", got.Amount, want)
	}
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
