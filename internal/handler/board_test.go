package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func setupBoardTest(t *testing.T) (*BoardHandler, *store.UserStore, *store.BoardStore, *auth.TokenService, func()) {
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
	handler := NewBoardHandler(boards, users)

	cleanup := func() { db.Close() }
	return handler, users, boards, tokens, cleanup
}

func createUserWithToken(t *testing.T, users *store.UserStore, tokens *auth.TokenService, email string) (*domain.User, string) {
	t.Helper()
	hash, err := auth.HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(context.Background(), email, hash, "Test User", domain.RoleUser)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return user, token
}

func boardRequest(method, boardID, userID string, body []byte) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, "/api/boards/"+boardID, nil)
	} else {
		req = httptest.NewRequest(method, "/api/boards/"+boardID, bytes.NewReader(body))
	}
	ctx := auth.WithUser(req.Context(), auth.AuthUser{ID: userID, Role: domain.RoleUser})
	if boardID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", boardID)
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	}
	return req.WithContext(ctx)
}

func TestBoardCreateAndList(t *testing.T) {
	handler, users, _, _, cleanup := setupBoardTest(t)
	defer cleanup()

	user, _ := createUserWithToken(t, users, auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long")), "owner@example.com")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/boards", jsonBody(map[string]string{"name": "Sprint 1"}))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: user.ID, Role: domain.RoleUser}))
	handler.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: user.ID, Role: domain.RoleUser}))
	handler.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	var listResp struct {
		Boards []domain.BoardSummary `json:"boards"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Boards) != 1 || listResp.Boards[0].Name != "Sprint 1" {
		t.Fatalf("unexpected boards: %+v", listResp.Boards)
	}
}

func TestBoardPatchMoveCard(t *testing.T) {
	handler, users, boards, _, cleanup := setupBoardTest(t)
	defer cleanup()

	owner, _ := createUserWithToken(t, users, auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long")), "owner@example.com")
	ctx := context.Background()

	board, err := boards.Create(ctx, "Move Board", owner.ID)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	board.Columns[0].Cards = []domain.Card{{Title: "Move me", Position: 0}}
	if err := boards.Update(ctx, board); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	loaded, err := boards.GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("load board: %v", err)
	}

	patch := `[{"op":"move","from":"/columns/0/cards/0","path":"/columns/1/cards/0"}]`
	req := boardRequest(http.MethodPatch, board.ID, owner.ID, []byte(patch))
	req.Header.Set("If-Match", strconv.Quote(strconv.Itoa(loaded.Version)))

	rec := httptest.NewRecorder()
	handler.Patch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var updated domain.Board
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode patched board: %v", err)
	}
	if len(updated.Columns[0].Cards) != 0 || len(updated.Columns[1].Cards) != 1 {
		t.Fatalf("unexpected columns after move: %+v", updated.Columns)
	}
	if updated.Columns[1].Cards[0].Title != "Move me" {
		t.Fatalf("card title = %q, want Move me", updated.Columns[1].Cards[0].Title)
	}
}

func TestBoardPatchRequiresIfMatch(t *testing.T) {
	handler, users, boards, _, cleanup := setupBoardTest(t)
	defer cleanup()

	owner, _ := createUserWithToken(t, users, auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long")), "owner@example.com")
	board, err := boards.Create(context.Background(), "Patch Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	req := boardRequest(http.MethodPatch, board.ID, owner.ID, []byte(`[]`))
	rec := httptest.NewRecorder()
	handler.Patch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("patch without If-Match status = %d, want 400", rec.Code)
	}
}

func TestBoardCreateShare(t *testing.T) {
	handler, users, boards, _, cleanup := setupBoardTest(t)
	defer cleanup()

	owner, _ := createUserWithToken(t, users, auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long")), "owner@example.com")
	collab, _ := createUserWithToken(t, users, auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long")), "collab@example.com")

	board, err := boards.Create(context.Background(), "Share Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	bodyReader := jsonBody(map[string]string{"email": collab.Email, "permission": "write"})
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	req := boardRequest(http.MethodPost, board.ID, owner.ID, body)
	rec := httptest.NewRecorder()
	handler.CreateShare(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create share status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
