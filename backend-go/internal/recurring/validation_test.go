package recurring

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
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
