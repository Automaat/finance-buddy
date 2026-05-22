package auth

import "testing"

func TestPasswordHashVerifyRoundtrip(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !checkPassword(hash, "correct horse") {
		t.Fatal("expected matching password to verify")
	}
	if checkPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestPasswordHashIsSalted(t *testing.T) {
	a, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	b, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if a == b {
		t.Fatal("expected distinct salted hashes for the same password")
	}
}
