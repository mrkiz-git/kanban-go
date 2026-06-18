# Kanba Go

Kanban web app with a Go backend, Next.js static frontend, and MCP integration for AI agents.

**Status:** Part 4 (frontend base) — Go serves Next.js static export with app shell layouts.

## Prerequisites

- [Go](https://go.dev/) 1.23+
- [Node.js](https://nodejs.org/) 22+ (frontend build)
- [Podman](https://podman.io/)

## Quick start

### Container (default)

```bash
./scripts/start.sh container
curl http://localhost:8080/api/health
curl -I http://localhost:8080/boards/
./scripts/stop.sh
```

### Local development

**Verbose mode** — foreground server, debug logs on stdout:

```bash
./scripts/start.sh verbose
```

**Background mode** — daemon with persistent log file:

```bash
./scripts/start.sh background
curl http://localhost:8080/api/health
tail -f data/logs/kanba.log
./scripts/stop.sh
```

Expected health response:

```json
{ "status": "ok", "version": "0.1.0" }
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STATIC_DIR` | `web/out` | Path to Next.js static export |
| `HOST_PORT` | `8080` | Host port mapped to the container (`PORT` is an alias) |
| `CONTAINER_PORT` | `8080` | Port the Go server listens on inside the container |
| `LOG_LEVEL` | `info` | Log level: `error`, `info`, or `debug` |
| `LOG_FILE` | `data/logs/kanba.log` | Log file path for background/container modes |
| `LOG_STDOUT` | `1` | Set to `0` to write logs to file only |
| `KANBA_LOG_DIR` | `data/logs` | Directory for local background logs |
| `KANBA_PID_FILE` | `data/kanba.pid` | PID file for local background mode |
| `KANBA_IMAGE` | `kanba-go:local` | Podman image tag |
| `KANBA_CONTAINER` | `kanba-go` | Container name |
| `KANBA_DATA_VOLUME` | `<container>-data` | Named Podman volume for `/app/data` |

Copy `.env.example` to `.env` for local reference. The server reads `PORT`, `HOST`, `LOG_LEVEL`, and `LOG_FILE` at runtime.

## Logging

Three levels, least to most verbose:

| Level | Shows |
|-------|-------|
| `error` | Errors only |
| `info` | Errors + startup/shutdown + HTTP requests |
| `debug` | Everything (verbose mode default) |

## Local development

**Backend only:**

```bash
go run ./cmd/kanba
```

**Frontend** (static export served by Go):

```bash
cd web && npm install && npm run build
go run ./cmd/kanba
# open http://localhost:8080/boards/
```

For hot reload during UI work:

```bash
cd web && npm run dev
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
internal/           Server, handlers, config, logging
web/                Next.js static export (AppShell, routes)
scripts/start.sh    Start verbose, background, or container mode
scripts/stop.sh     Stop background server or container
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
