# Story 1.2: Daemon Lifecycle & Health Endpoint

Status: review

## Story

As a user,
I want to start, stop, and check the status of the pakai daemon,
so that I can manage the background service that powers my status bar integrations.

## Acceptance Criteria

1. **Given** no daemon is running, **When** I run `pakai daemon start`, **Then** the daemon starts, binds exclusively to `127.0.0.1:7731`, writes its PID to `$XDG_RUNTIME_DIR/pakai/daemon.pid`, and the foreground process exits with code 0

2. **Given** the daemon is running, **When** I run `pakai daemon status`, **Then** it prints the daemon's PID, uptime, and listening port and exits with code 0

3. **Given** the daemon is running, **When** I run `pakai daemon stop`, **Then** the daemon receives SIGTERM, drains in-flight requests within 5 seconds, removes the PID file (via `defer`, not signal handler), and exits cleanly

4. **Given** the daemon receives SIGTERM directly (e.g. `kill <pid>`), **When** shutdown completes, **Then** the PID file is removed and the process exits — no stale PID file remains

5. **Given** the daemon is running, **When** `GET /health` is called, **Then** it returns HTTP 200 with JSON containing `status`, `uptime_seconds`, `connections`, `port`, and `providers` within 50ms

6. **Given** port 7731 is already in use by another process, **When** I run `pakai daemon start`, **Then** it prints an explicit message ("port 7731 is already in use by another process") and exits with a non-zero code

7. **Given** the daemon is running, **When** any external interface (not 127.0.0.1) attempts a connection, **Then** the connection is refused — the server never binds to 0.0.0.0 or any non-loopback interface

8. **Given** the daemon crashes unexpectedly and systemd restarts it, **When** the new daemon starts, **Then** it starts cleanly — it does not fail due to a stale PID file from the previous run

## Tasks / Subtasks

- [x] Create `internal/daemon/` package with HTTP server (AC: 1, 7)
  - [x] Bind exclusively to `127.0.0.1:7731` — never `0.0.0.0`
  - [x] Implement `/health` handler returning `{"status","uptime_seconds","connections","port","providers"}` (AC: 5)
- [x] Implement PID file management (AC: 1, 4, 8)
  - [x] Write PID to `$XDG_RUNTIME_DIR/pakai/daemon.pid` on start
  - [x] Remove PID file in `defer` (not signal handler) on shutdown
  - [x] On start: if stale PID file exists, check if process is alive; if not, remove and continue (handles crash recovery)
- [x] Implement graceful shutdown with `signal.NotifyContext` (AC: 3, 4)
  - [x] Use `signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)`
  - [x] 5-second drain timeout for in-flight requests
  - [x] PID file removal via `defer` before server shutdown completes
- [x] Add `pakai daemon start/stop/status` subcommands in `cmd/pakai/` (AC: 1, 2, 3)
  - [x] `daemon start` — fork daemon and exit foreground
  - [x] `daemon stop` — send SIGTERM via PID file, wait for exit
  - [x] `daemon status` — read PID file, report uptime and port
- [x] Port conflict detection (AC: 6)
  - [x] Attempt `net.Listen("tcp", "127.0.0.1:7731")` — on EADDRINUSE, print explicit message and exit non-zero

## Dev Notes

- **Prerequisite:** Story 1.1 (`internal/model/`) must be complete. This story creates `internal/daemon/` which imports `model/`.
- **Port binding:** Use `net.Listen("tcp", "127.0.0.1:7731")` — never `":7731"` or `"0.0.0.0:7731"`. Binding to loopback is a security requirement (NFR11).
- **Signal handling pattern:**
  ```go
  ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
  defer stop()
  // ... start server ...
  <-ctx.Done()
  shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  server.Shutdown(shutdownCtx)
  ```
  PID file removal goes in `defer` before `signal.NotifyContext`, so it fires on any exit path.
- **PID file path:** Use `os.Getenv("XDG_RUNTIME_DIR")` with fallback to `/run/user/<uid>/pakai/`. Create directory if it doesn't exist (mode 0700).
- **Stale PID recovery (AC 8):** On start, if `daemon.pid` exists, read the PID and check `os.FindProcess(pid)` + `p.Signal(syscall.Signal(0))`. If process is dead, remove the stale PID file and proceed normally. Do not fail.
- **`/health` response shape:**
  ```json
  {"status":"ok","uptime_seconds":3600,"connections":2,"port":7731,"providers":["claude"]}
  ```
  `providers` list populated from registered adapters (empty array at this story — providers come in 1.3). Response must be within 50ms (NFR3) — serve from in-memory state, no I/O.
- **HTTP handler receiver:** Use `h *Handler` — not `s`, `srv`, `d`, or `self`.
- **`/health` endpoint does NOT require a running poll loop or providers.** It reports what's connected. At this story stage, `providers` is an empty array.

### Project Structure Notes

```
internal/
└── daemon/
    ├── daemon.go       ← Daemon struct, Start(), Shutdown()
    └── handlers.go     ← Handler struct, ServeHTTP, /health handler
cmd/pakai/
└── main.go            ← add daemon start/stop/status subcommands
```

### References

- [Source: architecture.md#API & Communication Patterns] — /health response shape
- [Source: architecture.md#Infrastructure & Deployment] — daemon auto-spawn protocol, PID file path
- [Source: architecture.md#Communication Patterns] — signal handling, SSE WaitGroup (WaitGroup scaffold can be added now, used in 1.5)
- [Source: architecture.md#Enforcement Guidelines] — `{"error":"..."}` shape for HTTP errors
- [Source: epics.md#Story 1.2] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Fixed port binding from `localhost` to `127.0.0.1` in `internal/daemon/server.go`, `client/client.go`
- Updated `/health` response to return `{"status","uptime_seconds","connections","port","providers"}` (was `{"status","uptime"}`)
- Created `internal/daemon/pid.go` with `pidFilePath()` (XDG_RUNTIME_DIR fallback), `isProcessAlive()`, `ensurePIDDir()`, `readPIDFile()`
- Added stale PID recovery in `NewServer` — removes dead PID file before starting
- Moved PID removal to `defer os.Remove(s.pidFile)` in `Start()` (was in `Shutdown()`)
- Replaced `signal.Notify` channel with `signal.NotifyContext` in `runDaemonForeground()`
- Added port conflict detection via `net.Listen` probe with explicit "port X is already in use" message
- Updated `daemon stop` to wait up to 5s for process to exit after SIGTERM
- Fixed `daemon status` to use new `HealthResponse` fields (`UptimeSeconds` int)
- Updated `renderer/dashboard.go` for new `HealthResponse` field name
- Tests added: `daemon_test.go` (9 tests covering health shape, PID management, stale recovery, port conflict, addr-in-use detection)

### File List

- `internal/daemon/server.go` (modified)
- `internal/daemon/pid.go` (new)
- `internal/daemon/daemon_test.go` (new)
- `internal/client/client.go` (modified)
- `cmd/pakai/main.go` (modified)
- `internal/renderer/dashboard.go` (modified)

## Change Log

- 2026-05-12: Story 1.2 implementation — loopback binding, XDG_RUNTIME_DIR PID, defer cleanup, signal.NotifyContext, /health response shape, port conflict detection, stale PID recovery
