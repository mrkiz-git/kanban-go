# Branch Status

Track unresolved issues per feature branch. The agent checks this file before starting any plan step.

**Legend:** `- [ ]` open · `- [x]` resolved

---

## main

_Integration branch — parts 1–7 merged for testing and deployment._

Includes: contracts, UI planning docs, scaffolding, frontend base, auth, board database layer, and board REST API.

_No open issues._

---

## part-01-contracts

_Status: complete (post-review fixes applied)_

_No open issues._

Deliverables:
- [x] `.docs/BOARD_SCHEMA.md` — Go domain types, multi-board model, JSON Patch paths, WebSocket events
- [x] `.docs/DATABASE.md` — SQLite schema, migrations, repository interfaces
- [x] `.docs/API.md` — REST, WebSocket, admin, and chat endpoints
- [x] `.docs/AUTH.md` — JWT, RBAC, board-level permissions
- [x] `.docs/MCP.md` — MCP tool schemas (`get_boards`, `read_board`, `add_card`, `update_card`)

Design review fixes (all 10 issues resolved):
- [x] AUTH: Middleware now accepts Bearer header OR httpOnly cookie (was contradictory)
- [x] DATABASE: Admin override is step 1 in access resolution, not step 3 (was defeatable by share row)
- [x] DATABASE: Attachment auth JOIN path defined (was unspecified for headerless routes)
- [x] API + BOARD_SCHEMA: ETag changed from `updatedAt` timestamp to monotonic `version` integer
- [x] MCP: `update_card` gained optional `board_version` concurrency guard
- [x] BOARD_SCHEMA: JSON Patch retry-on-409 contract documented
- [x] BOARD_SCHEMA + DATABASE: `SharePermission` type introduced (excludes `owner`; `Share()` interface updated)
- [x] MCP: `update_card` schema enforces at least one update field via `anyOf`
- [x] MCP: stdio token 24h expiry limitation documented
- [x] BOARD_SCHEMA + API + MCP: `card.created` event added; deterministic dispatch rules for `card.updated` vs `card.moved`

---

## part-02-ui-planning

_Status: not started_

_No open issues._

---

## part-03-scaffolding

_Status: complete_

_No open issues._

Deliverables:
- [x] `go.mod` — module `github.com/mrkiz-git/kanba-go`, chi router
- [x] `cmd/kanba/main.go` + `internal/` — server with `GET /api/health`
- [x] `web/` — minimal Next.js static-export stub for Containerfile frontend stage
- [x] `Containerfile` — multi-stage: Node → Go → Alpine
- [x] `scripts/start.sh` — verbose, background, and container run modes
- [x] `scripts/stop.sh` — stop background server or Podman container
- [x] `internal/logging` — structured logging with `error`, `info`, and `debug` levels

---

## part-04-frontend-base

_Status: complete_

_No open issues._

Deliverables:
- [x] `internal/handler/static.go` — static file server with SPA fallback to `index.html`
- [x] `internal/server/server.go` — serves Next.js export at `/`, API at `/api/*`
- [x] `web/` — Tailwind v4, AppShell, route pages with empty states per `UI.md`
- [x] `scripts/start.sh` — builds frontend before local server start

Code review fixes (all 8 issues resolved):
- [x] Auth forms: submit buttons use `type="submit"`
- [x] Request logger emits at `Debug` level per PLAN.md QA
- [x] Static handler returns 500 on non-notexist `os.Stat` errors
- [x] Background start validates PID belongs to kanba binary
- [x] `middleware.RealIP` omission documented in server.go
- [x] Logger: deduplicated file-open logic in `NewFromConfig`
- [x] Logger: replaced `dirOf` with `filepath.Dir`
- [x] Static handler: removed dead `http.NotFound` branch

---

## part-05-auth

_Status: complete_

_No open issues._

Code review fixes (all 10 issues resolved):
- [x] loginRateLimiter data race — added `sync.Mutex` to rate limiter
- [x] golang-migrate driver mismatch — switched to `database/sqlite` (modernc-compatible)
- [x] Unauthenticated logout (CSRF) — logout moved inside `Auth` middleware group
- [x] extractToken silent fallthrough — malformed `Authorization` header returns 401
- [x] Account enumeration via suspended check — login returns 401 for suspended accounts
- [x] SecureCookie defaults to false — secure cookies default on; `KANBA_INSECURE_COOKIE=1` opts out
- [x] Default admin password has no warning — startup log when `ADMIN_PASSWORD` is default
- [x] SeedAdmin ignores password rotation — updates password/name when admin already exists
- [x] tokenCookieName duplicated — exported `auth.TokenCookieName`
- [x] EmailExists TOCTOU in Register — rely on UNIQUE constraint; return 409 on duplicate

Deliverables:
- [x] SQLite users table with embedded migrations (`internal/store/migrations/`)
- [x] JWT auth APIs: register, login, logout, me, refresh (`internal/handler/auth.go`)
- [x] Auth middleware with Bearer header and `kanba_token` cookie (`internal/middleware/auth.go`)
- [x] Admin role guard on `/api/admin/*` endpoints
- [x] Frontend auth provider, API client, login/register forms
- [x] `AuthGuard` and `AdminGuard` for protected routes
- [x] Seeded admin user from env (`ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_NAME`)

---

## part-06-database

_Status: complete_

_No open issues._

Deliverables:
- [x] `internal/store/migrations/002_init_boards.up.sql` — boards, board_shares, columns, cards, attachments
- [x] `internal/domain/board.go` — board domain types per `BOARD_SCHEMA.md`
- [x] `internal/store/board.go` — BoardStore with CRUD, JSON Patch, shares, permission resolution
- [x] `internal/store/errors.go` — shared store sentinel errors
- [x] `internal/store/board_test.go` — repository tests (create, list, share, patch, delete)

---

## part-07-board-api

_Status: complete_

_No open issues._

Deliverables:
- [x] `internal/handler/board.go` — board CRUD, JSON Patch, and share handlers
- [x] `internal/middleware/board.go` — `RequireBoardPerm` with read/write/owner checks
- [x] `internal/auth/board_context.go` — board access in request context
- [x] `internal/server/server.go` — wired `/api/boards` routes with auth and permission middleware
- [x] `cmd/kanba/main.go` — injects `BoardStore` into server dependencies
- [x] Handler, middleware, and server integration tests including card move via PATCH

---

## part-08-admin-panel

_Status: not started_

_No open issues._

---

## part-09-mcp

_Status: not started_

_No open issues._

---

## part-10-ai-chat

_Status: not started_

_No open issues._

---

## part-11-ai-sidebar

_Status: not started_

_No open issues._
