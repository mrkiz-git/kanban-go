# API

REST + WebSocket API served by the Go backend. JSON request/response bodies. Static Next.js export served at `/`.

## Conventions

| Rule | Detail |
|------|--------|
| Base path | `/api` |
| Content-Type | `application/json` unless noted |
| Auth header | `Authorization: Bearer <jwt>` (see `AUTH.md`) |
| IDs in paths | UUID strings |
| Router | `net/http` with `chi` or `echo` (decided at scaffolding) |
| Errors | Consistent `APIError` envelope (below) |
| Pagination | `?limit=50&offset=0` on list endpoints; default limit 50, max 100 |

## Error Envelope

```go
type APIError struct {
    Error struct {
        Code    string `json:"code"`              // machine-readable, e.g. "not_found"
        Message string `json:"message"`           // human-readable
        Details any    `json:"details,omitempty"` // validation field errors
    } `json:"error"`
}
```

| HTTP Status | Code | When |
|-------------|------|------|
| 400 | `bad_request` | Malformed JSON, invalid patch |
| 401 | `unauthorized` | Missing or expired JWT |
| 403 | `forbidden` | Valid JWT but insufficient role/permission |
| 404 | `not_found` | Resource does not exist or caller lacks visiblity |
| 409 | `conflict` | Duplicate email, board name, stale patch |
| 422 | `validation_error` | Field-level validation failure |
| 500 | `internal_error` | Unhandled server error |

## Health

### `GET /api/health`

No auth required.

**Response 200:**

```json
{ "status": "ok", "version": "0.1.0" }
```

---

## Auth Endpoints

See `AUTH.md` for JWT details. Summary:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | None | Create account (role defaults to `user`) |
| POST | `/api/auth/login` | None | Returns JWT |
| GET | `/api/auth/me` | Bearer | Current user profile |
| POST | `/api/auth/refresh` | Bearer | Issue new JWT |

---

## Boards

All board endpoints require authentication. Permission checks per `AUTH.md`.

### `GET /api/boards`

List boards accessible to the caller (owned + shared).

**Response 200:**

```json
{
  "boards": [
    {
      "id": "uuid",
      "name": "Sprint 12",
      "permission": "owner",
      "updatedAt": "2026-06-18T10:00:00Z"
    }
  ]
}
```

### `POST /api/boards`

Requires: any authenticated user.

**Request:**

```json
{ "name": "New Board" }
```

**Response 201:** Full `Board` object (see `BOARD_SCHEMA.md`) with default columns.

### `GET /api/boards/:id`

Requires: `read`, `write`, or `owner` permission.

**Response 200:** Full `Board` object.

### `PUT /api/boards/:id`

Requires: `owner` or `write` permission.

Replaces the entire board document (columns + cards). Prefer `PATCH` for granular updates.

**Request:** Full `Board` object (without `id`/`updatedAt`).

**Response 200:** Updated `Board`.

### `PATCH /api/boards/:id`

Requires: `owner` or `write` permission.

Applies [RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902) JSON Patch. Valid paths defined in `BOARD_SCHEMA.md`.

**Request:**

```json
[
  { "op": "replace", "path": "/columns/0/cards/1/title", "value": "Updated title" },
  { "op": "move", "from": "/columns/0/cards/1", "path": "/columns/1/cards/0" }
]
```

**Response 200:** Updated `Board`.

**Response 409:** Patch conflicts with concurrent modification (ETag mismatch).

Optimistic concurrency: clients send `If-Match: "<board.updatedAt>"` header; server rejects stale patches with 409.

### `DELETE /api/boards/:id`

Requires: `owner` permission only.

**Response 204:** No body.

---

## Sharing

### `GET /api/boards/:id/shares`

Requires: `owner` permission.

**Response 200:**

```json
{
  "shares": [
    {
      "id": "uuid",
      "userId": "uuid",
      "userEmail": "collaborator@example.com",
      "permission": "write",
      "createdAt": "2026-06-18T10:00:00Z"
    }
  ]
}
```

### `POST /api/boards/:id/shares`

Requires: `owner` permission.

**Request:**

```json
{ "email": "collaborator@example.com", "permission": "read" }
```

`permission`: `"read"` or `"write"`.

**Response 201:** Share object.

### `DELETE /api/boards/:id/shares/:userId`

Requires: `owner` permission.

**Response 204:** No body.

---

## Attachments

### `POST /api/cards/:id/attachments`

Requires: `write` or `owner` on the card's board.

`Content-Type: multipart/form-data`, field name `file`.

**Response 201:** `Attachment` object.

### `GET /api/attachments/:id`

Requires: `read`, `write`, or `owner` on the card's board.

**Response 200:** File bytes with appropriate `Content-Type`.

### `DELETE /api/attachments/:id`

Requires: `write` or `owner`.

**Response 204:** No body.

---

## Admin Endpoints

Require `role: admin` (see `AUTH.md`). Implemented in Part 8.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/users` | List all users (paginated) |
| GET | `/api/admin/users/:id` | User detail |
| PUT | `/api/admin/users/:id` | Update role/status |
| DELETE | `/api/admin/users/:id` | Delete user and owned boards |
| GET | `/api/admin/stats` | System-wide board/user counts |

---

## WebSocket — Real-Time Sync

### `GET /api/boards/:id/ws`

Requires: `read`, `write`, or `owner` permission.

Upgrade via `gorilla/websocket`. JWT passed as query param `?token=<jwt>` (browsers cannot set headers on WS handshake).

**Server → Client events:** `WSEvent` (see `BOARD_SCHEMA.md`).

| Event type | Trigger |
|------------|---------|
| `board.updated` | Full board replace (PUT) |
| `card.updated` | JSON Patch applied |
| `card.moved` | Card moved between columns |

**Client → Server:** No client messages in v1. Board changes go through REST; server broadcasts to all subscribers.

Connection limits: 5 concurrent WS connections per user per board.

---

## AI Chat (Part 10)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/chat` | Send message; AI may apply board patches |

**Request:**

```json
{
  "boardId": "uuid",
  "message": "Move the login bug to In Progress"
}
```

**Response 200:**

```json
{
  "reply": "Moved 'Login bug' to In Progress.",
  "patches": [ /* RFC 6902 ops applied, if any */ ]
}
```

---

## Static Files

| Path | Served by |
|------|-----------|
| `/` | Next.js static export (`out/`) |
| `/admin` | Admin panel static route (Part 8) |
| `/_next/*` | Next.js assets |

Go file server falls back to `index.html` for client-side routing.

## Cross-References

- Domain types: `BOARD_SCHEMA.md`
- Database: `DATABASE.md`
- Auth & permissions: `AUTH.md`
- MCP tools: `MCP.md`
