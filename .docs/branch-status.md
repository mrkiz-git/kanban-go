# Branch Status

Track unresolved issues per feature branch. The agent checks this file before starting any plan step.

**Legend:** `- [ ]` open · `- [x]` resolved

---

## main

No active development. Feature work happens on part branches.

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

_Status: not started_

_No open issues._

---

## part-04-frontend-base

_Status: not started_

_No open issues._

---

## part-05-auth

_Status: not started_

_No open issues._

---

## part-06-database

_Status: not started_

_No open issues._

---

## part-07-board-api

_Status: not started_

_No open issues._

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
