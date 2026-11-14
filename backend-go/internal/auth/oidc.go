package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

const oidcHTTPTimeout = 10 * time.Second

// Callback failure classes. Each maps to the /login?error= code the login
// page renders.
var (
	errOIDCFailed      = errors.New("oidc_failed")
	errOIDCNoUser      = errors.New("oidc_no_user")
	errOIDCUnavailable = errors.New("oidc_unavailable")
)

// userLookup is the slice of Store the OIDC flow needs.
type userLookup interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// oidcClient is the provider state built from a successful discovery.
type oidcClient struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
}

// OIDCHandler serves /api/auth/oidc/*: single sign-on through an OpenID
// Connect provider (Authelia) for users that already exist locally. client
// stays nil until the first successful discovery.
type OIDCHandler struct {
	cfg          OIDCConfig
	users        userLookup
	tokens       *TokenService
	codec        stateCodec
	cookieSecure bool
	httpClient   *http.Client
	logger       *slog.Logger

	mu     sync.Mutex
	client *oidcClient
}

// NewOIDCHandler wires the OIDC endpoints. Discovery is deferred to the first
// login so the backend starts while the provider is down.
func NewOIDCHandler(cfg OIDCConfig, users userLookup, tokens *TokenService, cookieSecure bool, logger *slog.Logger) *OIDCHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &OIDCHandler{
		cfg:          cfg,
		users:        users,
		tokens:       tokens,
		codec:        newStateCodec(tokens.secret),
		cookieSecure: cookieSecure,
		httpClient:   &http.Client{Timeout: oidcHTTPTimeout},
		logger:       logger,
	}
}

type oidcStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// Status serves GET /api/auth/oidc/status (public).
func (h *OIDCHandler) Status(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, oidcStatusResponse{Enabled: h.cfg.Enabled()})
}

// Login serves GET /api/auth/oidc/login (public). It stores state, nonce and
// the PKCE verifier in a signed cookie and redirects to the provider.
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled() {
		httputil.WriteDetailError(w, http.StatusNotFound, "OIDC login is not configured")
		return
	}
	client, err := h.discover(r.Context())
	if err != nil {
		h.logger.Warn("oidc: provider unavailable", "err", err)
		redirectToLogin(w, r, errOIDCUnavailable)
		return
	}
	st := oidcState{
		State:    rand.Text(),
		Nonce:    rand.Text(),
		Verifier: oauth2.GenerateVerifier(),
		Expires:  time.Now().Add(oidcStateTTL).Unix(),
	}
	value, err := h.codec.seal(st)
	if err != nil {
		h.logger.Error("oidc: seal state", "err", err)
		redirectToLogin(w, r, errOIDCFailed)
		return
	}
	setStateCookie(w, value, int(oidcStateTTL.Seconds()), h.cookieSecure)
	authURL := client.oauth.AuthCodeURL(st.State, oidc.Nonce(st.Nonce), oauth2.S256ChallengeOption(st.Verifier))
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback serves GET /api/auth/oidc/callback (public). It clears the
// single-use state cookie whatever the outcome; on success it issues the same
// session cookie as password login and redirects to the app.
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled() {
		httputil.WriteDetailError(w, http.StatusNotFound, "OIDC login is not configured")
		return
	}
	setStateCookie(w, "", -1, h.cookieSecure)
	user, err := h.authenticate(r)
	if err != nil {
		h.logger.Warn("oidc: login rejected", "err", err)
		redirectToLogin(w, r, err)
		return
	}
	if _, err := startSession(w, h.tokens, h.cookieSecure, user, false); err != nil {
		h.logger.Error("oidc: start session", "err", err)
		redirectToLogin(w, r, errOIDCFailed)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// authenticate validates the callback and returns the matching local user.
// Errors wrap one of the errOIDC* classes.
func (h *OIDCHandler) authenticate(r *http.Request) (*User, error) {
	st, err := h.codec.read(r, time.Now())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errOIDCFailed, err)
	}
	q := r.URL.Query()
	if code := q.Get("error"); code != "" {
		return nil, fmt.Errorf("%w: provider returned error %q", errOIDCFailed, code)
	}
	if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(st.State)) != 1 {
		return nil, fmt.Errorf("%w: state mismatch", errOIDCFailed)
	}
	client, err := h.discover(r.Context())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errOIDCUnavailable, err)
	}
	ctx := oidc.ClientContext(r.Context(), h.httpClient)
	token, err := client.oauth.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(st.Verifier))
	if err != nil {
		if _, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
			return nil, fmt.Errorf("%w: exchange code: %w", errOIDCFailed, err)
		}
		return nil, fmt.Errorf("%w: exchange code: %w", errOIDCUnavailable, err)
	}
	username, err := preferredUsername(ctx, client, token, st.Nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errOIDCFailed, err)
	}
	user, err := h.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("%w: no local user %q", errOIDCNoUser, username)
		}
		return nil, fmt.Errorf("%w: look up user: %w", errOIDCFailed, err)
	}
	return user, nil
}

type usernameClaims struct {
	PreferredUsername string `json:"preferred_username"`
}

// preferredUsername verifies the ID token (signature, issuer, audience,
// expiry, nonce) and returns its preferred_username. Authelia omits profile
// claims from the ID token by default, so it falls back to userinfo for the
// same subject.
func preferredUsername(ctx context.Context, client *oidcClient, token *oauth2.Token, nonce string) (string, error) {
	rawID, ok := token.Extra("id_token").(string)
	if !ok || rawID == "" {
		return "", errors.New("token response has no id_token")
	}
	idToken, err := client.verifier.Verify(ctx, rawID)
	if err != nil {
		return "", fmt.Errorf("verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return "", errors.New("id_token nonce mismatch")
	}
	var claims usernameClaims
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("decode id_token claims: %w", err)
	}
	if claims.PreferredUsername != "" {
		return claims.PreferredUsername, nil
	}
	info, err := client.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return "", fmt.Errorf("userinfo: %w", err)
	}
	if info.Subject != idToken.Subject {
		return "", errors.New("userinfo subject does not match id_token")
	}
	if err := info.Claims(&claims); err != nil {
		return "", fmt.Errorf("decode userinfo claims: %w", err)
	}
	if claims.PreferredUsername == "" {
		return "", errors.New("no preferred_username claim")
	}
	return claims.PreferredUsername, nil
}

// discover fetches the provider metadata once and caches it. Failures are
// not cached, so the next login retries.
func (h *OIDCHandler) discover(ctx context.Context) (*oidcClient, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client != nil {
		return h.client, nil
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, h.httpClient), h.cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	h.client = &oidcClient{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: h.cfg.ClientID}),
		oauth: oauth2.Config{
			ClientID:     h.cfg.ClientID,
			ClientSecret: h.cfg.ClientSecret,
			RedirectURL:  h.cfg.RedirectURL,
			Endpoint:     withBasicAuth(provider.Endpoint()),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}
	return h.client, nil
}

// withBasicAuth pins client_secret_basic instead of oauth2's auto-detection.
func withBasicAuth(e oauth2.Endpoint) oauth2.Endpoint {
	e.AuthStyle = oauth2.AuthStyleInHeader
	return e
}

// redirectToLogin sends the browser back to the login page with the error
// class of err as ?error=. Unclassified errors count as oidc_failed.
func redirectToLogin(w http.ResponseWriter, r *http.Request, err error) {
	code := errOIDCFailed.Error()
	for _, class := range []error{errOIDCNoUser, errOIDCUnavailable} {
		if errors.Is(err, class) {
			code = class.Error()
		}
	}
	http.Redirect(w, r, "/login?error="+code, http.StatusFound)
}
