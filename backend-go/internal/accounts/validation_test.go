package accounts

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
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

func TestBuildCreateRequestMinimal(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "Konto główne",
		"type":          "asset",
		"category":      "bank",
		"owner_user_id": 1,
		"purpose":       "general",
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected validation error: %+v", vErr)
	}
	if r.Name != "Konto główne" || r.Type != "asset" || r.Category != "bank" || r.Purpose != "general" {
		t.Fatalf("unexpected request: %+v", r)
	}
	if r.OwnerUserID == nil || *r.OwnerUserID != 1 {
		t.Fatalf("owner mismatch: %+v", r.OwnerUserID)
	}
	if r.Currency != "PLN" {
		t.Fatalf("currency fallback failed: %q", r.Currency)
	}
	if !r.ReceivesContributions {
		t.Fatalf("receives_contributions default should be true")
	}
}

func TestBuildCreateRequestSharedOwner(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "Wspólne",
		"type":          "asset",
		"category":      "bank",
		"owner_user_id": nil,
		"purpose":       "general",
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if r.OwnerUserID != nil {
		t.Fatalf("shared owner should be nil, got %+v", r.OwnerUserID)
	}
}

func TestBuildCreateRequestRequiredFields(t *testing.T) {
	cases := []struct {
		name       string
		omit       string
		wantField  string
		wantSubstr string
	}{
		{"missing name", "name", "name", "Field required"},
		{"missing type", "type", "type", "Field required"},
		{"missing category", "category", "category", "Field required"},
		{"missing owner", "owner_user_id", "owner_user_id", "Field required"},
		{"missing purpose", "purpose", "purpose", "Field required"},
	}
	base := map[string]any{
		"name":          "X",
		"type":          "asset",
		"category":      "bank",
		"owner_user_id": 1,
		"purpose":       "general",
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]any{}
			for k, v := range base {
				if k == tc.omit {
					continue
				}
				input[k] = v
			}
			_, vErr := buildCreateRequest(rawJSON(t, input))
			if vErr == nil {
				t.Fatalf("expected validation error for omit %s", tc.omit)
			}
			if vErr.Field != tc.wantField {
				t.Fatalf("field mismatch: got %q want %q", vErr.Field, tc.wantField)
			}
		})
	}
}

func TestBuildCreateRequestInvalidEnum(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "X",
		"type":          "bogus",
		"category":      "bank",
		"owner_user_id": 1,
		"purpose":       "general",
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "type" {
		t.Fatalf("expected type validation error, got %+v", vErr)
	}
}

func TestBuildCreateRequestBlankName(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "   ",
		"type":          "asset",
		"category":      "bank",
		"owner_user_id": 1,
		"purpose":       "general",
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name validation error, got %+v", vErr)
	}
}

func TestBuildCreateRequestSquareMeters(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "Mieszkanie",
		"type":          "asset",
		"category":      "real_estate",
		"owner_user_id": 1,
		"purpose":       "general",
		"square_meters": 55.5,
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if r.SquareMeters == nil || !r.SquareMeters.Equal(decimal.RequireFromString("55.5")) {
		t.Fatalf("square meters mismatch: %+v", r.SquareMeters)
	}
}

func TestBuildCreateRequestReceivesContributionsFalse(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":                   "Konto",
		"type":                   "asset",
		"category":               "bank",
		"owner_user_id":          1,
		"purpose":                "general",
		"receives_contributions": false,
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if r.ReceivesContributions {
		t.Fatalf("expected receives_contributions=false")
	}
}

func TestBuildCreateRequestExcludedFromFireDefaultFalse(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":          "Konto",
		"type":          "asset",
		"category":      "bank",
		"owner_user_id": 1,
		"purpose":       "general",
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if r.ExcludedFromFire {
		t.Fatalf("excluded_from_fire default should be false")
	}
}

func TestBuildCreateRequestExcludedFromFireTrue(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":               "Mieszkanie",
		"type":               "asset",
		"category":           "real_estate",
		"owner_user_id":      1,
		"purpose":            "general",
		"excluded_from_fire": true,
	})
	r, vErr := buildCreateRequest(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !r.ExcludedFromFire {
		t.Fatalf("expected excluded_from_fire=true")
	}
}

func TestBuildCreateRequestExcludedFromFireNotABool(t *testing.T) {
	raw := rawJSON(t, map[string]any{
		"name":               "Konto",
		"type":               "asset",
		"category":           "bank",
		"owner_user_id":      1,
		"purpose":            "general",
		"excluded_from_fire": "yes",
	})
	_, vErr := buildCreateRequest(raw)
	if vErr == nil || vErr.Field != "excluded_from_fire" {
		t.Fatalf("expected excluded_from_fire validation error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchExcludedFromFire(t *testing.T) {
	raw := rawJSON(t, map[string]any{"excluded_from_fire": true})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if p.ExcludedFromFire == nil || !*p.ExcludedFromFire {
		t.Fatalf("expected ExcludedFromFire pointer to true, got %+v", p.ExcludedFromFire)
	}
}

func TestBuildUpdatePatchExcludedFromFireInvalid(t *testing.T) {
	raw := rawJSON(t, map[string]any{"excluded_from_fire": "no"})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "excluded_from_fire" {
		t.Fatalf("expected validation error for excluded_from_fire, got %+v", vErr)
	}
}

func TestBuildUpdatePatchEmpty(t *testing.T) {
	p, vErr := buildUpdatePatch(map[string]json.RawMessage{})
	if vErr != nil {
		t.Fatalf("empty patch should be valid, got %+v", vErr)
	}
	if p.Name != nil || p.Category != nil || p.OwnerUserIDSet || p.AccountWrapperSet || p.SquareMetersSet || p.ReceivesContributions != nil {
		t.Fatalf("empty patch should be zero-valued, got %+v", p)
	}
}

func TestBuildUpdatePatchOwnerNullClears(t *testing.T) {
	raw := rawJSON(t, map[string]any{"owner_user_id": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.OwnerUserIDSet {
		t.Fatalf("OwnerUserIDSet should be true after explicit null")
	}
	if p.OwnerUserID != nil {
		t.Fatalf("OwnerUserID should be nil after explicit null, got %+v", p.OwnerUserID)
	}
}

func TestBuildUpdatePatchSquareMetersNullClears(t *testing.T) {
	raw := rawJSON(t, map[string]any{"square_meters": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.SquareMetersSet || p.SquareMeters != nil {
		t.Fatalf("expected SquareMetersSet=true and value=nil, got set=%v value=%+v", p.SquareMetersSet, p.SquareMeters)
	}
}

func TestBuildUpdatePatchAccountWrapperNullClears(t *testing.T) {
	raw := rawJSON(t, map[string]any{"account_wrapper": nil})
	p, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		t.Fatalf("unexpected error: %+v", vErr)
	}
	if !p.AccountWrapperSet || p.AccountWrapper != nil {
		t.Fatalf("expected wrapper cleared, got set=%v value=%+v", p.AccountWrapperSet, p.AccountWrapper)
	}
}

func TestBuildUpdatePatchAccountWrapperInvalid(t *testing.T) {
	raw := rawJSON(t, map[string]any{"account_wrapper": "BOGUS"})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "account_wrapper" {
		t.Fatalf("expected account_wrapper validation error, got %+v", vErr)
	}
}

func TestBuildUpdatePatchBlankName(t *testing.T) {
	raw := rawJSON(t, map[string]any{"name": "  "})
	_, vErr := buildUpdatePatch(raw)
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name validation error, got %+v", vErr)
	}
}

func TestIsNullVariants(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"null", true},
		{"  null  ", true},
		{`"null"`, false},
		{"0", false},
		{`""`, false},
	}
	for _, tc := range cases {
		if got := validation.IsNull(json.RawMessage(tc.in)); got != tc.want {
			t.Errorf("validation.IsNull(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestOptionalDecimalParsesAndFails(t *testing.T) {
	good := rawJSON(t, map[string]any{"x": 12.345})
	d, vErr := validation.OptionalDecimal(good, "x")
	if vErr != nil || d == nil || !d.Equal(decimal.RequireFromString("12.345")) {
		t.Fatalf("good parse failed: %+v err=%+v", d, vErr)
	}
	bad := rawJSON(t, map[string]any{"x": "not-a-number"})
	if _, vErr := validation.OptionalDecimal(bad, "x"); vErr == nil {
		t.Fatalf("expected error on bad number")
	}
	missing := map[string]json.RawMessage{}
	if d, vErr := validation.OptionalDecimal(missing, "x"); d != nil || vErr != nil {
		t.Fatalf("missing should be nil/nil, got %+v err=%+v", d, vErr)
	}
}

func TestOptionalBoolFallbacks(t *testing.T) {
	if b, vErr := optionalBoolDefaultTrue(map[string]json.RawMessage{}, "x"); !b || vErr != nil {
		t.Fatalf("missing should fall back to true")
	}
	if _, vErr := optionalBoolDefaultTrue(rawJSON(t, map[string]any{"x": nil}), "x"); vErr == nil {
		t.Fatalf("null should be a validation error")
	}
	b, vErr := optionalBoolDefaultTrue(rawJSON(t, map[string]any{"x": false}), "x")
	if vErr != nil || b {
		t.Fatalf("explicit false should pass through")
	}
}
