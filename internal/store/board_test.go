package store_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func openTestDB(t *testing.T) (*store.UserStore, *store.BoardStore, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cleanup := func() { db.Close() }
	return store.NewUserStore(db), store.NewBoardStore(db), cleanup
}

func createTestUser(t *testing.T, users *store.UserStore, email string) *domain.User {
	t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := users.Create(ctx, email, hash, "Test User", domain.RoleUser)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestBoardCreateDefaultColumns(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	board, err := boards.Create(ctx, "Sprint 1", owner.ID)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	if board.Version != 1 {
		t.Fatalf("version = %d, want 1", board.Version)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(board.Columns))
	}
	wantTitles := []string{"To Do", "In Progress", "Done"}
	for i, title := range wantTitles {
		if board.Columns[i].Title != title {
			t.Fatalf("column[%d] title = %q, want %q", i, board.Columns[i].Title, title)
		}
		if board.Columns[i].Position != i {
			t.Fatalf("column[%d] position = %d, want %d", i, board.Columns[i].Position, i)
		}
	}

	loaded, err := boards.GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("get board: %v", err)
	}
	if loaded.Name != "Sprint 1" {
		t.Fatalf("name = %q, want Sprint 1", loaded.Name)
	}
	if len(loaded.Columns) != 3 {
		t.Fatalf("loaded columns = %d, want 3", len(loaded.Columns))
	}
}

func TestBoardDuplicateNamePerOwner(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	if _, err := boards.Create(ctx, "Backlog", owner.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := boards.Create(ctx, "Backlog", owner.ID); err == nil {
		t.Fatal("expected conflict for duplicate board name")
	} else if err != store.ErrConflict {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestBoardListForUserOwnedAndShared(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	viewer := createTestUser(t, users, "viewer@example.com")

	owned, err := boards.Create(ctx, "Owned Board", owner.ID)
	if err != nil {
		t.Fatalf("create owned: %v", err)
	}
	shared, err := boards.Create(ctx, "Shared Board", owner.ID)
	if err != nil {
		t.Fatalf("create shared: %v", err)
	}
	if err := boards.Share(ctx, shared.ID, viewer.ID, domain.SharePermissionRead); err != nil {
		t.Fatalf("share: %v", err)
	}

	ownerList, err := boards.ListForUser(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list owner: %v", err)
	}
	if len(ownerList) != 2 {
		t.Fatalf("owner boards = %d, want 2", len(ownerList))
	}
	for _, summary := range ownerList {
		if summary.Permission != domain.PermissionOwner {
			t.Fatalf("owner permission = %q, want owner", summary.Permission)
		}
	}

	viewerList, err := boards.ListForUser(ctx, viewer.ID)
	if err != nil {
		t.Fatalf("list viewer: %v", err)
	}
	if len(viewerList) != 1 {
		t.Fatalf("viewer boards = %d, want 1", len(viewerList))
	}
	if viewerList[0].ID != shared.ID {
		t.Fatalf("viewer board id = %q, want %q", viewerList[0].ID, shared.ID)
	}
	if viewerList[0].Permission != domain.PermissionRead {
		t.Fatalf("viewer permission = %q, want read", viewerList[0].Permission)
	}
	_ = owned
}

func TestBoardShareRevokeAndList(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	collab := createTestUser(t, users, "collab@example.com")
	board, err := boards.Create(ctx, "Team Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := boards.Share(ctx, board.ID, owner.ID, domain.SharePermissionWrite); err != store.ErrForbidden {
		t.Fatalf("share with owner err = %v, want ErrForbidden", err)
	}
	if err := boards.Share(ctx, board.ID, collab.ID, domain.SharePermissionWrite); err != nil {
		t.Fatalf("share: %v", err)
	}

	shares, err := boards.ListShares(ctx, board.ID)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %d, want 1", len(shares))
	}
	if shares[0].UserEmail != collab.Email {
		t.Fatalf("share email = %q, want %q", shares[0].UserEmail, collab.Email)
	}

	if err := boards.RevokeShare(ctx, board.ID, collab.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	shares, err = boards.ListShares(ctx, board.ID)
	if err != nil {
		t.Fatalf("list shares after revoke: %v", err)
	}
	if len(shares) != 0 {
		t.Fatalf("shares after revoke = %d, want 0", len(shares))
	}
}

func TestBoardResolvePermission(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	writer := createTestUser(t, users, "writer@example.com")
	reader := createTestUser(t, users, "reader@example.com")
	admin := createTestUser(t, users, "admin@example.com")
	outsider := createTestUser(t, users, "outsider@example.com")

	board, err := boards.Create(ctx, "Permissions", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := boards.Share(ctx, board.ID, writer.ID, domain.SharePermissionWrite); err != nil {
		t.Fatalf("share write: %v", err)
	}
	if err := boards.Share(ctx, board.ID, reader.ID, domain.SharePermissionRead); err != nil {
		t.Fatalf("share read: %v", err)
	}

	cases := []struct {
		userID string
		role   domain.UserRole
		want   domain.BoardPermission
		err    error
	}{
		{owner.ID, domain.RoleUser, domain.PermissionOwner, nil},
		{writer.ID, domain.RoleUser, domain.PermissionWrite, nil},
		{reader.ID, domain.RoleUser, domain.PermissionRead, nil},
		{admin.ID, domain.RoleAdmin, domain.PermissionOwner, nil},
		{outsider.ID, domain.RoleUser, "", store.ErrForbidden},
	}

	for _, tc := range cases {
		got, err := boards.ResolvePermission(ctx, tc.userID, tc.role, board.ID)
		if tc.err != nil {
			if err != tc.err {
				t.Fatalf("user %s err = %v, want %v", tc.userID, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("user %s unexpected err: %v", tc.userID, err)
		}
		if got != tc.want {
			t.Fatalf("user %s permission = %q, want %q", tc.userID, got, tc.want)
		}
	}
}

func TestBoardUpdateAndApplyPatch(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	board, err := boards.Create(ctx, "Patch Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	board.Name = "Renamed Board"
	board.Columns[0].Cards = []domain.Card{{
		Title:    "First task",
		Position: 0,
	}}
	if err := boards.Update(ctx, board); err != nil {
		t.Fatalf("update: %v", err)
	}
	if board.Version != 2 {
		t.Fatalf("version after update = %d, want 2", board.Version)
	}

	loaded, err := boards.GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if loaded.Name != "Renamed Board" {
		t.Fatalf("name = %q, want Renamed Board", loaded.Name)
	}
	if len(loaded.Columns[0].Cards) != 1 {
		t.Fatalf("cards = %d, want 1", len(loaded.Columns[0].Cards))
	}
	if loaded.Columns[0].Cards[0].Title != "First task" {
		t.Fatalf("card title = %q, want First task", loaded.Columns[0].Cards[0].Title)
	}

	patchJSON := `[{"op":"replace","path":"/columns/0/cards/0/title","value":"Updated task"}]`
	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	patched, err := boards.ApplyPatch(ctx, board.ID, loaded.Version, patch)
	if err != nil {
		t.Fatalf("apply patch: %v", err)
	}
	if patched.Version != 3 {
		t.Fatalf("patched version = %d, want 3", patched.Version)
	}
	if patched.Columns[0].Cards[0].Title != "Updated task" {
		t.Fatalf("patched title = %q, want Updated task", patched.Columns[0].Cards[0].Title)
	}

	_, err = boards.ApplyPatch(ctx, board.ID, loaded.Version, patch)
	if err != store.ErrConflict {
		t.Fatalf("stale patch err = %v, want ErrConflict", err)
	}
}

func TestBoardApplyPatchMoveCard(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	board, err := boards.Create(ctx, "Move Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	board.Columns[0].Cards = []domain.Card{{Title: "Move me", Position: 0}}
	if err := boards.Update(ctx, board); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	loaded, err := boards.GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	cardID := loaded.Columns[0].Cards[0].ID

	patchJSON := `[{"op":"move","from":"/columns/0/cards/0","path":"/columns/1/cards/0"}]`
	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	patched, err := boards.ApplyPatch(ctx, board.ID, loaded.Version, patch)
	if err != nil {
		t.Fatalf("apply move patch: %v", err)
	}

	found := false
	for _, card := range patched.Columns[1].Cards {
		if card.ID == cardID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("card %q not found in second column after move", cardID)
	}
}

func TestBoardDeleteCascades(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	board, err := boards.Create(ctx, "Delete Me", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	board.Columns[0].Cards = []domain.Card{{Title: "Temp", Position: 0}}
	if err := boards.Update(ctx, board); err != nil {
		t.Fatalf("add card: %v", err)
	}

	if err := boards.Delete(ctx, board.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := boards.GetByID(ctx, board.ID); err != store.ErrNotFound {
		t.Fatalf("get after delete err = %v, want ErrNotFound", err)
	}
}

func TestBoardPatchRoundTripJSON(t *testing.T) {
	ctx := context.Background()
	users, boards, cleanup := openTestDB(t)
	defer cleanup()

	owner := createTestUser(t, users, "owner@example.com")
	board, err := boards.Create(ctx, "JSON Board", owner.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	loaded, err := boards.GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	raw, err := json.Marshal(loaded)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	patch, err := jsonpatch.DecodePatch([]byte(`[{"op":"replace","path":"/name","value":"JSON Updated"}]`))
	if err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	patchedRaw, err := patch.Apply(raw)
	if err != nil {
		t.Fatalf("apply patch to raw: %v", err)
	}
	var patched domain.Board
	if err := json.Unmarshal(patchedRaw, &patched); err != nil {
		t.Fatalf("unmarshal patched: %v", err)
	}
	if patched.Name != "JSON Updated" {
		t.Fatalf("patched name = %q, want JSON Updated", patched.Name)
	}
	_ = board
}
