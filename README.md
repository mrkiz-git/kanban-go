# Kanba Go

Kanban web app with a Go backend, Next.js static frontend, and MCP integration for AI agents.

**Status:** Part 3 (scaffolding) — Go server with health check, Podman container build.

## Prerequisites

- [Go](https://go.dev/) 1.23+
- [Node.js](https://nodejs.org/) 22+ (frontend build)
- [Podman](https://podman.io/)

## Quick start

Build and run the container:

```bash
./scripts/start.sh
curl http://localhost:8080/api/health
```

Expected response:

```json
{ "status": "ok", "version": "0.1.0" }
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST_PORT` | `8080` | Host port mapped to the container (`PORT` is an alias) |
| `CONTAINER_PORT` | `8080` | Port the Go server listens on inside the container |
| `KANBA_IMAGE` | `kanba-go:local` | Podman image tag |
| `KANBA_CONTAINER` | `kanba-go` | Container name |
| `KANBA_DATA_VOLUME` | `<container>-data` | Named Podman volume for `/app/data` |

Copy `.env.example` to `.env` for local reference. The container reads `PORT` and `HOST` at runtime.

## Local development

**Backend only:**

```bash
go run ./cmd/kanba
```

**Frontend stub** (static export; served by Go in Part 4):

```bash
cd web && npm install && npm run dev
```

**Tests:**

```bash
go test ./...
```

**Manual container build:**

```bash
podman build -f Containerfile -t kanba-go:local .
podman run --rm -p 8080:8080 kanba-go:local
```

## Project layout

```
cmd/kanba/          Go entrypoint
internal/           Server, handlers, config
web/                Next.js static export (stub)
scripts/start.sh    Build and run via Podman
Containerfile       Multi-stage: Node → Go → Alpine
.docs/              API, auth, schema, and development plan
```

## Documentation

Contract and design docs live in [`.docs/`](.docs/):

| Doc | Contents |
|-----|----------|
| [`PLAN.md`](.docs/PLAN.md) | Step-by-step development plan |
| [`API.md`](.docs/API.md) | REST and WebSocket endpoints |
| [`AUTH.md`](.docs/AUTH.md) | JWT, RBAC, permissions |
| [`BOARD_SCHEMA.md`](.docs/BOARD_SCHEMA.md) | Board domain model |
| [`DATABASE.md`](.docs/DATABASE.md) | SQLite schema and migrations |
| [`MCP.md`](.docs/MCP.md) | MCP tool schemas |

## License

Not specified.
