package scenarios

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateCreateAcceptsObject(t *testing.T) {
	t.Parallel()
	if err := validateCreate("Plan A", "retirement", json.RawMessage(`{"a":1}`)); err != nil {
		t.Fatalf("expected valid, got %+v", err)
	}
}

func TestValidateCreateEmptyName(t *testing.T) {
	t.Parallel()
	cases := []string{"", "   ", "\t\n"}
	for _, n := range cases {
		if err := validateCreate(n, "retirement", json.RawMessage(`{}`)); err == nil || err.Field != "name" {
			t.Errorf("expected name error for %q, got %+v", n, err)
		}
	}
}

func TestValidateCreateLongName(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", maxNameLen+1)
	if err := validateCreate(long, "retirement", json.RawMessage(`{}`)); err == nil || err.Field != "name" {
		t.Fatalf("expected name error for over-length, got %+v", err)
	}
}

func TestValidateCreateBadKind(t *testing.T) {
	t.Parallel()
	if err := validateCreate("Plan", "bogus", json.RawMessage(`{}`)); err == nil || err.Field != "kind" {
		t.Fatalf("expected kind error, got %+v", err)
	}
}

func TestValidateCreateMissingKind(t *testing.T) {
	t.Parallel()
	if err := validateCreate("Plan", "", json.RawMessage(`{}`)); err == nil || err.Field != "kind" {
		t.Fatalf("expected kind error, got %+v", err)
	}
}

func TestValidateCreateInputsMustBeObject(t *testing.T) {
	t.Parallel()
	cases := []string{`null`, `[]`, `42`, `"x"`, `true`}
	for _, in := range cases {
		err := validateCreate("Plan", "retirement", json.RawMessage(in))
		if err == nil || err.Field != "inputs_json" {
			t.Errorf("expected inputs error for %q, got %+v", in, err)
		}
	}
}

func TestValidateCreateInputsMissing(t *testing.T) {
	t.Parallel()
	if err := validateCreate("Plan", "retirement", nil); err == nil || err.Field != "inputs_json" {
		t.Fatalf("expected inputs error for nil payload, got %+v", err)
	}
}

func TestValidateCreateInputsTooLarge(t *testing.T) {
	t.Parallel()
	// Build a JSON object that exceeds maxInputsBytes.
	pad := strings.Repeat("a", maxInputsBytes)
	body := []byte(`{"x":"` + pad + `"}`)
	if err := validateCreate("Plan", "retirement", body); err == nil || err.Field != "inputs_json" {
		t.Fatalf("expected inputs over-size error, got %+v", err)
	}
}

func TestValidateUpdateEmptyName(t *testing.T) {
	t.Parallel()
	if err := validateUpdate("", json.RawMessage(`{}`)); err == nil || err.Field != "name" {
		t.Fatalf("expected name error, got %+v", err)
	}
}

func TestValidateUpdateInputsMustBeObject(t *testing.T) {
	t.Parallel()
	if err := validateUpdate("Plan", json.RawMessage(`[]`)); err == nil || err.Field != "inputs_json" {
		t.Fatalf("expected inputs error, got %+v", err)
	}
}

func TestValidateCloneNameOptional(t *testing.T) {
	t.Parallel()
	if err := validateCloneName(""); err != nil {
		t.Errorf("empty clone name should pass, got %+v", err)
	}
	if err := validateCloneName("Custom name"); err != nil {
		t.Errorf("valid clone name should pass, got %+v", err)
	}
}

func TestValidateCloneNameTooLong(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", maxNameLen+1)
	if err := validateCloneName(long); err == nil || err.Field != "name" {
		t.Fatalf("expected name length error, got %+v", err)
	}
}
