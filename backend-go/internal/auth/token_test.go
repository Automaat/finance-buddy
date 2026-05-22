package auth

import (
	"testing"
	"time"
)

func TestTokenSignVerifyRoundtrip(t *testing.T) {
	ts := NewTokenService("test-secret")
	token, err := ts.Sign(42, "marcin", "Marcin", true, time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	claims, err := ts.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "marcin" || claims.Name != "Marcin" || !claims.IsAdmin {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestTokenVerifyRejectsExpired(t *testing.T) {
	ts := NewTokenService("test-secret")
	token, err := ts.Sign(1, "ewa", "Ewa", false, -time.Minute)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := ts.Verify(token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestTokenVerifyRejectsWrongSecret(t *testing.T) {
	token, err := NewTokenService("secret-a").Sign(1, "ewa", "Ewa", false, time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := NewTokenService("secret-b").Verify(token); err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}
}

func TestTokenVerifyRejectsGarbage(t *testing.T) {
	if _, err := NewTokenService("test-secret").Verify("not-a-jwt"); err == nil {
		t.Fatal("expected garbage token to be rejected")
	}
}
