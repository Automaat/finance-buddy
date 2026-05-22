// Package auth implements application login: a users table, password
// hashing, JWT session tokens, and the chi middleware that gates /api.
//
// Users authenticate; they are intentionally separate from `personas`,
// which own financial data. Linking the two is deliberately out of scope.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload: who the user is and whether they are admin.
type Claims struct {
	UserID   int    `json:"uid"`
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// TokenService signs and verifies session JWTs with a single HMAC secret.
type TokenService struct {
	secret []byte
}

// NewTokenService wraps the configured signing secret.
func NewTokenService(secret string) *TokenService {
	return &TokenService{secret: []byte(secret)}
}

// Sign issues a token for the user that expires after ttl. name is the
// display name carried for the UI; it may be empty.
func (t *TokenService) Sign(userID int, username, name string, isAdmin bool, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Name:     name,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a token, returning its claims. Expired or
// tampered tokens yield an error.
func (t *TokenService) Verify(raw string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}
	return &claims, nil
}
