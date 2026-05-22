package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Session token lifetimes. "Remember me" extends a login from one day to five.
const (
	sessionTTL  = 24 * time.Hour
	rememberTTL = 5 * 24 * time.Hour
)

// Handler is the HTTP boundary for /api/auth.
type Handler struct {
	store        *Store
	tokens       *TokenService
	logger       *slog.Logger
	cookieSecure bool
}

// NewHandler wires the store, token service, cookie policy and logger.
func NewHandler(store *Store, tokens *TokenService, cookieSecure bool, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, tokens: tokens, cookieSecure: cookieSecure, logger: logger}
}

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt string `json:"created_at"`
}

func toUserResponse(u *User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05.999999"),
	}
}

// Login serves POST /api/auth/login (public). On success it sets the session
// cookie and returns the token plus the user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.store.GetByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || !checkPassword(user.PasswordHash, req.Password) {
		// Uniform message — don't reveal whether the username exists.
		writeDetailError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	ttl := sessionTTL
	if req.RememberMe {
		ttl = rememberTTL
	}
	token, err := h.tokens.Sign(user.ID, user.Username, user.IsAdmin, ttl)
	if err != nil {
		h.logger.Error("sign token", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.setSessionCookie(w, token, req.RememberMe, ttl)
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": toUserResponse(user)})
}

// Logout serves POST /api/auth/logout (public). It clears the session cookie;
// the stateless token itself simply expires.
func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Me serves GET /api/auth/me — the currently authenticated user.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	user, err := h.store.GetByUsername(r.Context(), claims.Username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeDetailError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}
		h.logger.Error("get user", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// ListUsers serves GET /api/auth/users (admin only).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list users", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]userResponse, 0, len(users))
	for i := range users {
		out = append(out, toUserResponse(&users[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateUser serves POST /api/auth/users (admin only). The admin sets the
// initial password.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	username, vErr := validateUsername(req.Username)
	if vErr != "" {
		writeDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	if vErr := validatePassword(req.Password); vErr != "" {
		writeDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		h.logger.Error("hash password", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	user, err := h.store.Create(r.Context(), username, hash)
	if err != nil {
		if errors.Is(err, ErrNameConflict) {
			writeDetailError(w, http.StatusConflict, "Username '"+username+"' already exists")
			return
		}
		h.logger.Error("create user", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, persistent bool, ttl time.Duration) {
	c := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	// Persistent cookie for "remember me"; otherwise a session cookie that
	// the browser drops on close.
	if persistent {
		c.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, c)
}
