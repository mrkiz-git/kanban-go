# Authentication & Authorization

JWT-based authentication with Role-Based Access Control (RBAC) at the system level and per-board permissions for collaborative sharing.

## Stack

| Component | Library |
|-----------|---------|
| JWT | `github.com/golang-jwt/jwt/v5` |
| Password hashing | `golang.org/x/crypto/bcrypt` (cost 12) |
| Token signing | HMAC-SHA256 (`HS256`) |
| Secret | `JWT_SECRET` env var (min 32 bytes) |

## User Roles (System-Level RBAC)

| Role | Value | Capabilities |
|------|-------|-------------|
| Admin | `admin` | Full system access; `/api/admin/*`; can view/manage all boards and users |
| User | `user` | Own boards; access shared boards; no admin endpoints |

Stored in `users.role` (see `DATABASE.md`).

### User Status

| Status | Behavior |
|--------|----------|
| `active` | Normal access |
| `suspended` | JWT validation fails; login returns 403 |

Only admins can change `role` or `status` (via admin API).

## Board-Level Permissions

Separate from system role. Resolved per request (see `DATABASE.md` access resolution).

| Permission | Create board | Read board | Write board | Delete board | Manage shares |
|------------|-------------|------------|-------------|--------------|---------------|
| `owner` | — | yes | yes | yes | yes |
| `write` | — | yes | yes | no | no |
| `read` | — | yes | no | no | no |
| `admin` (system) | yes | yes | yes | yes | yes |

Admins acting on another user's board use admin override (logged for audit in Part 8).

## JWT Structure

```go
type Claims struct {
    jwt.RegisteredClaims
    UserID string `json:"uid"`
    Email  string `json:"email"`
    Role   string `json:"role"` // "admin" | "user"
}
```

| Claim | Value |
|-------|-------|
| `sub` | User UUID |
| `uid` | User UUID (duplicate for convenience) |
| `email` | User email |
| `role` | `admin` or `user` |
| `iat` | Issued at |
| `exp` | Issued at + 24 hours |

No board permissions in the JWT — resolved from database on each request to reflect live share changes.

## Auth Endpoints

### `POST /api/auth/register`

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepass123",
  "name": "Jane Doe"
}
```

**Validation:**

| Field | Rule |
|-------|------|
| `email` | Valid email format, unique |
| `password` | 8–128 chars |
| `name` | 1–100 chars |

**Response 201:**

```json
{
  "token": "eyJ...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Jane Doe",
    "role": "user"
  }
}
```

New users always receive `role: user`. Admin promotion only via admin API.

### `POST /api/auth/login`

**Request:**

```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

**Response 200:** Same shape as register.

**Response 401:** Invalid credentials (generic message: "Invalid email or password").

**Response 403:** Account suspended.

### `GET /api/auth/me`

**Response 200:**

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "Jane Doe",
  "role": "user"
}
```

### `POST /api/auth/refresh`

Re-issues JWT if current token is valid and not expired. Same response shape as login (without password check).

## Middleware

```go
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler
func RequireRole(roles ...string) func(http.Handler) http.Handler
func RequireBoardPerm(perm domain.BoardPermission) func(http.Handler) http.Handler
```

`AuthMiddleware` checks token sources in this order:
1. `Authorization: Bearer <jwt>` header (non-browser clients: MCP, curl, programmatic access)
2. `kanba_token` httpOnly cookie (browser clients — sent automatically by the browser, no JavaScript access required)

Execution order for board routes:

1. `AuthMiddleware` — resolve token from header or cookie, parse JWT, set user in context; 401 on failure
2. `RequireBoardPerm` — load board, resolve permission; 403 on failure
3. Handler

Admin routes use `RequireRole("admin")` instead of board permission.

## Frontend Auth Flow

1. Unauthenticated visitors hitting `/` or `/boards/*` redirect to `/login`.
2. Login sets an `httpOnly` cookie `kanba_token` (SameSite=Lax, Secure in production) via `Set-Cookie` response header.
3. The browser automatically includes `kanba_token` on same-origin requests — no JavaScript token handling needed.
4. Admin users see `/admin` nav link; `/admin/*` routes check `role === 'admin'` client-side and server-side.
5. On 401 response, client redirects to `/login` (cookie cleared server-side on logout via `Set-Cookie: kanba_token=; Max-Age=0`).

## MCP & WebSocket Auth

| Channel | Auth method |
|---------|-------------|
| REST API | `Authorization: Bearer <jwt>` header |
| WebSocket | `?token=<jwt>` query parameter |
| MCP (stdio) | Inherits OS user session; MCP HTTP transport uses Bearer header |
| MCP (HTTP) | `Authorization: Bearer <jwt>` on `/mcp` endpoint |

MCP tools operate within the authenticated user's permission scope — no elevated access beyond their boards and shares.

## Security Notes

- Passwords never logged or returned in API responses.
- Rate-limit login: 10 attempts per IP per minute (implemented at scaffolding).
- JWT secret rotation requires restart; no refresh-token pair in v1.
- CORS: same-origin only (Go serves both API and static frontend).

## Cross-References

- API endpoints: `API.md`
- User persistence: `DATABASE.md`
- MCP tool auth: `MCP.md`
