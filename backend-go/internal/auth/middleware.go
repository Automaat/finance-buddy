package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// CookieName is the session cookie carrying the JWT.
const CookieName = "fb_token"

type ctxKey int

const claimsKey ctxKey = 0

// Authenticate verifies the session JWT — taken from the fb_token cookie or a
// Bearer Authorization header — and stores the claims in the request context.
// Requests without a valid token get 401.
func Authenticate(tokens *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			claims, err := tokens.Verify(raw)
			if err != nil {
				httputil.WriteDetailError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

// RequireAdmin rejects authenticated non-admin users with 403. It must run
// after Authenticate.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(r.Context())
		if !ok || !claims.IsAdmin {
			httputil.WriteDetailError(w, http.StatusForbidden, "Admin privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, prefix) {
		return strings.TrimPrefix(h, prefix)
	}
	return ""
}

func claimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}
