package holdings

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestBuildLotInputAcceptsQuotedDecimals(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id":  json.RawMessage(`1`),
		"security_id": json.RawMessage(`2`),
		"side":        json.RawMessage(`"buy"`),
		"quantity":    json.RawMessage(`"10.5"`),
		"price":       json.RawMessage(`"100"`),
		"fee":         json.RawMessage(`"1.25"`),
		"date":        json.RawMessage(`"2026-01-02"`),
	}

	got, vErr := buildLotInput(raw)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	assertDecimal(t, got.Quantity, "10.5")
	assertDecimal(t, got.Price, "100")
	assertDecimal(t, got.Fee, "1.25")
}

func TestBuildDividendInputAcceptsQuotedDecimals(t *testing.T) {
	raw := map[string]json.RawMessage{
		"account_id":      json.RawMessage(`1`),
		"security_id":     json.RawMessage(`2`),
		"pay_date":        json.RawMessage(`"2026-01-02"`),
		"gross_amount":    json.RawMessage(`"100"`),
		"withholding_tax": json.RawMessage(`"15"`),
	}

	got, vErr := buildDividendInput(raw)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	assertDecimal(t, got.GrossAmount, "100")
	assertDecimal(t, got.WithholdingTax, "15")
	assertDecimal(t, got.Net(), "85")
}

func assertDecimal(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()
	wantDecimal := decimal.RequireFromString(want)
	if !got.Equal(wantDecimal) {
		t.Fatalf("got = %s, want %s", got, wantDecimal)
	}
}
