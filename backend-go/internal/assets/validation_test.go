package assets

import (
	"encoding/json"
	"testing"

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

func TestRequireNameValid(t *testing.T) {
	name, vErr := requireName(rawJSON(t, map[string]any{"name": "  Konto główne  "}))
	if vErr != nil {
		t.Fatalf("unexpected: %+v", vErr)
	}
	if name != "Konto główne" {
		t.Fatalf("expected trimmed name, got %q", name)
	}
}

func TestRequireNameMissing(t *testing.T) {
	_, vErr := requireName(map[string]json.RawMessage{})
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name error, got %+v", vErr)
	}
}

func TestRequireNameNull(t *testing.T) {
	_, vErr := requireName(rawJSON(t, map[string]any{"name": nil}))
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name error, got %+v", vErr)
	}
}

func TestRequireNameEmpty(t *testing.T) {
	_, vErr := requireName(rawJSON(t, map[string]any{"name": "   "}))
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name error, got %+v", vErr)
	}
}

func TestRequireNameNonString(t *testing.T) {
	_, vErr := requireName(rawJSON(t, map[string]any{"name": 42}))
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected name error, got %+v", vErr)
	}
}

func TestOptionalNameAbsentIsNoop(t *testing.T) {
	p, vErr := optionalName(map[string]json.RawMessage{})
	if p != nil || vErr != nil {
		t.Fatalf("expected nil/nil, got %+v err=%+v", p, vErr)
	}
}

func TestOptionalNameNullIsNoop(t *testing.T) {
	p, vErr := optionalName(rawJSON(t, map[string]any{"name": nil}))
	if p != nil || vErr != nil {
		t.Fatalf("expected nil/nil, got %+v err=%+v", p, vErr)
	}
}

func TestOptionalNameEmptyFails(t *testing.T) {
	_, vErr := optionalName(rawJSON(t, map[string]any{"name": " "}))
	if vErr == nil || vErr.Field != "name" {
		t.Fatalf("expected error, got %+v", vErr)
	}
}

func TestOptionalNameTrims(t *testing.T) {
	p, vErr := optionalName(rawJSON(t, map[string]any{"name": "  Cele  "}))
	if vErr != nil || p == nil || *p != "Cele" {
		t.Fatalf("expected trimmed name, got %+v err=%+v", p, vErr)
	}
}

func TestIsNullCases(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"null", true},
		{"  null  ", true},
		{`"null"`, false},
		{"0", false},
	}
	for _, tc := range cases {
		if got := validation.IsNull(json.RawMessage(tc.in)); got != tc.want {
			t.Errorf("validation.IsNull(%q) = %v want %v", tc.in, got, tc.want)
		}
	}
}
