package auth

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateUsernameTrims(t *testing.T) {
	u, msg := validateUsername("  alice  ")
	if msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
	if u != "alice" {
		t.Fatalf("expected alice, got %q", u)
	}
}

func TestValidateUsernameEmpty(t *testing.T) {
	_, msg := validateUsername("   ")
	if msg == "" {
		t.Fatal("expected empty error")
	}
}

func TestValidateUsernameTooLong(t *testing.T) {
	_, msg := validateUsername(strings.Repeat("a", 101))
	if msg == "" {
		t.Fatal("expected too-long error")
	}
}

func TestValidatePasswordTooShort(t *testing.T) {
	if msg := validatePassword("short"); msg == "" {
		t.Fatal("expected error for short password")
	}
}

func TestValidatePasswordOK(t *testing.T) {
	if msg := validatePassword("12345678"); msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
}

func TestOptionalNameNil(t *testing.T) {
	out, msg := optionalName(nil, "first_name")
	if msg != "" || out != nil {
		t.Fatalf("expected nil/empty, got %+v %q", out, msg)
	}
}

func TestOptionalNameEmptyToNil(t *testing.T) {
	v := "   "
	out, msg := optionalName(&v, "first_name")
	if msg != "" || out != nil {
		t.Fatalf("expected nil/empty, got %+v %q", out, msg)
	}
}

func TestOptionalNameTrims(t *testing.T) {
	v := "  Marcin  "
	out, msg := optionalName(&v, "first_name")
	if msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
	if out == nil || *out != "Marcin" {
		t.Fatalf("expected Marcin, got %+v", out)
	}
}

func TestOptionalNameTooLong(t *testing.T) {
	v := strings.Repeat("a", 101)
	_, msg := optionalName(&v, "first_name")
	if msg == "" || !strings.Contains(msg, "first_name") {
		t.Fatalf("expected too-long error mentioning field, got %q", msg)
	}
}

func TestValidatePPKNilOK(t *testing.T) {
	if msg := validatePPK(nil); msg != "" {
		t.Fatalf("nil should be ok, got %q", msg)
	}
}

func TestValidatePPKOutOfRange(t *testing.T) {
	low := decimal.NewFromFloat(0.1)
	if msg := validatePPK(&low); msg == "" {
		t.Fatal("expected error on low rate")
	}
	high := decimal.NewFromFloat(5.0)
	if msg := validatePPK(&high); msg == "" {
		t.Fatal("expected error on high rate")
	}
}

func TestValidatePPKInRange(t *testing.T) {
	rate := decimal.NewFromFloat(2.0)
	if msg := validatePPK(&rate); msg != "" {
		t.Fatalf("in-range should be ok, got %q", msg)
	}
}

func TestPPKOrDefaultReturnsProvided(t *testing.T) {
	rate := decimal.NewFromFloat(3.0)
	fallback := decimal.NewFromFloat(1.5)
	got := ppkOrDefault(&rate, fallback)
	if got == nil || !got.Equal(rate) {
		t.Fatalf("expected provided rate, got %+v", got)
	}
}

func TestPPKOrDefaultReturnsFallback(t *testing.T) {
	fallback := decimal.NewFromFloat(1.5)
	got := ppkOrDefault(nil, fallback)
	if got == nil || !got.Equal(fallback) {
		t.Fatalf("expected fallback, got %+v", got)
	}
}
