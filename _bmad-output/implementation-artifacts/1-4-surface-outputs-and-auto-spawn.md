# Story 1.4: Surface Outputs & Auto-Spawn

Status: review

## Story

As a terminal user,
I want to run `pakai tmux`, `pakai waybar`, and `pakai status` to get formatted Claude usage for my status bar,
so that I can embed live usage data into my terminal environment without manually managing the daemon.

## Acceptance Criteria

1. **Given** the daemon is running and Claude data has `state: "ok"`, **When** I run `pakai tmux`, **Then** it prints a compact plain-text string (e.g. `claude:72%`) and exits with code 0 within 200ms

2. **Given** the daemon is running and Claude data has `state: "ok"`, **When** I run `pakai waybar`, **Then** it prints a JSON object with `text`, `tooltip`, and `class` fields; `class` is `"ok"` and exits with code 0 within 200ms

3. **Given** the daemon is running and Claude data has `state: "ok"`, **When** I run `pakai status`, **Then** it prints a human-readable multi-line usage summary with provider name, usage value, and state, and exits with code 0 within 200ms

4. **Given** the daemon is running, **When** I run `pakai status --json`, **Then** it prints the raw `[]Usage` JSON array identical to `GET /status` and exits with code 0

5. **Given** the daemon is NOT running, **When** I run `pakai tmux`, **Then** the CLI dials localhost:7731, gets connection refused, spawns `pakai daemon start` as a detached background process, polls `GET /health` every 200ms up to 3 seconds, then prints the usage string — total elapsed time under 3 seconds

6. **Given** the daemon fails to become healthy within 3 seconds after auto-spawn, **When** the timeout expires, **Then** `pakai tmux` exits with a non-zero code and an explicit error message

7. **Given** Claude data has `state: "error"`, **When** I run `pakai tmux`, **Then** it prints `claude:??` — not silence, not zero, not an empty string

8. **Given** Claude data has `state: "stale"`, **When** I run `pakai waybar`, **Then** the JSON `class` field is `"stale"` and `tooltip` includes the last successful refresh time

9. **Given** Claude usage exceeds the limit (e.g. 105%), **When** I run `pakai waybar`, **Then** the JSON `class` field is `"over-limit"` and `text` shows the raw percentage (e.g. `claude:105%`)

10. **Given** the daemon was just auto-spawned and has not yet completed its first poll cycle, **When** `GET /status` returns an empty array, **Then** `pakai tmux` prints an empty string and exits with code 0 — empty cache is a valid transient state, not an error

11. **Given** a third-party tool sends `GET http://127.0.0.1:7731/status`, **When** the request is received, **Then** it returns HTTP 200 with the `[]Usage` JSON array — no special headers or auth required

## Tasks / Subtasks

- [x] Create `internal/renderer/` package with pure renderer functions (AC: 1, 2, 3, 7, 8, 9)
  - [x] `tmux.go` — `RenderTmux(usages []*schema.Usage, separator string) string`
    - [x] Format: `provider:72%` or `provider:??` on error/stale
    - [x] `over-limit` state shows raw percent (e.g. `claude:105%`)
    - [x] Default separator: ` | ` between providers
  - [x] `waybar.go` — `RenderWaybar(usages []*schema.Usage, separator string) string`
    - [x] Output JSON: `{"text":"...","tooltip":"...","class":"..."}`
    - [x] `class` values: `"ok"`, `"error"`, `"stale"`, `"over-limit"` (warning/critical added in Epic 4)
    - [x] `tooltip` includes last refresh time on stale; error message on error
  - [x] `status.go` — `RenderStatus(usages []*schema.Usage, refreshedAt time.Time) string`
    - [x] Human-readable multi-line: one row per provider with name, value, state
- [x] Add `pakai tmux`, `pakai waybar`, `pakai status` subcommands in `cmd/pakai/` (AC: 1, 2, 3, 4)
  - [x] Fetch `GET /status` from daemon
  - [x] Pass result through appropriate renderer
  - [x] `pakai status --json` outputs raw JSON from `/status` unchanged
- [x] Implement auto-spawn logic (AC: 5, 6, 10)
  - [x] TCP dial `127.0.0.1:7731` — on connection refused, auto-spawn
  - [x] Spawn: `exec.Command(os.Executable(), "daemon", "start")` as detached process
  - [x] Poll `GET /health` every 200ms up to 3s
  - [x] On timeout: exit non-zero with explicit message
  - [x] On empty `/status` array (first poll not yet fired): print empty string, exit 0
- [x] Add renderer golden file tests (AC: 1, 2, 7, 8, 9)
  - [x] `testdata/*.golden` files for each renderer
  - [x] Test all states: ok, error, stale, over-limit

## Dev Notes

- **Prerequisite:** Story 1.3 (Claude adapter + `/status` endpoint + daemon poll loop) must be complete.
- **Renderer package is pure functions** — no goroutines, no HTTP, no I/O. Signature: `func Render(usages []model.Usage, cfg config.Config) string`. Each renderer is independently testable with golden files.
- **200ms budget (NFR1):** The surface subcommand makes one HTTP call to `GET /status` (daemon responds in <50ms from cache) and formats output. No filesystem access on the request path. Total round-trip including process startup must stay under 200ms with a warm daemon.
- **Auto-spawn protocol (must match exactly):**
  1. `net.Dial("tcp", "127.0.0.1:7731")` — if refused, spawn
  2. `exec.Command(os.Executable(), "daemon", "start")` with `SysProcAttr{Setsid: true}` for detachment
  3. Poll `GET http://127.0.0.1:7731/health` every 200ms, timeout 3s
  4. On timeout: non-zero exit with explicit message
  5. On empty `/status` array: print `""` and exit 0 (AC 10 — this is not an error)
- **CSS class state machine for waybar (this story covers `ok`, `error`, `stale`, `over-limit`):**
  - `ok` — usage below threshold (defaults apply; warning/critical added in Epic 4)
  - `error` — adapter failed
  - `stale` — data past TTL
  - `over-limit` — `Percent != nil && *Percent > 100`
- **`pakai tmux` error/stale display:** `provider:??` — not empty, not zero. The `??` string is the unavailability indicator (FR47).
- **No auth on `/status` (FR42):** Third-party tools query the same endpoint. No CORS headers, no tokens, no special middleware needed.
- **Golden file test pattern:**
  ```go
  // In renderer/tmux_test.go
  got := Render(usages, cfg)
  golden := testdata/tmux_ok.golden
  // compare got == readFile(golden)
  ```

### Project Structure Notes

```
internal/
└── renderer/
    ├── tmux.go         ← pure func Render([]model.Usage, config.Config) string
    ├── tmux_test.go
    ├── waybar.go
    ├── waybar_test.go
    ├── status.go
    ├── status_test.go
    └── testdata/
        ├── tmux_ok.golden
        ├── tmux_error.golden
        ├── tmux_stale.golden
        ├── tmux_over_limit.golden
        ├── waybar_ok.golden
        └── ...
cmd/pakai/
└── main.go    ← add tmux, waybar, status subcommands + auto-spawn logic
```

### References

- [Source: architecture.md#Package Layout] — renderer/ pure functions, no state
- [Source: architecture.md#Infrastructure & Deployment] — daemon auto-spawn protocol (exact steps)
- [Source: architecture.md#Format Patterns] — JSON shapes for waybar, SSE wire format
- [Source: architecture.md#Process Patterns] — stale vs error distinction for CSS classes
- [Source: epics.md#Story 1.4] — acceptance criteria (including patched AC 10 for empty cache)

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Rewrote `internal/renderer/tmux.go` — shows percentage (`claude:72%`) or `??` on error/stale, no longer skips errors, default separator ` | `
- Rewrote `internal/renderer/waybar.go` — class values: `ok`, `error`, `stale`, `over-limit`; handles all states (no longer skips errors); tooltip shows error message or last refresh time; text shows percentage or `??`
- Updated `internal/renderer/status.go` — kept multi-line human-readable format unchanged (already correct)
- Added `pakai status --json` subcommand — outputs raw JSON identical to `GET /status`
- Updated `internal/client/client.go` — `EnsureRunning` now returns `error` instead of `bool`; uses TCP dial first, then `SysProcAttr{Setsid: true}` for detachment, polls every 200ms up to 3s, returns explicit error on timeout
- Created golden file tests: `tmux_test.go` (6 tests), `waybar_test.go` (4 tests)
- Created golden testdata: `testdata/tmux_*.golden` (6 files), `testdata/waybar_*.golden` (4 files)

### File List

- `internal/renderer/tmux.go` (modified)
- `internal/renderer/waybar.go` (modified)
- `internal/renderer/tmux_test.go` (new)
- `internal/renderer/waybar_test.go` (new)
- `internal/renderer/testdata/tmux_ok.golden` (new)
- `internal/renderer/testdata/tmux_error.golden` (new)
- `internal/renderer/testdata/tmux_stale.golden` (new)
- `internal/renderer/testdata/tmux_over_limit.golden` (new)
- `internal/renderer/testdata/tmux_multi.golden` (new)
- `internal/renderer/testdata/tmux_mixed.golden` (new)
- `internal/renderer/testdata/waybar_ok.golden` (new)
- `internal/renderer/testdata/waybar_error.golden` (new)
- `internal/renderer/testdata/waybar_stale.golden` (new)
- `internal/renderer/testdata/waybar_over_limit.golden` (new)
- `internal/client/client.go` (modified)
- `cmd/pakai/main.go` (modified)

## Change Log

- 2026-05-12: Story 1.4 implementation — percentage-based renderers, error/stale state handling, over-limit class, `status --json`, auto-spawn with 200ms polling/3s timeout, golden file tests
