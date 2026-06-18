package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
	boards := store.NewBoardStore(db)
	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	cfg := config.Config{StaticDir: filepath.Join("..", "..", "web", "out")}
	srv := server.New(cfg, server.Dependencies{Users: users, Boards: boards, Tokens: tokens}, nil)

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

func TestBoardAPIPatchMoveCard(t *testing.T) {
	ts, users := setupTestServer(t)

	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(context.Background(), "owner@example.com", hash, "Owner", domain.RoleUser)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	token, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	createReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/boards", bytesReader(`{"name":"Integration Board"}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	defer createRes.Body.Close()
	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", createRes.StatusCode)
	}

	var board domain.Board
	if err := json.NewDecoder(createRes.Body).Decode(&board); err != nil {
		t.Fatalf("decode board: %v", err)
	}

	board.Columns[0].Cards = []domain.Card{{Title: "Move me", Position: 0}}
	updateBody, err := json.Marshal(board)
	if err != nil {
		t.Fatalf("marshal board: %v", err)
	}
	putReq, err := http.NewRequest(http.MethodPut, ts.URL+"/api/boards/"+board.ID, bytes.NewReader(updateBody))
	if err != nil {
		t.Fatalf("put request: %v", err)
	}
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("If-Match", `"1"`)
	putRes, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("put board: %v", err)
	}
	putRes.Body.Close()
	if putRes.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d", putRes.StatusCode)
	}

	patchReq, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/boards/"+board.ID, bytesReader(`[{"op":"move","from":"/columns/0/cards/0","path":"/columns/1/cards/0"}]`))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("If-Match", `"2"`)
	patchRes, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("patch board: %v", err)
	}
	defer patchRes.Body.Close()
	if patchRes.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", patchRes.StatusCode, readBody(patchRes))
	}

	var patched domain.Board
	if err := json.NewDecoder(patchRes.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patched: %v", err)
	}
	if len(patched.Columns[1].Cards) != 1 || patched.Columns[1].Cards[0].Title != "Move me" {
		t.Fatalf("unexpected patched board: %+v", patched.Columns)
	}
}

func bytesReader(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func readBody(res *http.Response) string {
	b, _ := io.ReadAll(res.Body)
	return string(b)
}
