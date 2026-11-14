package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	oidcStateCookie = "fb_oidc_state"
	oidcCookiePath  = "/api/auth/oidc"
	oidcStateTTL    = 10 * time.Minute
)

var (
	errStateMissing  = errors.New("state cookie missing")
	errStateTampered = errors.New("state cookie signature invalid")
	errStateExpired  = errors.New("state cookie expired")
)

// oidcState is the per-login secret material kept in the browser between
// the redirect to the provider and the callback.
type oidcState struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Expires  int64  `json:"exp"`
}

// stateCodec signs and verifies the state cookie with a key derived from the
// JWT secret, so no extra secret has to be configured.
type stateCodec struct {
	key []byte
}

func newStateCodec(jwtSecret []byte) stateCodec {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("finance-buddy oidc state cookie v1"))
	return stateCodec{key: mac.Sum(nil)}
}

func (c stateCodec) sign(payload string) string {
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (c stateCodec) seal(st oidcState) (string, error) {
	raw, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + c.sign(payload), nil
}

func (c stateCodec) open(value string, now time.Time) (oidcState, error) {
	payload, sig, ok := strings.Cut(value, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(c.sign(payload))) {
		return oidcState{}, errStateTampered
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return oidcState{}, fmt.Errorf("decode state: %w", err)
	}
	var st oidcState
	if err := json.Unmarshal(raw, &st); err != nil {
		return oidcState{}, fmt.Errorf("unmarshal state: %w", err)
	}
	if now.Unix() >= st.Expires {
		return oidcState{}, errStateExpired
	}
	return st, nil
}

func (c stateCodec) read(r *http.Request, now time.Time) (oidcState, error) {
	cookie, err := r.Cookie(oidcStateCookie)
	if err != nil || cookie.Value == "" {
		return oidcState{}, errStateMissing
	}
	return c.open(cookie.Value, now)
}

// setStateCookie writes the state cookie; maxAge < 0 deletes it.
func setStateCookie(w http.ResponseWriter, value string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    value,
		Path:     oidcCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
