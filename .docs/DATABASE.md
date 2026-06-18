# Database

SQLite persistence layer using `modernc.org/sqlite` (pure Go, no CGO). Migrations managed by `github.com/golang-migrate/migrate/v4` embedded in the Go binary.

## Conventions

| Rule | Detail |
|------|--------|
| Driver | `modernc.org/sqlite` via `database/sql` |
| Migrations | SQL files in `migrations/` — `{version}_{name}.up.sql` / `.down.sql` |
| IDs | `TEXT` UUIDs (primary keys) |
| Timestamps | `TEXT` RFC 3339 UTC, set by application layer |
| Booleans | `INTEGER` 0/1 |
| Enums | `TEXT` with `CHECK` constraints |
| Foreign keys | Enabled via `PRAGMA foreign_keys = ON` on every connection |
| File location | `/data/kanba.db` inside container; overridable via `DATABASE_PATH` env |

## Go Integration

```go
package store

import (
    "database/sql"
    _ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
    if err != nil {
        return nil, err
    }
    db.SetMaxOpenConns(1) // SQLite single-writer
    return db, nil
}
```

Migrations run at startup before the HTTP server binds:

```go
//go:embed migrations/*.sql
var migrationFS embed.FS

func Migrate(db *sql.DB) error { /* golang-migrate with iofs source */ }
```

## Schema

### users

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `email` | TEXT | NOT NULL, UNIQUE |
| `password_hash` | TEXT | NOT NULL (bcrypt) |
| `name` | TEXT | NOT NULL |
| `role` | TEXT | NOT NULL, CHECK (`role` IN ('admin', 'user')) |
| `status` | TEXT | NOT NULL DEFAULT 'active', CHECK (`status` IN ('active', 'suspended')) |
| `created_at` | TEXT | NOT NULL |
| `updated_at` | TEXT | NOT NULL |

### boards

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `owner_id` | TEXT | NOT NULL, REFERENCES `users(id)` ON DELETE CASCADE |
| `name` | TEXT | NOT NULL |
| `created_at` | TEXT | NOT NULL |
| `updated_at` | TEXT | NOT NULL |

Unique index: `(owner_id, name)`.

### board_shares

Collaborative sharing (Option B). Grants read or write access to a registered user.

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `board_id` | TEXT | NOT NULL, REFERENCES `boards(id)` ON DELETE CASCADE |
| `user_id` | TEXT | NOT NULL, REFERENCES `users(id)` ON DELETE CASCADE |
| `permission` | TEXT | NOT NULL, CHECK (`permission` IN ('read', 'write')) |
| `created_at` | TEXT | NOT NULL |

Unique index: `(board_id, user_id)`.

### columns

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `board_id` | TEXT | NOT NULL, REFERENCES `boards(id)` ON DELETE CASCADE |
| `title` | TEXT | NOT NULL |
| `position` | INTEGER | NOT NULL |
| `created_at` | TEXT | NOT NULL |
| `updated_at` | TEXT | NOT NULL |

Index: `(board_id, position)`.

### cards

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `column_id` | TEXT | NOT NULL, REFERENCES `columns(id)` ON DELETE CASCADE |
| `title` | TEXT | NOT NULL |
| `description` | TEXT | DEFAULT '' |
| `position` | INTEGER | NOT NULL |
| `created_at` | TEXT | NOT NULL |
| `updated_at` | TEXT | NOT NULL |

Index: `(column_id, position)`.

### attachments

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | TEXT | PRIMARY KEY |
| `card_id` | TEXT | NOT NULL, REFERENCES `cards(id)` ON DELETE CASCADE |
| `filename` | TEXT | NOT NULL |
| `mime_type` | TEXT | NOT NULL |
| `size_bytes` | INTEGER | NOT NULL |
| `storage_path` | TEXT | NOT NULL |
| `created_at` | TEXT | NOT NULL |

Files stored on disk at `/data/attachments/{id}` (path in `storage_path`).

## Entity Relationships

```
users ──< boards (owner_id)
users ──< board_shares >── boards
boards ──< columns ──< cards ──< attachments
```

## Access Resolution Query

Effective board permission for user `U` on board `B`:

1. If `boards.owner_id = U` → `owner`
2. Else if `board_shares` row exists → `read` or `write`
3. Else if caller is `admin` → `owner` (admin override for management)
4. Else → no access (403)

## Seed Data

Migration `002_seed_admin.up.sql` inserts the initial admin (credentials from env):

| Env var | Default |
|---------|---------|
| `ADMIN_EMAIL` | `admin@kanba.local` |
| `ADMIN_PASSWORD` | `changeme` (must change on first login in production) |
| `ADMIN_NAME` | `System Admin` |

## Migration Versions

| Version | Name | Purpose |
|---------|------|---------|
| 001 | `init_schema` | All tables above |
| 002 | `seed_admin` | Initial admin user |

Future migrations add columns/tables without breaking existing versions.

## Repository Layer

Go interfaces in `internal/store/`:

```go
type UserStore interface {
    Create(ctx context.Context, u *domain.User) error
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    GetByID(ctx context.Context, id string) (*domain.User, error)
    List(ctx context.Context, filter UserFilter) ([]domain.User, error)
    Update(ctx context.Context, u *domain.User) error
    Delete(ctx context.Context, id string) error
}

type BoardStore interface {
    Create(ctx context.Context, b *domain.Board, ownerID string) error
    GetByID(ctx context.Context, id string) (*domain.Board, error)
    ListForUser(ctx context.Context, userID string) ([]domain.BoardSummary, error)
    Update(ctx context.Context, b *domain.Board) error
    Delete(ctx context.Context, id string) error
    ApplyPatch(ctx context.Context, boardID string, patch jsonpatch.Patch) (*domain.Board, error)
    Share(ctx context.Context, boardID, userID, permission string) error
    RevokeShare(ctx context.Context, boardID, userID string) error
    ListShares(ctx context.Context, boardID string) ([]domain.BoardShare, error)
}
```

All mutating operations run inside SQLite transactions.

## Cross-References

- Domain types: `BOARD_SCHEMA.md`
- HTTP layer: `API.md`
- Auth & RBAC: `AUTH.md`
