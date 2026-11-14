package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testClientID     = "finance"
	testClientSecret = "client-secret"
	testRedirectURL  = "https://finance.example.test/api/auth/oidc/callback"
	testCode         = "auth-code"
	testAccessToken  = "access-token"
	testSubject      = "subject-123"
	testKeyID        = "test-key"
)

var testRSAKey = sync.OnceValue(func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
})

// fakeProvider is an in-process OIDC provider: discovery, JWKS, token and
// userinfo endpoints, signing ID tokens with an RSA key.
type fakeProvider struct {
	t   *testing.T
	srv *httptest.Server

	mu               sync.Mutex
	discoveryDown    bool
	challenge        string
	idTokenNonce     string
	idTokenUsername  string
	userinfoUsername string
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	fp := &fakeProvider{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", fp.discovery)
	mux.HandleFunc("GET /jwks", fp.jwks)
	mux.HandleFunc("POST /token", fp.token)
	mux.HandleFunc("GET /userinfo", fp.userinfo)
	fp.srv = httptest.NewServer(mux)
	t.Cleanup(fp.srv.Close)
	return fp
}

func (fp *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	fp.mu.Lock()
	down := fp.discoveryDown
	fp.mu.Unlock()
	if down {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	writeTestJSON(w, map[string]any{
		"issuer":                                fp.srv.URL,
		"authorization_endpoint":                fp.srv.URL + "/authorize",
		"token_endpoint":                        fp.srv.URL + "/token",
		"jwks_uri":                              fp.srv.URL + "/jwks",
		"userinfo_endpoint":                     fp.srv.URL + "/userinfo",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (fp *fakeProvider) jwks(w http.ResponseWriter, _ *http.Request) {
	pub := testRSAKey().PublicKey
	writeTestJSON(w, map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": testKeyID, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
}

func (fp *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	fp.mu.Lock()
	challenge, nonce, username := fp.challenge, fp.idTokenNonce, fp.idTokenUsername
	fp.mu.Unlock()
	id, secret, ok := r.BasicAuth()
	if !ok || id != testClientID || secret != testClientSecret {
		w.WriteHeader(http.StatusUnauthorized)
		writeTestJSON(w, map[string]string{"error": "invalid_client"})
		return
	}
	sum := sha256.Sum256([]byte(r.PostFormValue("code_verifier")))
	if r.PostFormValue("grant_type") != "authorization_code" ||
		r.PostFormValue("code") != testCode ||
		r.PostFormValue("redirect_uri") != testRedirectURL ||
		base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
		w.WriteHeader(http.StatusBadRequest)
		writeTestJSON(w, map[string]string{"error": "invalid_grant"})
		return
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": fp.srv.URL, "aud": testClientID, "sub": testSubject,
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "nonce": nonce,
	}
	if username != "" {
		claims["preferred_username"] = username
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKeyID
	idToken, err := tok.SignedString(testRSAKey())
	if err != nil {
		fp.t.Errorf("sign id token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeTestJSON(w, map[string]any{
		"access_token": testAccessToken, "token_type": "Bearer",
		"expires_in": 300, "id_token": idToken,
	})
}

func (fp *fakeProvider) userinfo(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+testAccessToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	fp.mu.Lock()
	username := fp.userinfoUsername
	fp.mu.Unlock()
	writeTestJSON(w, map[string]string{"sub": testSubject, "preferred_username": username})
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

type fakeUsers map[string]*User

func (f fakeUsers) GetByUsername(_ context.Context, username string) (*User, error) {
	if u, ok := f[username]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func testOIDCConfig(issuer string) OIDCConfig {
	return OIDCConfig{
		IssuerURL:    issuer,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirectURL,
	}
}

func newTestOIDCHandler(issuer string, tokens *TokenService) *OIDCHandler {
	users := fakeUsers{"mskalski": {ID: 7, Username: "mskalski", IsAdmin: true}}
	return NewOIDCHandler(testOIDCConfig(issuer), users, tokens, true, nil)
}

func findCookie(t *testing.T, res *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// startLogin runs /login and returns the state cookie and authorize URL.
func startLogin(t *testing.T, h *OIDCHandler) (*http.Cookie, url.Values) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
	res := rec.Result()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("login status = %d, want 302", res.StatusCode)
	}
	loc, err := url.Parse(res.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	cookie := findCookie(t, res, oidcStateCookie)
	if cookie == nil {
		t.Fatal("login did not set the state cookie")
	}
	return cookie, loc.Query()
}

func callback(t *testing.T, h *OIDCHandler, query url.Values, cookie *http.Cookie) *http.Response {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/callback?"+query.Encode(), http.NoBody)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.Callback(rec, req)
	return rec.Result()
}

func TestOIDCLoginRedirectsWithPKCEAndStateCookie(t *testing.T) {
	fp := newFakeProvider(t)
	h := newTestOIDCHandler(fp.srv.URL, NewTokenService("jwt-secret"))

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
	res := rec.Result()
	loc := res.Header.Get("Location")
	if res.StatusCode != http.StatusFound || !strings.HasPrefix(loc, fp.srv.URL+"/authorize?") {
		t.Fatalf("status %d, Location %q", res.StatusCode, loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	q := u.Query()
	want := map[string]string{
		"response_type":         "code",
		"client_id":             testClientID,
		"redirect_uri":          testRedirectURL,
		"scope":                 "openid profile email",
		"code_challenge_method": "S256",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	for _, k := range []string{"state", "nonce", "code_challenge"} {
		if q.Get(k) == "" {
			t.Errorf("%s missing from authorize URL", k)
		}
	}

	c := findCookie(t, res, oidcStateCookie)
	if c == nil {
		t.Fatal("state cookie not set")
	}
	if !c.HttpOnly || !c.Secure || c.Path != "/api/auth/oidc" ||
		c.SameSite != http.SameSiteLaxMode || c.MaxAge != 600 {
		t.Fatalf("state cookie attributes: %+v", c)
	}
	if strings.Contains(c.Value, q.Get("state")) || strings.Contains(c.Value, "{") {
		t.Fatal("state cookie must be encoded and signed, not raw")
	}
	st, err := h.codec.open(c.Value, time.Now())
	if err != nil || st.State != q.Get("state") || st.Nonce != q.Get("nonce") {
		t.Fatalf("cookie payload %+v, err %v", st, err)
	}
}

func TestOIDCCallback(t *testing.T) {
	tests := []struct {
		name             string
		idTokenUsername  string
		userinfoUsername string
		mutateQuery      func(url.Values)
		mutateCookie     func(*http.Cookie) *http.Cookie
		wrongNonce       bool
		wantLocation     string
	}{
		{
			name:            "existing user from id_token",
			idTokenUsername: "mskalski",
			wantLocation:    "/",
		},
		{
			name:             "existing user from userinfo fallback",
			userinfoUsername: "mskalski",
			wantLocation:     "/",
		},
		{
			name:            "unknown user",
			idTokenUsername: "stranger",
			wantLocation:    "/login?error=oidc_no_user",
		},
		{
			name:            "bad state",
			idTokenUsername: "mskalski",
			mutateQuery:     func(q url.Values) { q.Set("state", "forged") },
			wantLocation:    "/login?error=oidc_failed",
		},
		{
			name:            "nonce mismatch",
			idTokenUsername: "mskalski",
			wrongNonce:      true,
			wantLocation:    "/login?error=oidc_failed",
		},
		{
			name:            "missing cookie",
			idTokenUsername: "mskalski",
			mutateCookie:    func(*http.Cookie) *http.Cookie { return nil },
			wantLocation:    "/login?error=oidc_failed",
		},
		{
			name:            "tampered cookie",
			idTokenUsername: "mskalski",
			mutateCookie: func(c *http.Cookie) *http.Cookie {
				c.Value = "x" + c.Value
				return c
			},
			wantLocation: "/login?error=oidc_failed",
		},
		{
			name:            "provider error response",
			idTokenUsername: "mskalski",
			mutateQuery:     func(q url.Values) { q.Set("error", "access_denied") },
			wantLocation:    "/login?error=oidc_failed",
		},
		{
			name:            "wrong code rejected by token endpoint",
			idTokenUsername: "mskalski",
			mutateQuery:     func(q url.Values) { q.Set("code", "other") },
			wantLocation:    "/login?error=oidc_failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := newFakeProvider(t)
			tokens := NewTokenService("jwt-secret")
			h := newTestOIDCHandler(fp.srv.URL, tokens)

			stateCookie, authz := startLogin(t, h)
			fp.mu.Lock()
			fp.challenge = authz.Get("code_challenge")
			fp.idTokenNonce = authz.Get("nonce")
			if tt.wrongNonce {
				fp.idTokenNonce = "other-nonce"
			}
			fp.idTokenUsername = tt.idTokenUsername
			fp.userinfoUsername = tt.userinfoUsername
			fp.mu.Unlock()

			query := url.Values{"code": {testCode}, "state": {authz.Get("state")}}
			if tt.mutateQuery != nil {
				tt.mutateQuery(query)
			}
			if tt.mutateCookie != nil {
				stateCookie = tt.mutateCookie(stateCookie)
			}
			res := callback(t, h, query, stateCookie)

			if res.StatusCode != http.StatusFound {
				t.Fatalf("status = %d, want 302", res.StatusCode)
			}
			if got := res.Header.Get("Location"); got != tt.wantLocation {
				t.Fatalf("Location = %q, want %q", got, tt.wantLocation)
			}
			cleared := findCookie(t, res, oidcStateCookie)
			if cleared == nil || cleared.MaxAge >= 0 || cleared.Path != oidcCookiePath {
				t.Fatalf("state cookie not cleared: %+v", cleared)
			}
			session := findCookie(t, res, CookieName)
			if tt.wantLocation != "/" {
				if session != nil {
					t.Fatalf("failed login must not set a session: %+v", session)
				}
				return
			}
			assertSessionAuthenticates(t, tokens, session, "mskalski")
		})
	}
}

// assertSessionAuthenticates checks the session cookie matches password
// login's attributes and passes the Authenticate middleware.
func assertSessionAuthenticates(t *testing.T, tokens *TokenService, session *http.Cookie, wantUser string) {
	t.Helper()
	if session == nil {
		t.Fatal("no session cookie set")
	}
	if session.Path != "/" || !session.HttpOnly || !session.Secure ||
		session.SameSite != http.SameSiteLaxMode || session.MaxAge != 0 {
		t.Fatalf("session cookie attributes differ from password login: %+v", session)
	}
	protected := Authenticate(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(claims.Username))
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/accounts", http.NoBody)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != wantUser {
		t.Fatalf("protected endpoint: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestOIDCProviderDown(t *testing.T) {
	down := httptest.NewServer(http.NotFoundHandler())
	issuer := down.URL
	down.Close()
	tokens := NewTokenService("jwt-secret")

	t.Run("login", func(t *testing.T) {
		h := newTestOIDCHandler(issuer, tokens)
		rec := httptest.NewRecorder()
		h.Login(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
		if got := rec.Header().Get("Location"); rec.Code != http.StatusFound || got != "/login?error=oidc_unavailable" {
			t.Fatalf("status %d, Location %q", rec.Code, got)
		}
	})

	t.Run("callback", func(t *testing.T) {
		h := newTestOIDCHandler(issuer, tokens)
		value, err := h.codec.seal(oidcState{
			State: "st", Nonce: "n", Verifier: "v", Expires: time.Now().Add(time.Minute).Unix(),
		})
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		res := callback(t, h, url.Values{"code": {testCode}, "state": {"st"}},
			&http.Cookie{Name: oidcStateCookie, Value: value})
		if got := res.Header.Get("Location"); got != "/login?error=oidc_unavailable" {
			t.Fatalf("Location = %q", got)
		}
		if findCookie(t, res, CookieName) != nil {
			t.Fatal("session set while provider down")
		}
	})
}

func TestOIDCDiscoveryRetriesAfterFailure(t *testing.T) {
	fp := newFakeProvider(t)
	fp.discoveryDown = true
	h := newTestOIDCHandler(fp.srv.URL, NewTokenService("jwt-secret"))

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/login", http.NoBody))
	if got := rec.Header().Get("Location"); got != "/login?error=oidc_unavailable" {
		t.Fatalf("Location while down = %q", got)
	}

	fp.mu.Lock()
	fp.discoveryDown = false
	fp.mu.Unlock()
	startLogin(t, h)
}

func TestOIDCStateCookieExpires(t *testing.T) {
	codec := newStateCodec([]byte("jwt-secret"))
	value, err := codec.seal(oidcState{State: "s", Expires: time.Now().Add(oidcStateTTL).Unix()})
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := codec.open(value, time.Now().Add(oidcStateTTL+time.Second)); err == nil {
		t.Fatal("expired state cookie accepted")
	}
	other := newStateCodec([]byte("other-secret"))
	if _, err := other.open(value, time.Now()); err == nil {
		t.Fatal("state cookie signed with another key accepted")
	}
}

func TestOIDCDisabled(t *testing.T) {
	h := NewOIDCHandler(OIDCConfig{}, fakeUsers{}, NewTokenService("jwt-secret"), false, nil)

	rec := httptest.NewRecorder()
	h.Status(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/status", http.NoBody))
	if strings.TrimSpace(rec.Body.String()) != `{"enabled":false}` {
		t.Fatalf("status body = %q", rec.Body.String())
	}
	for name, serve := range map[string]http.HandlerFunc{"login": h.Login, "callback": h.Callback} {
		rec := httptest.NewRecorder()
		serve(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/"+name, http.NoBody))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", name, rec.Code)
		}
	}
}

func TestOIDCStatusEnabled(t *testing.T) {
	h := newTestOIDCHandler("https://auth.example.test", NewTokenService("jwt-secret"))
	rec := httptest.NewRecorder()
	h.Status(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/oidc/status", http.NoBody))
	if strings.TrimSpace(rec.Body.String()) != `{"enabled":true}` {
		t.Fatalf("status body = %q", rec.Body.String())
	}
}
