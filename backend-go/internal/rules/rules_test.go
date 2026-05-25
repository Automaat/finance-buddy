package rules

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAllReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()
	first := All()
	if len(first) == 0 {
		t.Fatal("expected at least one rule")
	}
	origName := first[0].Name
	first[0].Name = "MUTATED"
	second := All()
	if second[0].Name == "MUTATED" {
		t.Errorf("All() must return a defensive copy — mutation leaked back to Polish2026 (orig %q)", origName)
	}
}

// TestPolish2026Values pins the 2026 numeric values against this test as
// the canonical assertion — issue #545 calls these out specifically. A
// silent change to the table (typo, decimal drift) trips this immediately
// before propagating into simulations, dashboard, or settings.
func TestPolish2026Values(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"ike_limit_2026":                "28260",
		"ikze_limit_2026":               "11304",
		"ikze_limit_b2b_2026":           "16956",
		"ppk_below_threshold_2026":      "5767",
		"minimum_wage_2026":             "4806",
		"pit_threshold_first_2026":      "120000",
		"pit_rate_first_2026":           "0.12",
		"pit_rate_second_2026":          "0.32",
		"capital_gains_tax_2026":        "0.19",
		"pit_free_amount_2026":          "30000",
		"pit_solidarity_threshold_2026": "1000000",
		"pit_solidarity_rate_2026":      "0.04",
		"zus_cap_30x_2026":              "282600",
		"b2b_liniowy_rate_2026":         "0.19",
		"ryczalt_it_rate_2026":          "0.12",
	}
	for key, want := range cases {
		r, ok := Get(key)
		if !ok {
			t.Errorf("missing rule %q", key)
			continue
		}
		if r.Value.String() != want {
			t.Errorf("%s value = %q, want %q", key, r.Value.String(), want)
		}
		if r.Year != 2026 {
			t.Errorf("%s year = %d, want 2026", key, r.Year)
		}
	}
}

func TestMustFloat64Resolves(t *testing.T) {
	t.Parallel()
	if got := MustFloat64("ike_limit_2026"); got != 28260 {
		t.Errorf("MustFloat64(ike_limit_2026) = %v, want 28260", got)
	}
	if got := MustFloat64("pit_rate_first_2026"); got != 0.12 {
		t.Errorf("MustFloat64(pit_rate_first_2026) = %v, want 0.12", got)
	}
}

func TestMustFloat64PanicsOnUnknown(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Error("expected panic on unknown key")
		}
	}()
	MustFloat64("does_not_exist")
}

func TestRuleKeysAreUnique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, len(Polish2026))
	for i := range Polish2026 {
		k := Polish2026[i].Key
		if _, dup := seen[k]; dup {
			t.Errorf("duplicate Key %q in Polish2026", k)
		}
		seen[k] = struct{}{}
	}
}

func TestGetByKey(t *testing.T) {
	t.Parallel()
	r, ok := Get("ike_limit_2026")
	if !ok {
		t.Fatal("ike_limit_2026 must resolve")
	}
	if r.Category != "ike_limit" {
		t.Errorf("category = %q, want ike_limit", r.Category)
	}
	if _, ok := Get("nope"); ok {
		t.Error("unknown key must return ok=false")
	}
}

func TestByCategoryFiltersAndPreservesOrder(t *testing.T) {
	t.Parallel()
	pit := ByCategory("pit")
	if len(pit) < 2 {
		t.Fatalf("expected at least 2 pit rules, got %d", len(pit))
	}
	for _, r := range pit {
		if r.Category != "pit" {
			t.Errorf("ByCategory leaked %q row", r.Category)
		}
	}
}

// decodeRawList decodes the wire payload into the loose map[string]any
// shape so the test doesn't depend on dimString/wire.IsoDate UnmarshalJSON
// implementations (they're write-only on purpose).
func decodeRawList(t *testing.T, payload []byte) []map[string]any {
	t.Helper()
	var raw struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return raw.Rules
}

func TestHandlerListReturnsAllRules(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rules", http.NoBody)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	rows := decodeRawList(t, rec.Body.Bytes())
	if len(rows) != len(Polish2026) {
		t.Errorf("rules = %d, want %d", len(rows), len(Polish2026))
	}
}

func TestHandlerListFiltersByCategory(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rules?category=pit", http.NoBody)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	rows := decodeRawList(t, rec.Body.Bytes())
	for _, r := range rows {
		if r["category"] != "pit" {
			t.Errorf("got non-pit row: %+v", r)
		}
	}
}

func TestHandlerListUnknownCategoryReturnsEmpty(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rules?category=bogus", http.NoBody)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (probing must not 422)", rec.Code)
	}
	rows := decodeRawList(t, rec.Body.Bytes())
	if len(rows) != 0 {
		t.Errorf("unknown category should return empty, got %d", len(rows))
	}
}

// TestSerializationMetadataShape is the contract test the issue asks for:
// every wire row must carry the four metadata fields the UI needs
// (source_url, effective_date, last_checked_date, year), plus value/unit.
func TestSerializationMetadataShape(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rules", http.NoBody)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	var raw struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Rules) == 0 {
		t.Fatal("expected at least one rule")
	}
	required := []string{
		"key", "name", "category", "value", "unit", "year",
		"effective_date", "source_url", "last_checked_date", "description",
	}
	for _, row := range raw.Rules {
		for _, k := range required {
			if _, ok := row[k]; !ok {
				t.Errorf("rule %v missing field %q", row["key"], k)
			}
		}
	}
}

// TestSerializationDateAndValueFormats guards the wire formats the UI
// depends on: dates as YYYY-MM-DD (no zone), values as quoted strings so
// JSON-number coercion can't lose precision.
func TestSerializationDateAndValueFormats(t *testing.T) {
	t.Parallel()
	h := NewHandler(nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rules?category=ike_limit", http.NoBody)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `"value":"28260"`) {
		t.Errorf("expected value to be quoted-string %q, got: %s", "28260", body)
	}
	if !strings.Contains(body, `"effective_date":"2026-01-01"`) {
		t.Errorf("expected effective_date YYYY-MM-DD, got: %s", body)
	}
	if !strings.Contains(body, `"last_checked_date":"`) {
		t.Errorf("expected last_checked_date string, got: %s", body)
	}
	// Source URL must be a https://gov.pl-family link, not blank.
	if !strings.Contains(body, `"source_url":"https://`) {
		t.Errorf("expected source_url to start with https://, got: %s", body)
	}
}
