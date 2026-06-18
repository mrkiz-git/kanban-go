package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func TestSeedAdminUpdatesExistingPassword(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "seed.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	users := store.NewUserStore(db)
	email := "admin@kanba.local"

	hash1, err := auth.HashPassword("first-password")
	if err != nil {
		t.Fatalf("hash1: %v", err)
	}
	if err := store.SeedAdmin(ctx, users, email, hash1, "Admin One"); err != nil {
		t.Fatalf("seed first: %v", err)
	}

	hash2, err := auth.HashPassword("second-password")
	if err != nil {
		t.Fatalf("hash2: %v", err)
	}
	if err := store.SeedAdmin(ctx, users, email, hash2, "Admin Two"); err != nil {
		t.Fatalf("seed update: %v", err)
	}

	user, storedHash, err := users.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if user.Name != "Admin Two" {
		t.Fatalf("name = %q, want Admin Two", user.Name)
	}
	if !auth.CheckPassword(storedHash, "second-password") {
		t.Fatal("password hash was not updated on re-seed")
	}
	if auth.CheckPassword(storedHash, "first-password") {
		t.Fatal("old password still valid after re-seed")
	}
	if user.Role != domain.RoleAdmin {
		t.Fatalf("role = %q, want admin", user.Role)
	}
}
