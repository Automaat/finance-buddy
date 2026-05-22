package auth

import "strings"

const minPasswordLen = 8

// validateUsername trims and bounds the username. The second return value is
// a non-empty message when validation fails.
func validateUsername(raw string) (string, string) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", "Username cannot be empty"
	}
	if len(u) > 100 {
		return "", "Username too long (max 100 characters)"
	}
	return u, ""
}

// validatePassword returns a non-empty message when the password is too short.
func validatePassword(raw string) string {
	if len(raw) < minPasswordLen {
		return "Password must be at least 8 characters"
	}
	return ""
}
