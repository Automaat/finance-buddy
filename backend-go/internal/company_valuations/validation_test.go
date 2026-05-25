package companyvaluations

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/shopspring/decimal"
)

// numericFields are the keys whose values must travel over the wire as JSON
// numbers (the validator parses raw bytes through decimal.NewFromString to
// preserve precision). When a test supplies a string like "14.50" for these
// keys we treat it as the literal number and emit it without quotes.
var numericFields = map[string]struct{}{
	"fmv_per_share":             {},
	"fmv_low":                   {},
	"fmv_high":                  {},
	"common_stock_discount_pct": {},
}

func raw(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		if _, num := numericFields[k]; num {
			if s, ok := v.(string); ok {
				out[k] = json.RawMessage(s)
				continue
			}
		}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", k, err)
		}
		out[k] = b
	}
	return out
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func validBody(over map[string]any) map[string]any {
	body := map[string]any{
		"company":       "Acme",
		"date":          "2025-06-01",
		"fmv_per_share": "14.50",
		"source":        "409a",
	}
	maps.Copy(body, over)
	return body
}

func TestBuildCreate_HappyPath(t *testing.T) {
	got, vErr := buildCreateRequest(raw(t, validBody(nil)))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if got.Company != "Acme" {
		t.Errorf("Company: want Acme, got %q", got.Company)
	}
	if got.Currency != "USD" {
		t.Errorf("Currency default: want USD, got %q", got.Currency)
	}
	if !got.FMVPerShare.Equal(dec("14.50")) {
		t.Errorf("FMVPerShare: want 14.50, got %s", got.FMVPerShare)
	}
	if got.Source != "409a" {
		t.Errorf("Source: want 409a, got %q", got.Source)
	}
	if got.FMVLow != nil || got.FMVHigh != nil {
		t.Errorf("FMV range: want both nil when omitted")
	}
}

func TestBuildCreate_MissingCompany(t *testing.T) {
	body := validBody(nil)
	delete(body, "company")
	_, vErr := buildCreateRequest(raw(t, body))
	if vErr == nil || vErr.Field != "company" {
		t.Errorf("want company error, got %+v", vErr)
	}
}

func TestBuildCreate_BlankCompany(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"company": "   "})))
	if vErr == nil || vErr.Msg != "Company cannot be empty" {
		t.Errorf("want blank-company error, got %+v", vErr)
	}
}

func TestBuildCreate_BadDateFormat(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"date": "06/01/2025"})))
	if vErr == nil || vErr.Field != "date" {
		t.Errorf("want date error, got %+v", vErr)
	}
}

func TestBuildCreate_InvalidSource(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"source": "rumor"})))
	if vErr == nil || vErr.Field != "source" {
		t.Errorf("want source error, got %+v", vErr)
	}
}

func TestBuildCreate_InvalidCurrency(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"currency": "JPY"})))
	if vErr == nil || vErr.Field != "currency" {
		t.Errorf("want currency error, got %+v", vErr)
	}
}

func TestBuildCreate_AllValidCurrenciesAccepted(t *testing.T) {
	for _, c := range []string{"CHF", "EUR", "GBP", "PLN", "USD"} {
		got, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"currency": c})))
		if vErr != nil {
			t.Errorf("currency %s: unexpected err %+v", c, vErr)
		}
		if got.Currency != c {
			t.Errorf("currency %s: want %s, got %s", c, c, got.Currency)
		}
	}
}

func TestBuildCreate_NegativeFMVRejected(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"fmv_per_share": "-1"})))
	if vErr == nil || vErr.Field != "fmv_per_share" {
		t.Errorf("want fmv_per_share error, got %+v", vErr)
	}
}

func TestBuildCreate_FMVLowExceedsFMVPerShareRejected(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{
		"fmv_per_share": "10",
		"fmv_low":       "15",
	})))
	if vErr == nil || vErr.Field != "fmv_low" {
		t.Errorf("want fmv_low exceed error, got %+v", vErr)
	}
}

func TestBuildCreate_FMVHighBelowFMVPerShareRejected(t *testing.T) {
	_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{
		"fmv_per_share": "10",
		"fmv_high":      "5",
	})))
	if vErr == nil || vErr.Field != "fmv_high" {
		t.Errorf("want fmv_high below error, got %+v", vErr)
	}
}

func TestBuildCreate_FMVRangeBracketAccepted(t *testing.T) {
	got, vErr := buildCreateRequest(raw(t, validBody(map[string]any{
		"fmv_per_share": "10",
		"fmv_low":       "8",
		"fmv_high":      "12",
	})))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if got.FMVLow == nil || !got.FMVLow.Equal(dec("8")) {
		t.Errorf("FMVLow: want 8, got %v", got.FMVLow)
	}
	if got.FMVHigh == nil || !got.FMVHigh.Equal(dec("12")) {
		t.Errorf("FMVHigh: want 12, got %v", got.FMVHigh)
	}
}

func TestBuildCreate_DiscountPctOutsideRange(t *testing.T) {
	cases := []string{"-1", "101"}
	for _, v := range cases {
		_, vErr := buildCreateRequest(raw(t, validBody(map[string]any{
			"common_stock_discount_pct": v,
		})))
		if vErr == nil || vErr.Field != "common_stock_discount_pct" {
			t.Errorf("%s: want discount error, got %+v", v, vErr)
		}
	}
}

func TestBuildCreate_DiscountPctBoundariesAccepted(t *testing.T) {
	for _, v := range []string{"0", "100", "50.5"} {
		got, vErr := buildCreateRequest(raw(t, validBody(map[string]any{
			"common_stock_discount_pct": v,
		})))
		if vErr != nil {
			t.Errorf("%s: unexpected err %+v", v, vErr)
			continue
		}
		if got.CommonStockDiscountPct == nil {
			t.Errorf("%s: want set, got nil", v)
		}
	}
}

func TestBuildCreate_NotesOptional(t *testing.T) {
	got, vErr := buildCreateRequest(raw(t, validBody(map[string]any{"notes": "post-tender update"})))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if got.Notes == nil || *got.Notes != "post-tender update" {
		t.Errorf("Notes: want set, got %v", got.Notes)
	}
}

func TestBuildUpdate_NullSkipped(t *testing.T) {
	// All-null patch must produce an empty patch (no fields touched), matching
	// Python's "null means no-op" convention.
	patch, vErr := buildUpdatePatch(raw(t, map[string]any{
		"company": nil, "currency": nil, "source": nil, "notes": nil, "date": nil,
		"fmv_per_share": nil, "fmv_low": nil, "fmv_high": nil,
	}))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if patch.Company != nil || patch.Currency != nil || patch.Source != nil ||
		patch.Notes != nil || patch.Date != nil ||
		patch.FMVPerShare != nil || patch.FMVLow != nil || patch.FMVHigh != nil {
		t.Errorf("want all-nil patch, got %+v", patch)
	}
}

func TestBuildUpdate_PartialApplied(t *testing.T) {
	patch, vErr := buildUpdatePatch(raw(t, map[string]any{
		"company": "NewCo", "fmv_per_share": "20",
	}))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if patch.Company == nil || *patch.Company != "NewCo" {
		t.Errorf("Company: want NewCo, got %v", patch.Company)
	}
	if patch.FMVPerShare == nil || !patch.FMVPerShare.Equal(dec("20")) {
		t.Errorf("FMVPerShare: want 20, got %v", patch.FMVPerShare)
	}
}

func TestBuildUpdate_BlankCompanyRejected(t *testing.T) {
	_, vErr := buildUpdatePatch(raw(t, map[string]any{"company": "   "}))
	if vErr == nil || vErr.Field != "company" {
		t.Errorf("want company error, got %+v", vErr)
	}
}

func TestBuildUpdate_InvalidCurrencyRejected(t *testing.T) {
	_, vErr := buildUpdatePatch(raw(t, map[string]any{"currency": "BTC"}))
	if vErr == nil || vErr.Field != "currency" {
		t.Errorf("want currency error, got %+v", vErr)
	}
}

func TestBuildUpdate_CurrencyUppercased(t *testing.T) {
	patch, vErr := buildUpdatePatch(raw(t, map[string]any{"currency": "eur"}))
	if vErr != nil {
		t.Fatalf("unexpected err: %+v", vErr)
	}
	if patch.Currency == nil || *patch.Currency != "EUR" {
		t.Errorf("Currency: want EUR (upper), got %v", patch.Currency)
	}
}

func TestBuildUpdate_BadDate(t *testing.T) {
	_, vErr := buildUpdatePatch(raw(t, map[string]any{"date": "06-2025"}))
	if vErr == nil || vErr.Field != "date" {
		t.Errorf("want date error, got %+v", vErr)
	}
}

func TestBuildUpdate_NegativeFMVRejected(t *testing.T) {
	_, vErr := buildUpdatePatch(raw(t, map[string]any{"fmv_per_share": "-1"}))
	if vErr == nil || vErr.Field != "fmv_per_share" {
		t.Errorf("want fmv_per_share error, got %+v", vErr)
	}
}

func TestBuildUpdate_DiscountPctOutsideRange(t *testing.T) {
	_, vErr := buildUpdatePatch(raw(t, map[string]any{"common_stock_discount_pct": "150"}))
	if vErr == nil || vErr.Field != "common_stock_discount_pct" {
		t.Errorf("want discount error, got %+v", vErr)
	}
}

func TestIsNull(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`null`, true},
		{`  null  `, true},
		{`"null"`, false},
		{`0`, false},
	}
	for _, c := range cases {
		if got := isNull(json.RawMessage(c.in)); got != c.want {
			t.Errorf("isNull(%q): want %v, got %v", c.in, c.want, got)
		}
	}
}
