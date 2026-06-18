package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/server"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func setupTestServer(t *testing.T) (*httptest.Server, *store.UserStore) {
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
	cfg := config.Config{StaticDir: filepath.Join("..", "..", "web", "out")}
	srv := server.New(cfg, server.Dependencies{Users: users, Tokens: tokens}, nil)

	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})
	return ts, users
}

func TestProtectedBoardsRequiresAuth(t *testing.T) {
	ts, _ := setupTestServer(t)

	res, err := http.Get(ts.URL + "/api/boards")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func TestAdminEndpointRequiresAdminRole(t *testing.T) {
	ts, users := setupTestServer(t)

	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(context.Background(), "user@example.com", hash, "Regular User", domain.RoleUser)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	token, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/users", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.TokenCookieName, Value: token})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}
}

func TestSuspendedUserCannotAccessProtectedRoute(t *testing.T) {
	ts, users := setupTestServer(t)

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

	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	token, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/boards", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.TokenCookieName, Value: token})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}
}
