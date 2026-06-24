# Board Schema

Contract for Kanban board domain types. All Go structs use `encoding/json` tags and live in `internal/domain/`.

## Conventions

| Rule | Detail |
|------|--------|
| IDs | `string` UUIDs (`github.com/google/uuid`), serialized as lowercase hyphenated strings |
| Timestamps | `time.Time`, serialized as RFC 3339 UTC (`2006-01-02T15:04:05Z07:00`) |
| Nullable fields | Pointer types (`*string`, `*time.Time`) with `omitempty` |
| Ordering | `Position int` — zero-based, contiguous within a parent (column or board) |
| Text fields | Trimmed; empty string rejected at validation layer |

## Go Types

```go
package domain

import "time"

// UserRole is the system-level RBAC role.
type UserRole string

const (
    RoleAdmin UserRole = "admin"
    RoleUser  UserRole = "user"
)

// UserStatus controls account access.
type UserStatus string

const (
    StatusActive    UserStatus = "active"
    StatusSuspended UserStatus = "suspended"
)

// User is the domain user entity (never expose password_hash).
type User struct {
    ID        string     `json:"id"`
    Email     string     `json:"email"`
    Name      string     `json:"name"`
    Role      UserRole   `json:"role"`
    Status    UserStatus `json:"status"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
}

// SharePermission is the access level in a board share. Excludes "owner" — ownership is not shareable.
type SharePermission string

const (
    SharePermissionWrite SharePermission = "write"
    SharePermissionRead  SharePermission = "read"
)

// BoardShare grants a user access to someone else's board.
type BoardShare struct {
    ID         string          `json:"id"`
    BoardID    string          `json:"boardId"`
    UserID     string          `json:"userId"`
    UserEmail  string          `json:"userEmail"`
    Permission SharePermission `json:"permission"`
    CreatedAt  time.Time       `json:"createdAt"`
}

// BoardPermission is the caller's effective access to a board.
type BoardPermission string

const (
    PermissionOwner BoardPermission = "owner"
    PermissionWrite BoardPermission = "write"
    PermissionRead  BoardPermission = "read"
)

// BoardSummary is returned by list endpoints.
type BoardSummary struct {
    ID         string          `json:"id"`
    Name       string          `json:"name"`
    Permission BoardPermission `json:"permission"`
    Version    int             `json:"version"`
    UpdatedAt  time.Time       `json:"updatedAt"`
}

// Board is the full board document returned by GET /api/boards/:id.
type Board struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Version   int       `json:"version"`
    Columns   []Column  `json:"columns"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// Column groups cards within a board.
type Column struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Position int    `json:"position"`
    Cards    []Card `json:"cards"`
}

// Card is a single Kanban card. Description supports Markdown (rendered client-side).
type Card struct {
    ID          string       `json:"id"`
    Title       string       `json:"title"`
    Description string       `json:"description,omitempty"`
    Position    int          `json:"position"`
    Attachments []Attachment `json:"attachments,omitempty"`
    UpdatedAt   time.Time    `json:"updatedAt"`
}

// Attachment is a file linked to a card.
type Attachment struct {
    ID        string    `json:"id"`
    Filename  string    `json:"filename"`
    MimeType  string    `json:"mimeType"`
    SizeBytes int64     `json:"sizeBytes"`
    URL       string    `json:"url"` // GET /api/attachments/:id
    CreatedAt time.Time `json:"createdAt"`
}
```

## Multi-Board Model

- Each user owns zero or more boards (`boards.owner_id`).
- A user may also access boards shared with them via `board_shares` (see `DATABASE.md`).
- `BoardSummary.Permission` reflects the caller's effective access:
  - `owner` — user created the board
  - `write` — shared with read-write access
  - `read` — shared with read-only access
- Board names are unique per owner (not globally).

## Default Board Layout

New boards are created with three columns:

| Position | Title |
|----------|-------|
| 0 | To Do |
| 1 | In Progress |
| 2 | Done |

## JSON Patch Targets

Granular updates use [RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902) against a board snapshot. Valid patch paths:

| Path pattern | Allowed operations | Notes |
|--------------|-------------------|-------|
| `/columns/{colIdx}/cards/{cardIdx}/title` | `replace` | Max 200 chars |
| `/columns/{colIdx}/cards/{cardIdx}/description` | `replace`, `add`, `remove` | Markdown text, max 10 000 chars |
| `/columns/{colIdx}/cards/{cardIdx}` | `move` | Move card between columns or reorder |
| `/columns/{colIdx}/title` | `replace` | Max 100 chars |
| `/columns/{colIdx}` | `move` | Reorder columns |
| `/name` | `replace` | Board rename, max 100 chars |

`move` operations use RFC 6902 `from` pointing to the source path. The server recalculates `position` values after every patch.

### Retry Contract

Because a concurrent `move` operation shifts array indices, clients must handle `409 Conflict` from `PATCH /api/boards/:id` with this sequence:

1. Re-fetch the full board (`GET /api/boards/:id`) to get the current snapshot and `version`.
2. Recompute all array indices against the fresh snapshot.
3. Re-send the patch with the updated `If-Match: "<version>"` header.

## WebSocket Events

Real-time payloads wrap the changed entity:

```go
type WSEvent struct {
    Type      string    `json:"type"`      // "board.updated" | "card.created" | "card.updated" | "card.moved"
    BoardID   string    `json:"boardId"`
    Payload   any       `json:"payload"`   // Board | Card | CardMove
    Timestamp time.Time `json:"timestamp"`
}

type CardMove struct {
    CardID       string `json:"cardId"`
    FromColumnID string `json:"fromColumnId"`
    ToColumnID   string `json:"toColumnId"`
    Position     int    `json:"position"`
}
```

Clients subscribe per board: `GET /api/boards/:id/ws` (see `API.md`).

### Event Dispatch Rules

| Event type | Emitted when | Payload type |
|------------|--------------|--------------|
| `board.updated` | Full board replaced via PUT | `Board` |
| `card.created` | New card added (REST or MCP `add_card`) | `Card` |
| `card.updated` | Existing card fields changed with no column/position change | `Card` |
| `card.moved` | Card column or position changed (supersedes `card.updated` for that mutation) | `CardMove` |

`card.created` and `card.updated` are mutually exclusive for a given operation. `card.moved` is emitted instead of `card.updated` whenever `column_id` or `position` changes.

## Validation Rules

| Field | Constraint |
|-------|-----------|
| `Board.Name` | 1–100 chars, trimmed |
| `Column.Title` | 1–100 chars, trimmed |
| `Card.Title` | 1–200 chars, trimmed |
| `Card.Description` | 0–10 000 chars |
| `Attachment` | Max 10 MB per file; allowed MIME: `image/*`, `application/pdf`, `text/plain`, `text/markdown` |
| Columns per board | 1–20 |
| Cards per column | 0–500 |

## Cross-References

- Persistence: `DATABASE.md`
- HTTP endpoints: `API.md`
- Access control: `AUTH.md`
- MCP tool payloads: `MCP.md`
