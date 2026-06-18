# Part 4 — Code Review & QA Issues

Branch: `part-04-frontend-base`
Reviewed: 2026-06-18

Legend: **BUG** = correctness defect · **SECURITY** = security concern · **QA-MISS** = explicit PLAN.md QA criterion unmet · **SIMPLIFICATION** = maintenance/readability debt

---

## 1. [BUG] Login and Register submit buttons use `type="button"` — form submission broken

**Files:** `web/app/(auth)/login/page.tsx:27`, `web/app/(auth)/register/page.tsx:35`
**Severity:** High — will silently break auth wiring in Part 5.

Both forms contain:
```tsx
<Button type="button" className="w-full">Sign In</Button>
<Button type="button" className="w-full">Register</Button>
```

`Button.tsx` spreads `...props` onto a `<button>`, so `type="button"` is passed through. Buttons inside a `<form>` default to `type="submit"` — explicitly setting `type="button"` disables both Enter-key submission and click-to-submit. When handlers are wired in Part 5, neither button will trigger the form's submit event.

**Fix:** Change both to `type="submit"`.

---

## 2. [QA-MISS / BUG] Request logger emits at `Info` level — PLAN.md requires `Debug`

**File:** `internal/server/server.go:46`
**Severity:** High — explicit QA criterion failure.

PLAN.md states: *"Verify `scripts/start.sh verbose` runs in the foreground and prints request logs at `debug` level."*

The middleware hardcodes `logger.Info(...)`:
```go
logger.Info(
    "request",
    "method", r.Method,
    ...
)
```

With `LOG_LEVEL=info` (the default), every HTTP request — including static assets — produces a log line. The intent per QA is that request logs appear only at `debug` level (i.e., suppressed in normal operation). The current code never emits a `DEBUG`-tagged request line regardless of `LOG_LEVEL`.

**Fix:** Change `logger.Info("request", ...)` to `logger.Debug("request", ...)`.

---

## 3. [BUG] `os.Stat` errors silently fall back to `index.html` — hides real server errors

**File:** `internal/handler/static.go:28-29`
**Severity:** Medium — permission errors and I/O errors are indistinguishable from "not found."

```go
if _, err := os.Stat(filePath); err != nil {
    filePath = indexPath  // swallows ALL errors — not just os.IsNotExist
}
```

A permission-denied error (`EACCES`) or I/O error causes silent fallback to `index.html` instead of a 500 response. If `index.html` itself has a permission problem, the user sees the SPA shell (or a silent 403 from `http.ServeFile`) with no diagnostic. The error is never logged.

**Fix:** Check `os.IsNotExist(err)` before falling back; return 500 (with log) for other error types.

---

## 4. [BUG] Stale PID file after binary crash — restart fails or kills unrelated process

**File:** `scripts/start.sh` (background mode, lines ~64–65)
**Severity:** Medium — prevents clean restart after crash; risk of killing wrong process.

```bash
nohup "$BINARY" </dev/null >/dev/null 2>&1 &
echo $! >"$PID_FILE"
```

If the binary crashes immediately (e.g., port already bound, missing static dir), `$PID_FILE` holds the short-lived PID. The OS can recycle that PID for an unrelated process. On the next `start.sh background` invocation:
```bash
kill -0 "$(cat "$PID_FILE")" 2>/dev/null  # succeeds for unrelated process
echo "Kanba is already running."
exit 1
```

The server cannot be restarted without manually deleting `$PID_FILE`. Worse, if the user runs `stop.sh` first, it sends `SIGTERM` to the unrelated process (the one that recycled the PID).

**Fix:** After `nohup ... &`, sleep briefly and check `kill -0 $PID` then verify the process is actually the binary (e.g., check `/proc/$PID/exe` on Linux, or use a lock file instead of a PID file).

---

## 5. [QA-MISS] `middleware.RealIP` removal is undocumented

**File:** `internal/server/server.go` (middleware chain)
**Severity:** Medium — explicit PLAN.md QA criterion unmet.

PLAN.md states: *"Confirm `middleware.RealIP` is removed or documented as requiring a trusted proxy before Parts 4+ add IP-based controls."*

`middleware.RealIP` is correctly absent from the chi middleware stack. However, there is no code comment, ADR, or inline note documenting *why* it was omitted. A future contributor adding rate limiting or IP-based controls may add `middleware.RealIP` without understanding it trusts all proxies unconditionally by default.

**Fix:** Add a one-line comment above the middleware block:
```go
// RealIP is intentionally omitted — chi's default trusts all proxies; add only behind a trusted reverse proxy.
```

---

## 6. [SIMPLIFICATION] `NewFromConfig` duplicates the `MkdirAll` + `OpenFile` block

**File:** `internal/logging/logger.go:75-96`
**Severity:** Low — maintenance risk.

Both switch cases (`logFile != "" && !logStdout` and `logFile != ""`) contain identical file-open logic:
```go
os.MkdirAll(dirOf(logFile), 0o755)
file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
```

The only difference is whether `out` is `file` or `io.MultiWriter(os.Stdout, file)`. A future change to flags (e.g., adding `os.O_SYNC`, changing permissions) must be applied in two places and will silently diverge.

**Fix:** Extract file-open into one block; set `out` based on `logStdout`:
```go
// open file once
file, err := openLogFile(logFile)
if logStdout {
    out = io.MultiWriter(os.Stdout, file)
} else {
    out = file
}
```

---

## 7. [BUG] `dirOf` returns empty string for root-level absolute paths — diverges from `filepath.Dir`

**File:** `internal/logging/logger.go:145-150`
**Severity:** Low — latent; harmless in typical deployment.

```go
func dirOf(path string) string {
    if i := strings.LastIndex(path, "/"); i >= 0 {
        return path[:i]
    }
    return "."
}
```

For a path like `/kanba.log`, `strings.LastIndex` returns `0`, so `path[:0]` = `""`. Then `os.MkdirAll("", 0o755)` is a no-op (returns `nil` on Linux/macOS). `filepath.Dir("/kanba.log")` would correctly return `"/"`. In practice, the default log path (`data/logs/kanba.log`) has multiple components and works correctly. The edge case only affects unusual deployments.

**Fix:** Replace `dirOf` with `filepath.Dir`:
```go
// Before:
os.MkdirAll(dirOf(logFile), 0o755)
// After:
os.MkdirAll(filepath.Dir(logFile), 0o755)
```
Delete the `dirOf` function entirely.

---

## 8. [DEAD CODE] `http.NotFound` branch in `Static` is unreachable for all legitimate URLs

**File:** `internal/handler/static.go:22-26`
**Severity:** Low — misleading but harmless.

```go
filePath, ok := resolveStaticFile(root, r.URL.Path)
if !ok {
    http.NotFound(w, r)  // ← never reached for normal URLs
    return
}
```

`resolveStaticFile` always returns `true` on the final line:
```go
return filepath.Join(root, "index.html"), true  // SPA fallback
```

The only `false` return is inside the path-traversal guard which is itself unreachable via URL input (candidates are always constructed under `root` via `filepath.Join`). The `http.NotFound` branch is dead code — any "not found" case falls through to `http.ServeFile` on the `index.html` SPA fallback.

**Fix:** Either remove the `ok` return from `resolveStaticFile` and simplify the caller, or make `resolveStaticFile` return `false` when no candidate exists (and handle it with an actual 404 for non-SPA paths like `_next/` assets).

---

## Summary Table

| # | Severity | File | Issue |
|---|----------|------|-------|
| 1 | High | `login/page.tsx:27`, `register/page.tsx:35` | Submit buttons use `type="button"` — form submission broken |
| 2 | High | `server/server.go:46` | Request logs at `Info`; PLAN.md requires `Debug` level |
| 3 | Medium | `handler/static.go:28-29` | All `os.Stat` errors silently fall back to `index.html` |
| 4 | Medium | `scripts/start.sh` (background) | Stale PID after crash blocks restart / may kill wrong process |
| 5 | Medium | `server/server.go` (middleware) | `middleware.RealIP` removal undocumented (explicit PLAN.md QA criterion) |
| 6 | Low | `logging/logger.go:75-96` | Duplicated `MkdirAll`+`OpenFile` block in `NewFromConfig` |
| 7 | Low | `logging/logger.go:145` | `dirOf` returns `""` for root-level paths; should use `filepath.Dir` |
| 8 | Low | `handler/static.go:22-26` | `http.NotFound` branch is dead code — `resolveStaticFile` always returns `true` |
