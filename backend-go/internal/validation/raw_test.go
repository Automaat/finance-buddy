package validation

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestIsNull(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`null`, true},
		{`  null `, true},
		{`"null"`, false},
		{`0`, false},
		{`{}`, false},
		{``, false},
	}
	for _, tc := range cases {
		if got := IsNull(json.RawMessage(tc.in)); got != tc.want {
			t.Fatalf("IsNull(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRawString(t *testing.T) {
	s, err := RawString(json.RawMessage(`"hi"`))
	if err != nil || s != "hi" {
		t.Fatalf("got %q err %v", s, err)
	}
	if _, err := RawString(json.RawMessage(`123`)); err == nil {
		t.Fatal("expected error for non-string")
	}
}

func TestRawTrimmedString(t *testing.T) {
	s, err := RawTrimmedString(json.RawMessage(`"  hi  "`))
	if err != nil || s != "hi" {
		t.Fatalf("got %q err %v", s, err)
	}
}

func TestRawDate(t *testing.T) {
	d, err := RawDate(json.RawMessage(`"2026-05-25"`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := d.Format("2006-01-02"); got != "2026-05-25" {
		t.Fatalf("got %s", got)
	}
	if _, err := RawDate(json.RawMessage(`"25/05/2026"`)); err == nil {
		t.Fatal("expected error for invalid date")
	}
	if _, err := RawDate(json.RawMessage(`20260525`)); err == nil {
		t.Fatal("expected error for non-string date")
	}
}

func TestIsRawDateFormatError(t *testing.T) {
	_, err := RawDate(json.RawMessage(`"25/05/2026"`))
	if err == nil || !IsRawDateFormatError(err) {
		t.Fatalf("format err = %v", err)
	}
	_, err = RawDate(json.RawMessage(`20260525`))
	if err == nil || IsRawDateFormatError(err) {
		t.Fatalf("type err = %v", err)
	}
}

func TestRawInt(t *testing.T) {
	n, err := RawInt(json.RawMessage(`42`))
	if err != nil || n != 42 {
		t.Fatalf("got %d err %v", n, err)
	}
	if _, err := RawInt(json.RawMessage(`"42"`)); err == nil {
		t.Fatal("expected error for quoted int")
	}
}

func TestRawDecimal(t *testing.T) {
	d, err := RawDecimal(json.RawMessage(`123.45`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("got %s", d)
	}
	// Whitespace handling.
	d, err = RawDecimal(json.RawMessage(`  1.5  `))
	if err != nil || !d.Equal(decimal.RequireFromString("1.5")) {
		t.Fatalf("ws got %s err %v", d, err)
	}
}

func TestRawDecimalRejectsQuotedString(t *testing.T) {
	// Quoted strings are NOT accepted — callers translate the error
	// into "must be a number" to match pydantic's behavior.
	if _, err := RawDecimal(json.RawMessage(`"123.45"`)); err == nil {
		t.Fatal("expected error for quoted decimal")
	}
}
