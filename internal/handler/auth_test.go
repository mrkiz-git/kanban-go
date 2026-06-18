package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func setupAuthTest(t *testing.T) (*AuthHandler, *store.UserStore, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	users := store.NewUserStore(db)
	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	handler := NewAuthHandler(users, tokens, false)

	cleanup := func() {
		db.Close()
	}
	return handler, users, cleanup
}

func TestRegisterAndLogin(t *testing.T) {
	handler, _, cleanup := setupAuthTest(t)
	defer cleanup()

	registerBody := map[string]string{
		"email":    "user@example.com",
		"password": "securepass123",
		"name":     "Jane Doe",
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(registerBody))
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var regResp authResponse
	if err := json.NewDecoder(rec.Body).Decode(&regResp); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if regResp.Token == "" {
		t.Fatal("register token is empty")
	}
	if regResp.User.Role != domain.RoleUser {
		t.Fatalf("role = %q, want user", regResp.User.Role)
	}

	cookie := rec.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookie {
		if c.Name == auth.TokenCookieName {
			tokenCookie = c
			break
		}
	}
	if tokenCookie == nil || tokenCookie.Value == "" {
		t.Fatal("kanba_token cookie not set on register")
	}

	loginBody := map[string]string{
		"email":    "user@example.com",
		"password": "securepass123",
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(loginBody))
	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMeRequiresAuth(t *testing.T) {
	handler, users, cleanup := setupAuthTest(t)
	defer cleanup()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	handler.Me(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without auth status = %d, want 401", rec.Code)
	}

	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(context.Background(), "me@example.com", hash, "Me User", domain.RoleUser)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	token, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.TokenCookieName, Value: token})
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}))
	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("me with auth status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var me domain.User
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Email != "me@example.com" {
		t.Fatalf("email = %q", me.Email)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	handler, users, cleanup := setupAuthTest(t)
	defer cleanup()

	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	_, err = users.Create(context.Background(), "user@example.com", hash, "User", domain.RoleUser)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	loginBody := map[string]string{
		"email":    "user@example.com",
		"password": "wrongpassword",
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(loginBody))
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSuspendedUserLogin(t *testing.T) {
	handler, users, cleanup := setupAuthTest(t)
	defer cleanup()

	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(context.Background(), "suspended@example.com", hash, "Suspended", domain.RoleUser)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := users.SetStatus(context.Background(), user.ID, domain.StatusSuspended); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	loginBody := map[string]string{
		"email":    "suspended@example.com",
		"password": "securepass123",
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(loginBody))
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	handler, _, cleanup := setupAuthTest(t)
	defer cleanup()

	body := map[string]string{
		"email":    "dup@example.com",
		"password": "securepass123",
		"name":     "Dup User",
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(body))
	handler.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(body))
	handler.Register(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", rec.Code)
	}
}

func TestPasswordHashedInDatabase(t *testing.T) {
	_, users, cleanup := setupAuthTest(t)
	defer cleanup()

	password := "securepass123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	_, err = users.Create(context.Background(), "hash@example.com", hash, "Hash Test", domain.RoleUser)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, storedHash, err := users.GetByEmail(context.Background(), "hash@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if storedHash == password {
		t.Fatal("password stored in plaintext")
	}
	if !auth.CheckPassword(storedHash, password) {
		t.Fatal("stored hash does not match password")
	}
}

func jsonBody(v any) *bytes.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}
