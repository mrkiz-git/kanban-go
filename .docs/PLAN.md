# Kanban Web App (Go + MCP) Development Plan

This plan adopts the original Kanban web app architecture but transitions the backend to Go, retains Podman containerization, adds an MCP (Model Context Protocol) integration, introduces UI planning, and incorporates advanced user management with an admin panel.

## 3 Improvements (Included)

1. **Real-time Synchronization (WebSockets/SSE)**: The Go backend will push board changes via WebSockets. This ensures that if the AI (or another session) updates the board, the UI updates instantly.
2. **Robust Database Migrations**: We will use a migration tool (e.g., `golang-migrate`) embedded in the Go binary. This provides a safe, versioned path for schema evolution as the app grows.
3. **Granular Card Updates (JSON Patch)**: We will improve the API to support granular updates (e.g., moving a single card or updating its text) to prevent race conditions when multiple actions occur simultaneously.

## 3 Functional Options (Included)

1. **Option A: Multi-Board Support**: Users can create, name, and manage multiple Kanban boards.
2. **Option B: Collaborative Sharing**: Users can grant read-only or read-write access to their boards to other registered users.
3. **Option C: Markdown & Attachments in Cards**: Card descriptions will support rich Markdown rendering, and users can attach files/images to cards.

## User Management & Admin Panel (Included)
1. **RBAC & User Management**: Role-Based Access Control (Admin vs Regular User), handling collaborative sharing permissions.
2. **Admin Panel**: A dedicated UI and API scope for system administrators to manage users (create, suspend, delete), oversee all boards, and configure system-wide settings.

---

## Step-by-Step Development Plan

### Part 1: Plan & Contracts
- Update `BOARD_SCHEMA.md`, `DATABASE.md`, `API.md`, and `AUTH.md` to reflect Go idioms, multi-board relations, and RBAC (Role-Based Access Control).
- Define MCP Server schemas in `MCP.md`.
- **Goal:** All contract docs agree on Go implementation details.
- **QA Testing Criteria:**
  - Verify `BOARD_SCHEMA.md` contains valid multi-board definitions.
  - Verify `API.md` outlines all RBAC endpoints.
  - Verify `AUTH.md` defines the Go JWT structure.

### Part 2: UI Planning
- Create `UI.md` to define design tokens, color palette, wireframes, and component hierarchy.
- Include UI plans for the Admin Panel, multi-board switching, and sharing dialogs.
- **Goal:** User approves `UI.md` specifications.
- **QA Testing Criteria:**
  - Verify `UI.md` clearly defines color tokens.
  - Verify Admin Panel and multi-board switching wireframes exist.
  - Obtain explicit sign-off from the user before proceeding.

### Part 3: Scaffolding (Go + Podman)
- Initialize Go module (`go mod init`).
- Create Go backend structure with a router and `GET /api/health`.
- Update `Containerfile` to use a multi-stage build: Node (Next.js) -> Go -> Alpine.
- Add structured logging with three levels (`error`, `info`, `debug`) via `LOG_LEVEL`.
- Add server run modes: `scripts/start.sh verbose` (foreground logs), `scripts/start.sh background` (daemon with persistent log file), and `scripts/start.sh container` (Podman).
- Add `scripts/stop.sh` to stop a background local server or Podman container.
- **Goal:** `scripts/start.sh` spins up the Go server via Podman or local dev modes with configurable logging.
- **QA Testing Criteria:**
  - Run `go test -race ./...` successfully (even if empty); race detector must be enabled.
  - `GET /api/health` returns HTTP 200 with `Content-Type: application/json` and body `{"status":"ok","version":"..."}`.
  - Verify `scripts/start.sh container` builds and runs the container without errors; `podman ps` shows the container running.
  - Verify `scripts/start.sh verbose` runs in the foreground and prints request logs at `debug` level.
  - Verify `scripts/start.sh background` writes logs to a persistent file under `data/logs/` and `scripts/stop.sh` stops the process cleanly.
  - Verify `LOG_LEVEL=error` suppresses info/debug request logs; `LOG_LEVEL=debug` shows them.
  - Verify `http.Server` configures `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` (inspect server struct in code; absence is a bug — see issue #4).
  - Verify graceful shutdown: send SIGTERM to the container (`podman stop`); the process exits 0 and logs no fatal error (issue #5, #6).
  - Verify the container process runs as a non-root user: `podman exec <container> id` must not show `uid=0`.
  - Verify the port mapping is correct: `curl http://localhost:${PORT}/api/health` returns 200 with the server running inside on its configured port.
  - Confirm `middleware.RealIP` is removed or documented as requiring a trusted proxy before Parts 5+ add IP-based controls (issue #10).

### Part 4: Frontend (Static Base)
- Next.js static export (`output: 'export'`) served by the Go backend at `/`.
- Implement basic routing and empty state layouts.
- **Goal:** Browser hits `/` and sees the base app shell.
- **QA Testing Criteria:**
  - Verify Next.js build (`npm run build`) completes successfully with no TypeScript errors.
  - `GET /` returns HTTP 200 with `Content-Type: text/html` from the Go static file handler.
  - Load `/` in browser and confirm UI shell renders; verify no console errors on initial load.
  - Request a non-existent path (e.g. `GET /does-not-exist`); confirm the Go server returns 404, not a panic or 500.
  - Verify static assets (`/_next/static/...`) are served with appropriate cache headers (`Cache-Control: public, max-age=...`).
  - Verify the Containerfile still builds successfully end-to-end after this part (`scripts/start.sh` still passes).

### Part 5: Auth, Security, & User Management
- Implement JWT generation and validation in Go (`golang-jwt/jwt`).
- Build user management API (Registration, Login, RBAC for Admins).
- **Goal:** Unauthenticated users are redirected to login. Role-based routing is established.
- **QA Testing Criteria:**
  - Verify user registration API creates a valid user; duplicate email returns 409 Conflict.
  - Verify login API returns a valid, parseable JWT with correct claims (`sub`, `role`, `exp`).
  - Verify an expired JWT is rejected with 401 Unauthorized (test with a token whose `exp` is in the past).
  - Verify a tampered JWT signature is rejected with 401 Unauthorized.
  - Attempt to access a protected route without a JWT; confirm 401 Unauthorized, no stack trace in response body.
  - Verify admin endpoints reject non-admin users with 403 Forbidden; verify the response body does not leak role or user details.
  - Confirm `middleware.RealIP` is removed (or a trusted-proxy allowlist is enforced) before any rate limiting or IP-based logic is added — spoofing `X-Forwarded-For` must not bypass controls (issue #10).
  - Run `go test -race ./...`; all auth middleware tests must pass without data races.

### Part 6: Database Modeling & Go Integration
- Integrate pure-Go SQLite (`modernc.org/sqlite`).
- Implement schema migrations (Users, Boards, Permissions/Shares).
- Seed initial admin user.
- **Goal:** Database survives container restarts and supports RBAC.
- **QA Testing Criteria:**
  - Verify the container user (`nobody` or a dedicated `kanba` user) has write permission to the data directory **before** starting the server; `podman exec <container> touch /app/data/test` must succeed (issue #8).
  - Verify the SQLite file is created in a directory mounted as a named volume, not baked into the image layer.
  - Stop and restart the container (`podman stop` + `podman start`); verify the SQLite file and its data persist across the restart.
  - Run all migrations from scratch; confirm each migration is idempotent (running twice produces no error and no duplicate schema objects).
  - Verify the default Admin user is seeded on first run and is **not** re-seeded on subsequent starts.
  - Verify a failed migration (intentionally broken SQL) rolls back cleanly and the server refuses to start with a clear error, rather than starting with a partial schema.
  - Run `go test -race ./...` against all database layer tests; no data races allowed.

### Part 7: Core Board API & Sharing
- `GET`, `POST`, `PUT`, `PATCH`, `DELETE` for multiple boards.
- Implement collaborative sharing endpoints (granting access to users).
- Implement WebSocket broadcasting for real-time sync.
- **Goal:** Full multi-board CRUD and sharing persists to SQLite.
- **QA Testing Criteria:**
  - Create a board, restart the container, and confirm the board is still returned by `GET /api/boards`.
  - `DELETE` a board; confirm subsequent `GET` returns 404 and no orphaned cards remain in the database.
  - Grant read-only access to User B; `PUT`/`PATCH`/`DELETE` on that board as User B returns 403 Forbidden.
  - Revoke User B's access; confirm User B gets 403 on subsequent `GET` for that board.
  - Apply two concurrent `PATCH` (JSON Patch) operations to the same card from two clients; verify both are applied without silent data loss (last-write-wins is documented; a conflict error is preferred).
  - Open two WebSocket connections; update a card from one; verify the broadcast reaches the second connection within 1 second.
  - Disconnect a WebSocket client mid-session; verify the server releases the connection (no goroutine or file descriptor leak detectable via `podman stats`).
  - Run `go test -race ./...`; all board and WebSocket handler tests must pass without data races.

### Part 8: Admin Panel
- Create a frontend `/admin` route (protected by admin role).
- Build APIs to list/edit all users and view global board statistics.
- **Goal:** Admins can securely manage the system.
- **QA Testing Criteria:**
  - Log in as Admin; `GET /admin` returns 200 and the admin UI renders.
  - Log in as Regular User; `GET /admin` returns 403 and the admin UI is not rendered (no partial data leak).
  - Unauthenticated `GET /admin` returns 401 or redirects to login, not a 403 or 500.
  - Suspend a user via Admin API; suspended user's JWT is rejected with 401 on next request (or 403 if tokens are not immediately invalidated — document which).
  - Delete a user via Admin API; verify all their boards and permissions are cleaned up (no orphaned rows).
  - Admin API endpoints (`/api/admin/...`) return 403 for non-admin JWTs, not just 401 — the route must exist but be forbidden, not hidden.
  - Verify global board statistics endpoint returns consistent counts matching the database.

### Part 9: MCP Server Integration
- Implement MCP Server protocol on a dedicated endpoint or stdio wrapper in Go.
- Expose tools: `get_boards`, `read_board`, `update_card`, `add_card`.
- **Goal:** External AI agents can securely interact with the user's boards via MCP.
- **QA Testing Criteria:**
  - Connect a local MCP client to the server; verify the tool manifest lists all four tools with correct input schemas.
  - Execute `get_boards`; verify the response lists all boards the authenticated user owns or has access to, and no boards they do not.
  - Execute `read_board` with a valid board ID; verify the response matches the database state.
  - Execute `read_board` with a board ID the caller does not own; verify a permission error is returned, not an empty result.
  - Execute `update_card`; verify the database is updated and a WebSocket broadcast fires to connected UI clients.
  - Execute `add_card` with missing required fields; verify a structured error response (not a panic or 500).
  - Verify MCP calls require authentication (an API key or JWT); unauthenticated calls are rejected.

### Part 10: Built-in AI Chat
- Go backend acts as an OpenAI client to provide an in-app chat assistant.
- Use Structured Outputs to parse AI intentions into granular board updates.
- **Goal:** Chat endpoint can answer questions and apply valid board updates.
- **QA Testing Criteria:**
  - Send a natural language prompt via the chat API; verify the backend returns a valid structured JSON response with `intent` and `board_update` fields (or equivalent schema).
  - Verify the backend contacts the OpenAI API (check logs or a mock); confirm the request includes the correct model and structured output schema.
  - Verify a board update intent results in the correct database change and a WebSocket broadcast to connected clients.
  - Send a prompt that produces no board update (e.g. a question); verify the backend returns a chat reply with no mutation applied.
  - Simulate an OpenAI API error (e.g. timeout or 429); verify the endpoint returns a graceful error to the client (no panic, no raw upstream error body exposed).
  - Verify the chat endpoint requires authentication; unauthenticated requests return 401.

### Part 11: AI Sidebar UI & Real-time Updates
- Build Chat sidebar UI.
- Hook up WebSockets to auto-refresh the Kanban board when the AI (or MCP or collaborators) modifies it.
- **Goal:** Fluid user experience where chatting with AI visibly updates the board in real-time.
- **QA Testing Criteria:**
  - Open the chat sidebar; verify a loading spinner appears while the AI request is in flight and disappears on completion.
  - Submit a prompt that triggers a board update; verify the Kanban board re-renders with the change without a full page reload.
  - Open the same board in two browser sessions as different users; trigger an AI update in session A; verify the board in session B refreshes automatically within 1 second.
  - Simulate a WebSocket disconnection (disable network, re-enable); verify the UI reconnects and replays any missed updates without requiring a manual reload.
  - Verify no console errors appear during a normal chat + board update flow.
  - Verify the AI sidebar is not accessible (hidden or disabled) when the user is unauthenticated.

---

## Verification Plan

### Automated Tests
- Go backend: `go test -race ./...` covering API endpoints, JWT, RBAC, SQLite queries, and WebSocket handlers. Race detector is mandatory on every run.
- Frontend: `npm run build` (TypeScript type-check + Next.js build) must pass with zero errors.
- End-to-end: `npm run test:e2e` (Playwright) executed against a running Podman container (not a dev server).

### Security Baseline (run before each Part 5+ merge)
- Unauthenticated request to every protected endpoint returns 401 or redirect — never 200 or 500.
- Non-admin request to every admin endpoint returns 403 — not 401 or 404.
- No endpoint exposes stack traces, internal paths, or raw database errors in response bodies.
- `X-Forwarded-For` spoofing does not grant elevated access or bypass rate limits.

### Container Health Checks (run after every Containerfile change)
- `scripts/start.sh container` completes without error; `podman ps` shows container in `running` state.
- `curl http://localhost:${PORT}/api/health` returns `{"status":"ok","version":"..."}` with HTTP 200.
- `podman exec <container> id` shows a non-root user (uid ≠ 0).
- `podman stop <container>` exits with code 0 and no fatal log lines (graceful shutdown).
- Container restart (`podman restart`) preserves all database data via named volume.
- Background local server logs persist under `data/logs/` and survive process restarts until rotated manually.

### Manual Verification
- Build and run Podman container: `podman build` & `podman run`.
- Test login, multi-board CRUD, admin panel access, real-time sync, and AI chat.
- Connect an external MCP client to verify the MCP tools.
- Verify WebSocket reconnection after a simulated network drop.
