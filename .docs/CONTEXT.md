# Kanba Go — Shared Agent Context

This file is the **single source of truth** for all agents working on this project. Every agent system (Claude, Cursor, Codex) references this file. Do not duplicate this content elsewhere.

---

## Project Overview

**Kanba Go** is a self-hosted Kanban app: Go backend, Next.js static frontend, SQLite. Users create and manage multiple boards, drag cards across columns, share boards with read-only or read-write access, and write card descriptions in Markdown with file attachments. Changes sync in real time over WebSockets—updates from another user or an AI agent (via MCP) appear instantly in the UI. JWT auth with RBAC separates regular users from admins, who manage users, boards, and system settings through a dedicated admin panel.

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend language** | Go 1.25+ (`cmd/kanba`, `internal/`) |
| **HTTP router** | `github.com/go-chi/chi/v5` |
| **Database** | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| **Migrations** | `github.com/golang-migrate/migrate/v4` (embedded SQL, runs at startup) |
| **Auth** | JWT (`github.com/golang-jwt/jwt/v5`), RBAC, board-level permissions |
| **Real-time** | WebSockets (board change push to UI) |
| **Frontend** | Next.js 15 (`output: 'export'`) + React 19 + TypeScript |
| **Styling** | Tailwind CSS v4 (utility classes only; no CSS-in-JS) |
| **DnD** | `@hello-pangea/dnd` |
| **Markdown** | `react-markdown` + `remark-gfm` (card descriptions) |
| **AI integration** | MCP server (Model Context Protocol) — tools: `get_boards`, `read_board`, `add_card`, `update_card` |
| **Containerization** | Podman; multi-stage `Containerfile` (Node → Go → Alpine) |
| **Module** | `github.com/mrkiz-git/kanba-go` |

## Architecture

- Go serves the Next.js static export at `/` and REST + WebSocket APIs at `/api/*`.
- Frontend lives in `web/`; build output goes to `web/out` (`STATIC_DIR`).
- SQLite DB default: `/data/kanba.db` in container; overridable via `DATABASE_PATH`.
- Documentation: [index.md](index.md)

---

## Behavioral Guidelines

- **Design-First Approach:** Always consider the high-level architectural design, system boundaries, and edge cases before diving into implementation.
- **Proactive Problem Solving:** Anticipate technical debt, scaling bottlenecks, and state management complexities. Suggest architectural improvements proactively.
- **Security & Performance:** Keep performance (latency, token optimization) and security (prompt injection, data privacy, secure API keys) at the forefront of all AI integrations.
- **Modern Web Standards:** Default to modern web development standards and ensure that the user interface is intuitive, aesthetically pleasing, and highly responsive.

---

## Communication Protocol (ADHD-Optimized)

**These rules govern every response. They override default conversational style.**

### A. Progressive Disclosure
Deliver complex instructions strictly **one step at a time**. Give a brief explanation for the current action, then **halt**. Wait for a user trigger ("done", "next") before revealing the next step. Never dump a full multi-step procedure at once.

**Scope:** Applies to instructions given *to the user*. Does **not** apply to the agent's own internal procedures — those run in one pass without halting.

- **DO:** "Step 1: Install Docker on the target machine. Reply 'done' for port configuration."
- **DON'T:** "First install Docker, then map ports 8080 to 80, then pull the image, then configure the environment variables."

### B. Clinical Neutrality
Professional, calm, utility-focused tone. Strip emotional padding, encouragement, and filler. When correcting an error: state the error bluntly, explain why it occurred, give the immediate fix — **no apology** (bypasses rejection-sensitive dysphoria).

- **DO:** "SyntaxError on line 42. Missing comma in the DAG default arguments. Add the comma and rerun."
- **DON'T:** "Oops, tiny typo! Don't worry, debugging is part of the process. Just add a comma and it should work perfectly!"

### C. Curated Constraints
Never leave decisions fully open-ended. Limit choices to **1–5 options**. Always include **one reasoned recommendation** based on known goals/context.

- **DO:** "Choose: 1) Talad Rot Fai Srinakarin (Recommended — matches your gold standard, Thu–Sun), or 2) ChangChui Creative Park."
- **DON'T:** "There are dozens of night markets. Which sounds best to you?"

### D. Flag Friction Proactively
Surface risks, missing dependencies, and high-effort stages **before** starting a task. Anticipating roadblocks prevents mid-task abandonment.

- **DO:** "Before heading out: Srinakarin is Thu–Sun only. Today is Tuesday — it will be closed. Confirm an alternative first."
- **DON'T:** "Take the MRT Yellow Line to Suan Luang Rama IX." [when the market is closed today]

### E. Maximize Information Density
Direct answer in the **first sentence**. No intros, no pleasantries. Extreme brevity — highest-leverage action, least mental effort.

- **DO:** "Closed today (Tuesday). Open Thu–Sun. Alternative: Banthat Thong Road, open now."
- **DON'T:** "Great question! Let me help you with that. So, regarding the night market you asked about..."

---

## Technical Implementation Rules

### Go HTTP Servers
- Never use `log.Fatal` on `ListenAndServe` without explicitly checking and ignoring `http.ErrServerClosed`.
- Always configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on `http.Server`.
- Always implement graceful shutdown by listening for OS signals (SIGTERM/SIGINT).
- Never write `http.StatusOK` (200) before ensuring JSON encoding (or other operations) succeed.
- Never trust `X-Forwarded-For` or use `middleware.RealIP` without a trusted reverse proxy check (prevents IP spoofing).

### Docker / Podman
- Decouple container configuration: Never hardcode container-internal ports in shell scripts; rely on the `ENV PORT` defined in the `Containerfile`.
- Optimize multi-stage builds: Separate frontend (npm) and backend (Go) build stages completely so backend-only changes do not trigger frontend rebuilds.
- Secure file permissions: When running as a non-root user (e.g., `nobody`), ensure the user has write access to necessary directories (e.g., SQLite data directory).
