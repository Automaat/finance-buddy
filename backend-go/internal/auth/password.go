package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// checkPassword reports whether plain matches the stored bcrypt hash.
func checkPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
