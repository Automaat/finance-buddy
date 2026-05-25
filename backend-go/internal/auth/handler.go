package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
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
	Username        string           `json:"username"`
	Password        string           `json:"password"`
	Name            *string          `json:"name"`
	Surname         *string          `json:"surname"`
	PPKEmployeeRate *decimal.Decimal `json:"ppk_employee_rate"`
	PPKEmployerRate *decimal.Decimal `json:"ppk_employer_rate"`
}

type updateUserRequest struct {
	Name            *string          `json:"name"`
	Surname         *string          `json:"surname"`
	PPKEmployeeRate *decimal.Decimal `json:"ppk_employee_rate"`
	PPKEmployerRate *decimal.Decimal `json:"ppk_employer_rate"`
}

type userResponse struct {
	ID              int     `json:"id"`
	Username        string  `json:"username"`
	IsAdmin         bool    `json:"is_admin"`
	Name            *string `json:"name"`
	Surname         *string `json:"surname"`
	PPKEmployeeRate *string `json:"ppk_employee_rate"`
	PPKEmployerRate *string `json:"ppk_employer_rate"`
	CreatedAt       string  `json:"created_at"`
}

// ppkStr renders an optional rate as a quoted 2-decimal string (matches the
// Numeric(5,2) serialization the frontend already expects from personas).
func ppkStr(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.StringFixed(2)
	return &s
}

func toUserResponse(u *User) userResponse {
	return userResponse{
		ID:              u.ID,
		Username:        u.Username,
		IsAdmin:         u.IsAdmin,
		Name:            u.Name,
		Surname:         u.Surname,
		PPKEmployeeRate: ppkStr(u.PPKEmployeeRate),
		PPKEmployerRate: ppkStr(u.PPKEmployerRate),
		CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05.999999"),
	}
}

func derefName(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Login serves POST /api/auth/login (public). On success it sets the session
// cookie and returns the token plus the user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.store.GetByUsername(r.Context(), strings.TrimSpace(req.Username))
	// The user == nil guard short-circuits before checkPassword on the error
	// path; the uniform message avoids revealing whether the username exists.
	if err != nil || user == nil || !checkPassword(user.PasswordHash, req.Password) {
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	ttl := sessionTTL
	if req.RememberMe {
		ttl = rememberTTL
	}
	token, err := h.tokens.Sign(user.ID, user.Username, derefName(user.Name), user.IsAdmin, ttl)
	if err != nil {
		h.logger.Error("sign token", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.setSessionCookie(w, token, req.RememberMe, ttl)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "user": toUserResponse(user)})
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
		httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	user, err := h.store.GetByUsername(r.Context(), claims.Username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}
		h.logger.Error("get user", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// ownerOption is a household member offered as an owner-picker choice.
type ownerOption struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListOwners serves GET /api/users — every household member as an
// owner-picker option (id + display name). Authenticated, not admin-only;
// it carries no sensitive fields. The display name falls back to the
// username when no name is set.
func (h *Handler) ListOwners(w http.ResponseWriter, r *http.Request) {
	owners, err := h.store.ListOwners(r.Context())
	if err != nil {
		h.logger.Error("list owners", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]ownerOption, 0, len(owners))
	for i := range owners {
		name := owners[i].Username
		if owners[i].Name != nil && *owners[i].Name != "" {
			name = *owners[i].Name
		}
		out = append(out, ownerOption{ID: owners[i].ID, Name: name})
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// ListUsers serves GET /api/auth/users (admin only).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list users", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]userResponse, 0, len(users))
	for i := range users {
		out = append(out, toUserResponse(&users[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// CreateUser serves POST /api/auth/users (admin only). The admin sets the
// initial password, display name and PPK rates.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	username, vErr := validateUsername(req.Username)
	if vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	if vErr := validatePassword(req.Password); vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	name, surname, vErr := validateNames(req.Name, req.Surname)
	if vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	if vErr := validatePPKPair(req.PPKEmployeeRate, req.PPKEmployerRate); vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		h.logger.Error("hash password", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	user, err := h.store.Create(r.Context(), CreateParams{
		Username:        username,
		PasswordHash:    hash,
		Name:            name,
		Surname:         surname,
		PPKEmployeeRate: ppkOrDefault(req.PPKEmployeeRate, defaultPPKEmployeeRate),
		PPKEmployerRate: ppkOrDefault(req.PPKEmployerRate, defaultPPKEmployerRate),
	})
	if err != nil {
		if errors.Is(err, ErrNameConflict) {
			httputil.WriteDetailError(w, http.StatusConflict, "Username '"+username+"' already exists")
			return
		}
		h.logger.Error("create user", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toUserResponse(user))
}

// UpdateUser serves PUT /api/auth/users/{id} (admin only). It replaces the
// display name, surname and PPK rates — not username, password or admin flag.
// PUT (not PATCH): the request always carries the full editable set, so an
// omitted field clears the value rather than leaving it unchanged.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid user id")
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	name, surname, vErr := validateNames(req.Name, req.Surname)
	if vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	if vErr := validatePPKPair(req.PPKEmployeeRate, req.PPKEmployerRate); vErr != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, vErr)
		return
	}
	user, err := h.store.Update(r.Context(), id, UpdateParams{
		Name:            name,
		Surname:         surname,
		PPKEmployeeRate: req.PPKEmployeeRate,
		PPKEmployerRate: req.PPKEmployerRate,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "User not found")
			return
		}
		h.logger.Error("update user", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// validateNames trims name + surname; returns the first non-empty error.
func validateNames(rawName, rawSurname *string) (*string, *string, string) {
	name, vErr := optionalName(rawName, "Name")
	if vErr != "" {
		return nil, nil, vErr
	}
	surname, vErr := optionalName(rawSurname, "Surname")
	if vErr != "" {
		return nil, nil, vErr
	}
	return name, surname, ""
}

// validatePPKPair validates both PPK rates; returns the first error message.
func validatePPKPair(employee, employer *decimal.Decimal) string {
	if vErr := validatePPK(employee); vErr != "" {
		return vErr
	}
	return validatePPK(employer)
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
