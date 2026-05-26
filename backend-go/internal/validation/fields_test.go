package validation

import (
	"encoding/json"
	"testing"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

func TestRequiredTrimmedString(t *testing.T) {
	raw := map[string]json.RawMessage{"name": json.RawMessage(`"  Cash  "`)}
	got, vErr := RequiredTrimmedString(raw, "name", "Field required", "Name cannot be empty")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got != "Cash" {
		t.Fatalf("got = %q", got)
	}

	_, vErr = RequiredTrimmedString(map[string]json.RawMessage{}, "name", "Field required", "Name cannot be empty")
	requireValidation(t, vErr, "name", "Field required")

	_, vErr = RequiredTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  "`)},
		"name",
		"Field required",
		"Name cannot be empty",
	)
	requireValidation(t, vErr, "name", "Name cannot be empty")
}

func TestOptionalTrimmedString(t *testing.T) {
	got, vErr := OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  Cash  "`)},
		"name",
		"Name cannot be empty",
	)
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != "Cash" {
		t.Fatalf("got = %v", got)
	}

	got, vErr = OptionalTrimmedString(map[string]json.RawMessage{}, "name", "Name cannot be empty")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`null`)},
		"name",
		"Name cannot be empty",
	)
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	got, vErr = OptionalTrimmedString(
		map[string]json.RawMessage{"name": json.RawMessage(`"  "`)},
		"name",
		"Name cannot be empty",
	)
	if got != nil {
		t.Fatalf("got = %v", got)
	}
	requireValidation(t, vErr, "name", "Name cannot be empty")
}

func TestRequiredDate(t *testing.T) {
	got, vErr := RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`"2026-05-26"`)}, "date")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got.Format("2006-01-02") != "2026-05-26" {
		t.Fatalf("got = %s", got.Format("2006-01-02"))
	}

	_, vErr = RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`"26/05/2026"`)}, "date")
	requireValidation(t, vErr, "date", "must be YYYY-MM-DD")

	_, vErr = RequiredDate(map[string]json.RawMessage{"date": json.RawMessage(`20260526`)}, "date")
	requireValidation(t, vErr, "date", "must be a string")
}

func TestRequiredIntOrNull(t *testing.T) {
	got, vErr := RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`7`)}, "owner_user_id")
	if vErr != nil {
		t.Fatalf("vErr = %#v", vErr)
	}
	if got == nil || *got != 7 {
		t.Fatalf("got = %v", got)
	}

	got, vErr = RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`null`)}, "owner_user_id")
	if vErr != nil || got != nil {
		t.Fatalf("got = %v vErr = %#v", got, vErr)
	}

	_, vErr = RequiredIntOrNull(map[string]json.RawMessage{"owner_user_id": json.RawMessage(`"7"`)}, "owner_user_id")
	requireValidation(t, vErr, "owner_user_id", "must be an integer")
}

func requireValidation(t *testing.T, got *httputil.ValidationError, field, msg string) {
	t.Helper()
	if got == nil {
		t.Fatal("vErr is nil")
	}
	if got.Field != field || got.Msg != msg {
		t.Fatalf("vErr = %#v, want field=%q msg=%q", got, field, msg)
	}
}
