package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type AuthHandler struct {
	users        *store.UserStore
	tokens       *auth.TokenService
	secureCookie bool
	loginLimiter *loginRateLimiter
}

func NewAuthHandler(users *store.UserStore, tokens *auth.TokenService, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		users:        users,
		tokens:       tokens,
		secureCookie: secureCookie,
		loginLimiter: newLoginRateLimiter(),
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAuthRequest(w, r)
	if !ok {
		return
	}

	if err := validateRegister(req); err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	user, err := h.users.Create(r.Context(), email, hash, name, domain.RoleUser)
	if err != nil {
		if store.IsDuplicateEmail(err) {
			writeAPIError(w, http.StatusConflict, "conflict", "Email is already registered", nil)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	h.issueAuth(w, user, http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.loginLimiter.Allow(clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "bad_request", "Too many login attempts. Try again later.", nil)
		return
	}

	req, ok := decodeAuthRequest(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid email or password", nil)
		return
	}

	user, hash, err := h.users.GetByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid email or password", nil)
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	if !auth.CheckPassword(hash, req.Password) {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid email or password", nil)
		return
	}

	if user.Status == domain.StatusSuspended {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid email or password", nil)
		return
	}

	h.issueAuth(w, user, http.StatusOK)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	user, err := h.users.GetByID(r.Context(), authUser.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	if user.Status == domain.StatusSuspended {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Account suspended", nil)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	user, err := h.users.GetByID(r.Context(), authUser.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	if user.Status == domain.StatusSuspended {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Account suspended", nil)
		return
	}

	h.issueAuth(w, user, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearTokenCookie(w, h.secureCookie)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) issueAuth(w http.ResponseWriter, user *domain.User, status int) {
	token, err := h.tokens.Issue(user)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	setTokenCookie(w, token, h.secureCookie)

	data, err := json.Marshal(authResponse{Token: token, User: *user})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func decodeAuthRequest(w http.ResponseWriter, r *http.Request) (authRequest, bool) {
	var req authRequest
	if err := jsonDecode(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "Malformed JSON", nil)
		return authRequest{}, false
	}
	return req, true
}

func validateRegister(req authRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	password := req.Password

	if !emailRegex.MatchString(email) {
		return errors.New("Invalid email format")
	}
	if len(password) < 8 || len(password) > 128 {
		return errors.New("Password must be 8–128 characters")
	}
	if len(name) < 1 || len(name) > 100 {
		return errors.New("Name must be 1–100 characters")
	}
	return nil
}

func setTokenCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.TokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   int(24 * time.Hour / time.Second),
	})
}

func clearTokenCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.TokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}

func jsonDecode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

type loginRateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	hits   map[string][]time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		window: time.Minute,
		limit:  10,
		hits:   make(map[string][]time.Time),
	}
}

func (l *loginRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	times := l.hits[ip]
	filtered := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= l.limit {
		l.hits[ip] = filtered
		return false
	}

	filtered = append(filtered, now)
	l.hits[ip] = filtered
	return true
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	return host
}
