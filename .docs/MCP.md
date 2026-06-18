# MCP Server

[Model Context Protocol](https://modelcontextprotocol.io/) integration exposing Kanban board operations to external AI agents. Implemented in Go (Part 9).

## Transport

| Mode | Endpoint / Invocation | Use case |
|------|----------------------|----------|
| HTTP | `POST /mcp` (Streamable HTTP) | Remote agents, in-container sidecar |
| stdio | `kanba mcp` subcommand | Local CLI agents (Cursor, Claude Desktop) |

Both transports share the same tool implementations in `internal/mcp/`.

## Authentication

All MCP requests require a valid user JWT (see `AUTH.md`):

- **HTTP transport:** `Authorization: Bearer <jwt>` header
- **stdio transport:** `KANBA_TOKEN` env var set before launching the process

Tools operate within the authenticated user's permission scope. No admin-elevated tools in v1.

## Server Capabilities

```json
{
  "capabilities": {
    "tools": {}
  }
}
```

Resources and prompts are out of scope for v1.

## Tools

### `get_boards`

List all boards accessible to the authenticated user.

**Input schema:**

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

**Returns:** Array of `BoardSummary` (see `BOARD_SCHEMA.md`).

```json
{
  "boards": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Sprint 12",
      "permission": "owner",
      "updatedAt": "2026-06-18T10:00:00Z"
    }
  ]
}
```

---

### `read_board`

Fetch the full board document.

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "board_id": {
      "type": "string",
      "description": "UUID of the board to read"
    }
  },
  "required": ["board_id"],
  "additionalProperties": false
}
```

**Returns:** Full `Board` object (see `BOARD_SCHEMA.md`).

**Errors:**

| Condition | MCP error |
|-----------|-----------|
| Board not found | `not_found` |
| No read access | `forbidden` |

---

### `add_card`

Create a new card in a column.

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "board_id": {
      "type": "string",
      "description": "UUID of the target board"
    },
    "column_id": {
      "type": "string",
      "description": "UUID of the column to add the card to"
    },
    "title": {
      "type": "string",
      "description": "Card title (1-200 chars)"
    },
    "description": {
      "type": "string",
      "description": "Optional Markdown description"
    },
    "position": {
      "type": "integer",
      "description": "Zero-based position within the column. Defaults to end."
    }
  },
  "required": ["board_id", "column_id", "title"],
  "additionalProperties": false
}
```

**Returns:**

```json
{
  "card": {
    "id": "uuid",
    "title": "New task",
    "description": "",
    "position": 2,
    "updatedAt": "2026-06-18T10:00:00Z"
  }
}
```

Requires `write` or `owner` permission. Broadcasts `card.updated` WebSocket event.

---

### `update_card`

Apply granular updates to an existing card via JSON Patch semantics.

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "board_id": {
      "type": "string",
      "description": "UUID of the board containing the card"
    },
    "card_id": {
      "type": "string",
      "description": "UUID of the card to update"
    },
    "title": {
      "type": "string",
      "description": "New card title"
    },
    "description": {
      "type": "string",
      "description": "New Markdown description"
    },
    "column_id": {
      "type": "string",
      "description": "Move card to this column"
    },
    "position": {
      "type": "integer",
      "description": "New position within the target column"
    }
  },
  "required": ["board_id", "card_id"],
  "additionalProperties": false
}
```

At least one of `title`, `description`, `column_id`, or `position` must be provided.

**Returns:** Updated `Card` object.

Internally translated to RFC 6902 patch operations (see `BOARD_SCHEMA.md`) and applied via the same code path as `PATCH /api/boards/:id`.

Requires `write` or `owner` permission. Broadcasts `card.moved` or `card.updated` WebSocket event.

---

## Go Tool Registration

```go
package mcp

import "github.com/mark3labs/mcp-go/server"

func RegisterTools(s *server.MCPServer, boards BoardService) {
    s.AddTool(getBoardsTool(boards))
    s.AddTool(readBoardTool(boards))
    s.AddTool(addCardTool(boards))
    s.AddTool(updateCardTool(boards))
}
```

`BoardService` wraps `store.BoardStore` with permission checks from `AUTH.md`.

## Error Mapping

| Internal error | MCP tool result |
|----------------|-----------------|
| `domain.ErrNotFound` | `isError: true`, message "Board/card not found" |
| `domain.ErrForbidden` | `isError: true`, message "Insufficient permission" |
| `domain.ErrValidation` | `isError: true`, message with field details |
| Other | `isError: true`, message "Internal error" |

## Security Constraints

- Tools cannot access boards outside the authenticated user's owned + shared set.
- No tool for user management, admin operations, or attachment upload in v1.
- Input validated against JSON schemas before execution.
- All mutations logged with `user_id`, `tool_name`, `board_id`, timestamp.

## Client Configuration Example

**Claude Desktop (`claude_desktop_config.json`):**

```json
{
  "mcpServers": {
    "kanba": {
      "command": "/usr/local/bin/kanba",
      "args": ["mcp"],
      "env": {
        "KANBA_TOKEN": "<user-jwt>",
        "DATABASE_PATH": "/data/kanba.db"
      }
    }
  }
}
```

**HTTP client:**

```
POST https://kanba.example.com/mcp
Authorization: Bearer <user-jwt>
Content-Type: application/json
```

## Cross-References

- Board types: `BOARD_SCHEMA.md`
- REST equivalents: `API.md`
- Permission model: `AUTH.md`
- Persistence: `DATABASE.md`
