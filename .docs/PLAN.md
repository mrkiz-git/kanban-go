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

### Part 2: UI Planning
- Create `UI.md` to define design tokens, color palette, wireframes, and component hierarchy.
- Include UI plans for the Admin Panel, multi-board switching, and sharing dialogs.
- **Goal:** User approves `UI.md` specifications.

### Part 3: Scaffolding (Go + Podman)
- Initialize Go module (`go mod init`).
- Create Go backend structure with a router and `GET /api/health`.
- Update `Containerfile` to use a multi-stage build: Node (Next.js) -> Go -> Alpine.
- **Goal:** `scripts/start.sh` spins up the Go server via Podman.

### Part 4: Frontend (Static Base)
- Next.js static export (`output: 'export'`) served by the Go backend at `/`.
- Implement basic routing and empty state layouts.
- **Goal:** Browser hits `/` and sees the base app shell.

### Part 5: Auth, Security, & User Management
- Implement JWT generation and validation in Go (`golang-jwt/jwt`).
- Build user management API (Registration, Login, RBAC for Admins).
- **Goal:** Unauthenticated users are redirected to login. Role-based routing is established.

### Part 6: Database Modeling & Go Integration
- Integrate pure-Go SQLite (`modernc.org/sqlite`).
- Implement schema migrations (Users, Boards, Permissions/Shares).
- Seed initial admin user.
- **Goal:** Database survives container restarts and supports RBAC.

### Part 7: Core Board API & Sharing
- `GET`, `POST`, `PUT`, `PATCH`, `DELETE` for multiple boards.
- Implement collaborative sharing endpoints (granting access to users).
- Implement WebSocket broadcasting for real-time sync.
- **Goal:** Full multi-board CRUD and sharing persists to SQLite.

### Part 8: Admin Panel
- Create a frontend `/admin` route (protected by admin role).
- Build APIs to list/edit all users and view global board statistics.
- **Goal:** Admins can securely manage the system.

### Part 9: MCP Server Integration
- Implement MCP Server protocol on a dedicated endpoint or stdio wrapper in Go.
- Expose tools: `get_boards`, `read_board`, `update_card`, `add_card`.
- **Goal:** External AI agents can securely interact with the user's boards via MCP.

### Part 10: Built-in AI Chat
- Go backend acts as an OpenAI client to provide an in-app chat assistant.
- Use Structured Outputs to parse AI intentions into granular board updates.
- **Goal:** Chat endpoint can answer questions and apply valid board updates.

### Part 11: AI Sidebar UI & Real-time Updates
- Build Chat sidebar UI.
- Hook up WebSockets to auto-refresh the Kanban board when the AI (or MCP or collaborators) modifies it.
- **Goal:** Fluid user experience where chatting with AI visibly updates the board in real-time.

---

## Verification Plan

### Automated Tests
- Go backend: `go test ./...` covering API endpoints, JWT, RBAC, and SQLite queries.
- Frontend: `npm run test:unit` (Vitest) and `npm run test:e2e` (Playwright) against the Podman container.

### Manual Verification
- Build and run Podman container: `podman build` & `podman run`.
- Test login, multi-board CRUD, admin panel access, real-time sync, and AI chat.
- Connect an external MCP client to verify the MCP tools.
