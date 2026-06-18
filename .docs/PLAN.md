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

### Part 3: Base Scaffolding (Full-Stack)
- Initialize Go module (`go mod init`) and routing (`GET /api/health`).
- Initialize Next.js static export (`output: 'export'`) served by the Go backend at `/`.
- Implement basic Next.js routing and empty AppShell layouts.
- Update `Containerfile` to use a multi-stage build: Node (Next.js) -> Go -> Alpine.
- Add structured logging with three levels (`error`, `info`, `debug`) via `LOG_LEVEL`.
- Add server run modes: `scripts/start.sh verbose` (foreground logs), `scripts/start.sh background` (daemon with persistent log file), and `scripts/start.sh container` (Podman).
- Add `scripts/stop.sh` to stop a background local server or Podman container.
- **Goal:** `scripts/start.sh` spins up the server, browser hits `/` and sees the base app shell, with configurable logging for local and container runs.
- **QA Testing Criteria:**
  - Run `go test -race ./...` successfully; race detector must be enabled.
  - `GET /api/health` returns HTTP 200 with `Content-Type: application/json` and body `{"status":"ok","version":"..."}`.
  - Verify Next.js build (`npm run build`) completes successfully with no TypeScript errors.
  - `GET /` returns HTTP 200 with `Content-Type: text/html` from the Go static file handler.
  - Load `/` in browser and confirm UI shell renders; verify no console errors on initial load.
  - Verify `scripts/start.sh container` builds and runs the container without errors; `podman ps` shows the container running.
  - Verify `scripts/start.sh verbose` runs in the foreground and prints request logs at `debug` level.
  - Verify `scripts/start.sh background` writes logs to a persistent file under `data/logs/` and `scripts/stop.sh` stops the process cleanly.
  - Verify `LOG_LEVEL=error` suppresses info/debug request logs; `LOG_LEVEL=debug` shows them.
  - Verify `http.Server` configures `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.
  - Verify graceful shutdown: send SIGTERM (`podman stop` or `./scripts/stop.sh`); the process exits cleanly with no fatal error.
  - Verify the container process runs as a non-root user: `podman exec <container> id` must not show `uid=0`.
  - Verify the port mapping is correct: `curl http://localhost:${PORT}/api/health` returns 200.
  - Confirm `middleware.RealIP` is removed or documented as requiring a trusted proxy before Parts 4+ add IP-based controls.

### Part 4: Database, Auth & User Management (Full-Stack)
- Integrate pure-Go SQLite (`modernc.org/sqlite`) with schema migrations for Users.
- Implement backend JWT generation and auth APIs (Registration, Login, RBAC).
- Build frontend `/login` and `/register` pages.
- Implement client-side auth state and protected route redirects.
- **Goal:** Users can register, login, and access protected routes. Database persists and supports RBAC.
- **QA Testing Criteria:**
  - Verify user registration creates a valid user and hashes the password in SQLite.
  - Verify login issues a valid JWT and client saves it properly.
  - Verify accessing protected routes without a valid JWT returns 401 Unauthorized.
  - Verify the frontend correctly redirects unauthenticated users away from protected pages.
  - Verify SQLite database persists across application restarts.

### Part 5: Core Board & Kanban UI (Full-Stack)
- Implement database schema migrations for Boards, Columns, and Cards.
- Build backend APIs for multiple boards CRUD (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`).
- Build frontend `BoardListPage` (`/boards`) and `BoardPage` (`/boards/:id`).
- Implement drag-and-drop Kanban UI (`@hello-pangea/dnd`) with optimistic updates.
- Implement inline editing for boards/columns and `CardModal` for editing cards and Markdown.
- **Goal:** Users can fully manage their own boards, columns, and cards via a polished UI.
- **QA Testing Criteria:**
  - Verify complete CRUD operations for boards, columns, and cards via API and UI.
  - Verify drag-and-drop correctly updates card positions and column associations.
  - Verify optimistic updates immediately reflect UI changes and gracefully revert on API failure.
  - Verify CardModal correctly parses and renders Markdown inputs, including attachments.
  - Verify users cannot view, edit, or delete boards they do not own or have access to.

### Part 6: Real-time Sync & Sharing (Full-Stack)
- Implement schema migrations for Permissions/Shares.
- Implement backend collaborative sharing endpoints.
- Build frontend Sharing Dialog to grant access to other users.
- Implement backend WebSocket broadcasting for real-time sync.
- Hook up frontend WebSocket client to auto-refresh the Kanban board.
- **Goal:** Users can share boards, and changes reflect instantly across active sessions.
- **QA Testing Criteria:**
  - Verify board owners can successfully grant read-only and read-write access to other users.
  - Verify shared users have appropriate access constraints applied on both frontend and backend.
  - Verify WebSocket broadcasts accurately push card/column updates to all connected clients viewing the board.
  - Verify the frontend auto-reconnects to the WebSocket server upon connection drop.
  - Verify users cannot share boards they do not own.

### Part 7: Admin Panel (Full-Stack)
- Build backend APIs to list/edit all users and view global board statistics.
- Create frontend `/admin` routes (protected by admin role).
- Build the `AdminUsersPage` table, user editing modals, and `AdminStatsPage`.
- **Goal:** Admins can securely manage the system users and view stats.
- **QA Testing Criteria:**
  - Verify only users with an 'Admin' role can access the frontend `/admin` dashboard and backend admin endpoints.
  - Verify standard users receive a 403 Forbidden when attempting to access admin routes.
  - Verify admins can successfully suspend, delete, or edit roles for other users.
  - Verify the `AdminStatsPage` correctly computes and displays global board and user statistics.

### Part 8: AI Chat & MCP Integration (Full-Stack)
- Implement MCP Server protocol in Go to expose board management tools.
- Go backend acts as an OpenAI client to provide an in-app chat assistant.
- Build frontend AI Chat sidebar UI.
- Hook up WebSockets to auto-refresh the Kanban board when the AI or MCP modifies it.
- **Goal:** Chat endpoint answers questions, external AI agents can connect, and chatting visibly updates the board in real-time.
- **QA Testing Criteria:**
  - Verify the in-app chat interface responds to user queries accurately based on board context.
  - Verify any board modifications made by the AI immediately trigger WebSocket refreshes on the UI.
  - Verify an external MCP client can successfully discover and execute board management tools via the Go server.
  - Verify AI and MCP tool executions strictly adhere to the authenticated user's RBAC permissions.

---

## Verification Plan

### Automated Tests
- Go backend: `go test -race ./...` covering API endpoints, JWT, RBAC, SQLite queries, and WebSocket handlers. Race detector is mandatory on every run.
- Frontend: `npm run build` (TypeScript type-check + Next.js build) must pass with zero errors.
- End-to-end: `npm run test:e2e` (Playwright) executed against a running Podman container (not a dev server).

### Security Baseline (run before each Part 4+ merge)
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
