package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	appmiddleware "github.com/mrkiz-git/kanba-go/internal/middleware"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func setupBoardPermTest(t *testing.T) (*store.BoardStore, *store.UserStore, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cleanup := func() { db.Close() }
	return store.NewBoardStore(db), store.NewUserStore(db), cleanup
}

func TestRequireBoardPermReadOnlyCannotWrite(t *testing.T) {
	boards, users, cleanup := setupBoardPermTest(t)
	defer cleanup()

	ctx := context.Background()
	owner, err := users.Create(ctx, "owner@example.com", "$2a$12$abcdefghijklmnopqrstuvwxYz012345678901234567890", "Owner", domain.RoleUser)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	reader, err := users.Create(ctx, "reader@example.com", "$2a$12$abcdefghijklmnopqrstuvwxYz012345678901234567890", "Reader", domain.RoleUser)
	if err != nil {
		t.Fatalf("create reader: %v", err)
	}

	board, err := boards.Create(ctx, "Shared", owner.ID)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	if err := boards.Share(ctx, board.ID, reader.ID, domain.SharePermissionRead); err != nil {
		t.Fatalf("share: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/boards/"+board.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", board.ID)
	req = req.WithContext(context.WithValue(
		auth.WithUser(req.Context(), auth.AuthUser{ID: reader.ID, Role: domain.RoleUser}),
		chi.RouteCtxKey, rctx,
	))

	rec := httptest.NewRecorder()
	called := false
	appmiddleware.RequireBoardPerm(appmiddleware.BoardPermConfig{
		Boards: boards,
		Min:    domain.PermissionWrite,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should not be called for read-only user")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestRequireBoardPermWriterCannotDelete(t *testing.T) {
	boards, users, cleanup := setupBoardPermTest(t)
	defer cleanup()

	ctx := context.Background()
	owner, err := users.Create(ctx, "owner@example.com", "$2a$12$abcdefghijklmnopqrstuvwxYz012345678901234567890", "Owner", domain.RoleUser)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	writer, err := users.Create(ctx, "writer@example.com", "$2a$12$abcdefghijklmnopqrstuvwxYz012345678901234567890", "Writer", domain.RoleUser)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}

	board, err := boards.Create(ctx, "Shared", owner.ID)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	if err := boards.Share(ctx, board.ID, writer.ID, domain.SharePermissionWrite); err != nil {
		t.Fatalf("share: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/boards/"+board.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", board.ID)
	req = req.WithContext(context.WithValue(
		auth.WithUser(req.Context(), auth.AuthUser{ID: writer.ID, Role: domain.RoleUser}),
		chi.RouteCtxKey, rctx,
	))

	rec := httptest.NewRecorder()
	appmiddleware.RequireBoardPerm(appmiddleware.BoardPermConfig{
		Boards: boards,
		Min:    domain.PermissionOwner,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
