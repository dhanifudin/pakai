---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
lastStep: 8
status: 'complete'
completedAt: '2026-05-12'
inputDocuments: ['_bmad-output/planning-artifacts/prd.md']
workflowType: 'architecture'
project_name: 'pakai'
user_name: 'Dian'
date: '2026-05-12'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:** 50 FRs across 7 capability areas — Provider Data Reading, Daemon Management, Status Surface Delivery, Setup & Configuration, Alert & Adaptive Refresh, Developer & Integration Support, and Graceful Degradation. The FR density is highest in Daemon Management and Graceful Degradation, reflecting the core architectural complexity.

**Non-Functional Requirements:**
- Performance: 200ms end-to-end CLI response (warm daemon); 50ms HTTP endpoint response; 10 concurrent SSE clients without degradation
- Reliability: 7+ day daemon uptime; systemd auto-restart; zero goroutine leaks on disconnect
- Security: localhost-only binding (127.0.0.1); no telemetry; read-only file access only
- Portability: CGo-free binary; XDG Base Directory paths; Linux/amd64 + Linux/arm64

**Scale & Complexity:**
- Primary domain: System daemon + CLI tooling
- Complexity level: Medium-High
- Estimated architectural components: 8–10 distinct Go packages

### Technical Constraints & Dependencies

- **SQLite**: `modernc.org/sqlite` — pure Go, no CGo, read-only DSN (`_mode=ro&_query_only=true`); must tolerate SQLITE_BUSY gracefully
- **Polling over inotify**: adapters use a configurable interval (default 30s) with injected `FileReader` for testability; inotify explicitly prohibited
- **Singleflight**: cache refresh layer must deduplicate concurrent requests at TTL boundary
- **SSE goroutines**: every handler selects on `r.Context().Done()`; `sync.WaitGroup` tracking; max 10 clients enforced at accept time
- **Signal handling**: `signal.NotifyContext` for SIGTERM/SIGINT; 5s drain timeout; PID file removed in `defer`, not signal handler
- **Provider interface**: `Fetch(ctx context.Context) (Usage, error)` with injected dependencies (`io.Reader` or `*sql.DB`) — path resolution at call site, not inside adapter

### Cross-Cutting Concerns Identified

1. **Daemon lifecycle**: auto-spawn (port liveness probe), port conflict detection, PID file management, systemd unit generation, graceful shutdown with drain
2. **Concurrency model**: SSE goroutine lifecycle, singleflight at cache layer, non-blocking reads serving surfaces while polling runs in background
3. **Provider isolation**: independent `ok`/`error`/`stale` state machine per provider; adapter panics recovered and logged; no cross-provider error propagation
4. **Configuration observability**: live reload (FR32) requires daemon's poll loop to observe config changes; config is not a startup-only concern
5. **Observability**: all adapter errors captured with full context (path, field, raw value); never swallowed; surfaced via `pakai provider debug`

## Starter Template & Project Initialization

### Primary Technology Domain

Go CLI + background daemon. No external starter generator — project initialization is `go mod init` + intentional package layout. Tech stack fully decided in PRD and brainstorming; this step documents and validates those decisions.

### Initialization

```bash
go mod init github.com/dhanifudin/pakai
go get github.com/spf13/cobra@latest
go get github.com/BurntSushi/toml@latest
go get modernc.org/sqlite@latest
go get golang.org/x/sync@latest              # import path: golang.org/x/sync/singleflight
go get github.com/fsnotify/fsnotify@latest   # config live reload watcher
go get github.com/stretchr/testify@latest    # test only: assert + require
# log/slog  — stdlib (Go 1.21+), no go get needed
# net/http/httptest — stdlib, covers daemon endpoint tests
```

**Dropped:** `github.com/spf13/viper` — silently normalizes config keys to lowercase, corrupting provider-namespaced keys (e.g. `provider.opencode.limit`). Replaced by `BurntSushi/toml` + `fsnotify` direct. `internal/config` owns a plain `Config` struct with a live-reading `Config()` accessor function; config is never snapshotted at daemon startup.

### Package Layout

```
github.com/dhanifudin/pakai/
├── cmd/pakai/              ← CLI entry point (cobra root + subcommands)
├── internal/
│   ├── model/              ← Usage, ProviderStatus structs — zero deps on other internals;
│   │                         hub of the dependency graph imported by all other packages
│   ├── daemon/             ← HTTP server, SSE hub, adaptive poll loop
│   │   └── hub.go          ← SSE fan-out, WaitGroup tracking, initial snapshot on subscribe
│   ├── providers/
│   │   ├── claude/         ← adapter: injected io.Reader dependency
│   │   ├── opencode/       ← adapter: injected *sql.DB dependency
│   │   └── mock/           ← mock provider (FR40): implements Provider interface,
│   │                         in-memory state, swapped in via HTTP PUT/DELETE /mock
│   ├── cache/              ← TTL store + singleflight; pure data structure, no goroutines
│   ├── renderer/           ← tmux.go, waybar.go, status.go — pure functions over Usage
│   ├── config/             ← TOML decode, fsnotify watcher + 5s fallback poll,
│   │                         live Config() accessor (never snapshot at startup)
│   ├── detect/             ← provider path probing; returns config suggestions
│   └── systemd/            ← systemd user unit template, enable/disable
├── internal/testutil/      ← shared test helpers: seedDB(*sql.DB), fixturePath(t, name)
├── widget/                 ← Tauri widget (post-MVP, Rust)
└── docs/setup/             ← waybar, tmux, starship integration guides
```

### Architectural Decisions Established

- **`internal/model/`** is the dependency root — imported by all packages, imports none. Prevents import cycles between daemon, providers, renderer, and cache.
- **Poll loop lives in `daemon/`** — owns the lifecycle (start/stop tied to daemon startup/shutdown, adaptive interval driven by threshold state). `cache/` is a pure TTL data structure with no goroutines — fully unit-testable without timing dependencies.
- **`config/`** exposes a live-reading `Config()` function. Daemon poll loop reads fresh config on every cycle; changes propagate within one interval (FR32) without restart.
- **`detect/`** and **`systemd/`** are separate packages — provider path probing and unit file templating have distinct test surfaces and collaborators.
- **`renderer/`** contains separate files per surface (`tmux.go`, `waybar.go`, `status.go`); each is a pure function `func(model.Usage, config.Config) string` — no state, no imports of daemon or providers. Golden file tests via `testdata/*.golden`.
- **`log/slog`** (stdlib) is the single logging interface across all packages — no mixed `fmt.Println` / `log.Printf`.
- **SSE hub** lives in `daemon/hub.go` — not extracted to a separate package; not reusable outside daemon context.

## Core Architectural Decisions

### Decision Priority Analysis

**Already locked (from PRD, brainstorming, step 3):**
- No user-owned persistent storage — local file reads only
- No authentication — localhost is user-scoped
- REST + SSE over HTTP/1.1 at localhost:7731
- Go 1.21+ (log/slog stdlib requirement)
- Single binary distribution

**Decided in this step:**
- Cache TTL = poll interval + 10s buffer
- Usage struct schema (`internal/model`)
- HTTP API response shapes
- CI/CD pipeline design
- Daemon auto-spawn protocol

### Data Architecture

**Cache TTL:** poll interval + 10s buffer (default: 40s at base rate). A single missed poll cycle yields `stale`, not `error`. `error` is reserved for adapter failures (file missing, parse error, SQLITE_BUSY unresolvable).

**Core model (`internal/model/model.go`):**

```go
type State string

const (
    StateOK    State = "ok"
    StateError State = "error"
    StateStale State = "stale"
)

type Usage struct {
    Provider    string
    Label       string
    State       State
    Percent     *float64  // nil = no limit configured
    Cost        *float64  // nil = not cost-based
    RefreshedAt time.Time
    ErrorMsg    string    // non-empty on error or stale
}
```

`Percent` and `Cost` are pointers: nil distinguishes "no data" from "zero usage".

### Authentication & Security

None for MVP. `localhost:7731` is user-scoped on Linux. No bearer token, no TLS. Revisit if remote access use cases emerge post-MVP.

### API & Communication Patterns

| Endpoint | Method | Response |
|---|---|---|
| `/status` | GET | `200 []Usage` JSON |
| `/health` | GET | `200 {"status","uptime_seconds","connections","port","providers"}` |
| `/events` | GET | SSE stream — `event: update`, `data: []Usage`; heartbeat every 30s |
| `/mock/:name` | PUT | body `{"percent":N}` or `{"cost":N}` → 200 or 400 |
| `/mock/:name` | DELETE | 200 |
| Errors | any | `{"error":"<message>"}` + appropriate HTTP status |

### Infrastructure & Deployment

**Go version:** 1.21+ (log/slog stdlib).

**CI/CD (GitHub Actions):**
- PR/push to main: `go test ./...` + `go vet ./...` + `staticcheck` (Linux/amd64)
- Integration tests: `go test -tags integration ./...` (separate job)
- Tag push `v*`: goreleaser → Linux/amd64 + Linux/arm64 binaries, GitHub release
- AUR PKGBUILD: manual update on release

**Daemon auto-spawn protocol:**
1. TCP dial `localhost:7731` — if refused, spawn
2. `exec.Command(os.Executable(), "daemon", "start")` → log to `$XDG_RUNTIME_DIR/pakai/daemon.log`
3. Poll `/health` every 200ms, timeout 3s
4. PID file: `$XDG_RUNTIME_DIR/pakai/daemon.pid` — written at start, removed in `defer`

### Decision Impact Analysis

**Implementation Sequence:**
1. `internal/model/` — structs first; everything imports this
2. `internal/cache/` — pure TTL store; testable without goroutines
3. `internal/providers/{claude,opencode}/` — adapters against interface
4. `internal/config/` — TOML decode + fsnotify watcher
5. `internal/daemon/` — HTTP server, SSE hub, poll loop
6. `internal/renderer/` — pure functions over `model.Usage`
7. `cmd/pakai/` — cobra wiring, auto-spawn logic
8. `internal/detect/`, `internal/systemd/` — setup UX, last

**Cross-Component Dependencies:**
- `model/` ← imported by all; imports nothing
- `cache/` ← depends on `model/`; no goroutines (tested without timing)
- `providers/` ← depends on `model/`; injected `io.Reader` or `*sql.DB`
- `daemon/` ← depends on `model/`, `cache/`, `providers/`, `config/`; owns poll loop lifecycle
- `renderer/` ← depends on `model/`, `config/`; pure functions, no side effects
- `cmd/pakai/` ← depends on `daemon/`, `renderer/`, `config/`, `detect/`, `systemd/`

## Implementation Patterns & Consistency Rules

### Naming Patterns

**Go Identifiers:**
- Exported types/functions: `PascalCase` — `Usage`, `ProviderStatus`, `NewDaemon`
- Unexported identifiers: `camelCase` — `pollInterval`, `sseClients`
- Package names: lowercase, single word, no underscores — `daemon`, `providers`, `testutil`
- File names: `snake_case.go` — `hub.go`, `claude_adapter.go`
- Test files: `<file>_test.go` co-located with the file under test — never a separate `tests/` directory
- Constants: `PascalCase` for exported (`StateOK`), `camelCase` for package-private (`defaultInterval`)

**Provider Interface:**
The canonical method name is `Fetch`. No variations allowed:
```go
type Provider interface {
    Fetch(ctx context.Context) (model.Usage, error)
}
```

**HTTP Handler Receivers:** Use `h *Handler` — not `s`, `srv`, `d`, or `self`.

**Test Variables:** Always `got` / `want` — never `actual` / `expected`.

### Structure Patterns

**Test Organization:**
- `_test.go` files co-located with implementation — never a top-level `tests/` directory
- Shared test helpers only in `internal/testutil/`
- Golden files at `testdata/<test_name>.golden` relative to the test file
- Integration tests tagged `//go:build integration` (new-style; NOT the old `// +build` syntax)

**Dependency Injection:**
All external I/O is injected, never resolved inside an adapter:
```go
// CORRECT — path resolved at call site, reader injected
func NewClaudeAdapter(r io.Reader) *ClaudeAdapter

// WRONG — untestable
func NewClaudeAdapter(path string) *ClaudeAdapter
```

**Config Access:**
`config.Config()` is called fresh at every use site — never assigned to a local variable at startup:
```go
// CORRECT
cfg := config.Config()

// WRONG — captured at startup, misses live reload
type Daemon struct { cfg config.Config }
```

### Format Patterns

**JSON Response Shapes:**

Success responses are direct — no envelope wrapper:
```json
// /status
[{"provider":"claude","label":"Claude","state":"ok","percent":72.4,"refreshedAt":"2026-05-12T10:00:00Z"}]

// /health
{"status":"ok","uptime_seconds":3600,"connections":2,"port":7731,"providers":["claude","opencode"]}
```

Error responses use exactly `{"error":"<message>"}` — not `message`, not `errors`, not `detail`:
```json
{"error":"provider not found: badname"}
```

**JSON Field Names:** `camelCase` with explicit `json:"fieldName"` struct tags. No `snake_case` on the wire.

**Time Serialization:** `time.RFC3339` for all JSON time fields — no Unix timestamps in HTTP responses.

**SSE Wire Format:**
```
event: update
data: [<json array of Usage>]

event: heartbeat
data: {}

```
Two newlines terminate each event. Heartbeat every 30s. Initial snapshot sent immediately on connect before the first poll fires.

### Communication Patterns

**SSE Goroutine Contract:**
Every SSE handler goroutine must select on `r.Context().Done()` and register/deregister with the hub `sync.WaitGroup`:
```go
func (h *Hub) Subscribe(w http.ResponseWriter, r *http.Request) {
    h.wg.Add(1)
    defer h.wg.Done()
    for {
        select {
        case <-r.Context().Done():
            return
        case snapshot := <-ch:
            // write event
        }
    }
}
```

**Singleflight Keys:** One key per provider — `"refresh:claude"`, `"refresh:opencode"`. Never a single `"refresh"` key (that serializes unrelated providers).

**Logging:** `log/slog` only — zero `fmt.Println` or `log.Printf`. Every error log includes provider, field, and raw value:
```go
slog.Error("parse failed", "provider", "claude", "field", "tokens_in", "raw", rawVal, "err", err)
```

### Process Patterns

**Provider Error Isolation:**
Each provider's `ok`/`error`/`stale` state is independent — a Claude parse failure must not affect OpenCode's state. Poll loop recovers from adapter panics via `recover()`, logs them, and sets that provider's state to `error`.

**Stale vs Error Distinction:**
- `stale`: data exists in cache but past TTL (missed poll) — `ErrorMsg` states last refresh time
- `error`: adapter returned error or panicked — `ErrorMsg` states the failure
- Never collapse these; renderers apply different CSS classes to each

**SQLITE_BUSY Handling:** OpenCode adapter uses `?_busy_timeout=5000` in DSN. Persistent SQLITE_BUSY beyond timeout → `error` state, not `stale`.

**Context Propagation:** `context.Context` flows top-down from HTTP request → poll tick → `Provider.Fetch()`. Never pass `context.Background()` inside a handler where the request context is available.

### Enforcement Guidelines

**All agents MUST:**
- Use `Fetch(ctx context.Context) (model.Usage, error)` — no other method names on Provider
- Return `{"error":"..."}` for all HTTP error responses — no other error shapes
- Co-locate tests as `_test.go` files — never a `tests/` directory
- Use `//go:build integration` build tags — never old `// +build` style
- Call `config.Config()` fresh at each use site — never snapshot at startup
- Log via `slog` only — zero `fmt.Println` / `log.Printf`
- Inject I/O dependencies — never resolve paths inside adapters

**Anti-Patterns to Reject:**
```go
// WRONG — config snapshotted
type Daemon struct { cfg config.Config }

// WRONG — path inside adapter
func NewClaudeAdapter() *ClaudeAdapter { path := os.ExpandEnv("~/.claude/...") }

// WRONG — collapsed stale/error
if err != nil { usage.State = model.StateError } // even on cache miss (should be StateStale)

// WRONG — single singleflight key serializes all providers
sf.Do("refresh", func() { refreshAll() })

// WRONG — context lost
go func() { provider.Fetch(context.Background()) }() // inside handler
```

## Project Structure & Boundaries

### Complete Project Directory Structure

```
github.com/dhanifudin/pakai/
├── .github/
│   └── workflows/
│       ├── ci.yml                  ← go test + go vet + staticcheck on PR/push main
│       └── release.yml             ← goreleaser on v* tag push
├── .goreleaser.yaml                ← Linux/amd64 + arm64; CGO_ENABLED=0
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                        ← build, test, lint, run-daemon targets
├── cmd/
│   └── pakai/
│       └── main.go                 ← cobra root; subcommand registration
├── internal/
│   ├── model/
│   │   └── model.go                ← Usage, State; imports nothing
│   ├── cache/
│   │   ├── cache.go                ← TTL store + singleflight; no goroutines
│   │   └── cache_test.go
│   ├── config/
│   │   ├── config.go               ← Config struct; TOML decode; Config() accessor
│   │   ├── watcher.go              ← fsnotify + 5s fallback poll
│   │   └── config_test.go
│   ├── providers/
│   │   ├── provider.go             ← Provider interface definition
│   │   ├── claude/
│   │   │   ├── adapter.go          ← injected io.Reader; stats-cache.json parser
│   │   │   ├── adapter_test.go
│   │   │   └── testdata/
│   │   │       ├── stats_valid.json
│   │   │       ├── stats_malformed.json
│   │   │       └── stats_missing_fields.json
│   │   ├── opencode/
│   │   │   ├── adapter.go          ← injected *sql.DB; SQLite query + aggregation
│   │   │   └── adapter_test.go     ← uses testutil.SeedDB
│   │   └── mock/
│   │       ├── provider.go         ← in-memory Provider; state set via HTTP
│   │       └── provider_test.go
│   ├── daemon/
│   │   ├── server.go               ← http.Server setup; route registration; shutdown
│   │   ├── handlers.go             ← /status, /health, /events, /mock/:name handlers
│   │   ├── hub.go                  ← SSE fan-out; WaitGroup; initial snapshot on connect
│   │   ├── poll.go                 ← adaptive poll loop; ticker; threshold logic
│   │   ├── spawn.go                ← auto-spawn: dial → exec → poll /health
│   │   └── daemon_test.go          ← httptest-based; integration tag for DB tests
│   ├── renderer/
│   │   ├── tmux.go                 ← Tmux(u []model.Usage, cfg config.Config) string
│   │   ├── waybar.go               ← Waybar(u []model.Usage, cfg config.Config) string
│   │   ├── status.go               ← Status(u []model.Usage, cfg config.Config) string
│   │   ├── tmux_test.go
│   │   ├── waybar_test.go
│   │   ├── status_test.go
│   │   └── testdata/
│   │       ├── tmux_ok.golden
│   │       ├── tmux_warning.golden
│   │       ├── tmux_critical.golden
│   │       ├── tmux_over_limit.golden
│   │       ├── tmux_error.golden
│   │       ├── tmux_stale.golden
│   │       ├── waybar_ok.golden
│   │       ├── waybar_error.golden
│   │       └── status_all_providers.golden
│   ├── detect/
│   │   ├── detect.go               ← provider path probing; returns config suggestions
│   │   └── detect_test.go
│   ├── systemd/
│   │   ├── unit.go                 ← user unit template; enable/disable/status
│   │   └── unit_test.go
│   └── testutil/
│       ├── testutil.go             ← SeedDB(*sql.DB, rows); FixturePath(t, name)
│       └── testdata/
│           └── opencode_seed.sql   ← canonical seed data for opencode adapter tests
└── docs/
    └── setup/
        ├── tmux.md
        ├── waybar.md
        └── starship.md
```

### Architectural Boundaries

**API Boundary — External (localhost:7731):**

All status surfaces and external tools cross this boundary via HTTP. The boundary is the only point where `[]model.Usage` is serialized to JSON or SSE wire format.

| Boundary crossing | Direction | Format |
|---|---|---|
| `pakai tmux` / `pakai waybar` / `pakai status` | CLI → daemon | HTTP GET /status; JSON → rendered string |
| SSE subscriber (waybar live, starship) | client → daemon | HTTP GET /events; SSE stream |
| Mock control (FR40) | CLI → daemon | HTTP PUT/DELETE /mock/:name |
| Health probe (auto-spawn) | CLI → daemon | HTTP GET /health |

**Provider Boundary — Internal:**

The `Provider` interface is the seam between daemon orchestration and data source specifics. Nothing in `daemon/` imports `providers/claude` or `providers/opencode` directly — only the `Provider` interface from `providers/provider.go`. Concrete adapters are wired in `cmd/pakai/main.go` at startup.

**Cache Boundary:**

`internal/cache/` is a pure data structure — no goroutines, no network, no file I/O. The poll loop in `daemon/poll.go` is the only writer; handlers in `daemon/handlers.go` are the only readers. Keeps cache fully testable without timing dependencies.

**Config Boundary:**

`internal/config/` is the sole reader of `$XDG_CONFIG_HOME/pakai/config.toml`. All other packages call `config.Config()` — they never parse TOML or stat files themselves.

### Requirements to Structure Mapping

| FR Category | Files |
|---|---|
| FR1–8 Provider Data Reading | `model/model.go`, `providers/provider.go`, `providers/claude/adapter.go`, `providers/opencode/adapter.go`, `cache/cache.go` |
| FR9–20 Daemon Management | `daemon/server.go`, `daemon/handlers.go`, `daemon/hub.go`, `daemon/poll.go`, `daemon/spawn.go` |
| FR21–28 Status Surface Delivery | `renderer/tmux.go`, `renderer/waybar.go`, `renderer/status.go`, `cmd/pakai/main.go` |
| FR29–36 Setup & Configuration | `config/config.go`, `config/watcher.go`, `detect/detect.go`, `systemd/unit.go` |
| FR37–38 Adaptive Refresh | `daemon/poll.go` (threshold table + ticker reset) |
| FR39–41 Developer & Integration | `providers/mock/provider.go`, `daemon/handlers.go` (/mock endpoints) |
| FR42–50 Graceful Degradation | `providers/` (isolation + panic recovery in poll loop), `cache/cache.go` (stale state) |

### Integration Points

**Data Flow — Normal Poll Cycle:**
```
poll ticker fires (daemon/poll.go)
  → for each provider: Provider.Fetch(ctx)   [concurrent via goroutines]
      → adapter reads file or queries SQLite
      → returns model.Usage{State: ok/error}
  → results stored in cache (TTL = interval + 10s)
  → []model.Usage sent to hub channel
      → hub fans out to all connected SSE clients
```

**Data Flow — CLI Surface Request:**
```
pakai tmux (cmd/pakai/main.go)
  → dial localhost:7731 (auto-spawn via spawn.go if refused)
  → GET /status → []model.Usage JSON
  → renderer.Tmux(usage, config.Config()) → stdout
```

**Data Flow — Live SSE Client:**
```
GET /events (daemon/handlers.go → hub.go)
  → send initial snapshot from cache immediately
  → loop: select on hub channel or ctx.Done()
      → write SSE update event on new poll result
      → write SSE heartbeat event every 30s
```

**Config Live Reload:**
```
fsnotify detects config.toml write (config/watcher.go)
  → invalidates internal config state
  → next poll cycle: config.Config() returns new values
  → new poll interval takes effect on next ticker reset
```

### File Organization Patterns

**Runtime Files (not in repo):**
- `$XDG_CONFIG_HOME/pakai/config.toml` — user config
- `$XDG_RUNTIME_DIR/pakai/daemon.pid` — PID file; created at start, removed in `defer`
- `$XDG_RUNTIME_DIR/pakai/daemon.log` — daemon stdout/stderr

**Test Data Placement:**
- Provider adapter fixtures in `internal/providers/<name>/testdata/` — local to the adapter
- Renderer golden files in `internal/renderer/testdata/` — named by state variant
- OpenCode seed SQL in `internal/testutil/testdata/opencode_seed.sql` — shared via `testutil.SeedDB()`

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**
All technology choices are mutually compatible. `modernc.org/sqlite` is the only SQLite driver and is CGo-free — directly enabling the `CGO_ENABLED=0` goreleaser build. `BurntSushi/toml` + `fsnotify` replace viper without key-normalization risk. Go 1.21+ is required for `log/slog` and this constraint is consistently applied across all packages.

**Pattern Consistency:**
The injected-dependency pattern is uniform: `io.Reader` for Claude, `*sql.DB` for OpenCode, both resolved at call site in `cmd/pakai/main.go`. The `config.Config()` live-accessor pattern is consistent — no package snapshots config at startup. Logging is `log/slog` only with no exceptions. Test co-location (`_test.go`) is consistent across all packages.

**Structure Alignment:**
`internal/model/` as the import root prevents all known cycle risks (daemon ↔ providers ↔ renderer ↔ cache). The cache boundary (pure data structure, poll loop as sole writer) directly supports testability without timing. The SSE hub staying in `daemon/hub.go` avoids premature extraction of a component with no other consumers.

### Requirements Coverage Validation ✅

**Functional Requirements — all 7 FR categories:**

| FR Category | Coverage | Notes |
|---|---|---|
| FR1–8 Provider Data Reading | ✅ Full | Claude io.Reader + OpenCode *sql.DB; TTL stale/error distinction |
| FR9–20 Daemon Management | ✅ Full | server, hub, poll, spawn files cover all sub-requirements |
| FR21–28 Status Surface Delivery | ✅ Full | Three renderer functions + cobra subcommands |
| FR29–36 Setup & Configuration | ✅ Full | config + watcher + detect + systemd |
| FR37–38 Adaptive Refresh | ✅ Full | Threshold table in poll.go; config read fresh each cycle |
| FR39–41 Developer & Integration | ✅ Full | mock provider + /mock endpoints; debug command in cmd/pakai |
| FR42–50 Graceful Degradation | ✅ Full | Per-provider isolation, panic recovery, stale propagation, auto-recovery |

**Non-Functional Requirements:**

| NFR | Addressed by |
|---|---|
| 200ms end-to-end (warm daemon) | In-memory cache read + renderer pure function |
| 50ms HTTP endpoint | Cache read is O(1); no DB query on /status |
| 10 concurrent SSE clients | Enforced at accept time in hub.go |
| 7+ day daemon uptime | systemd auto-restart + PID file in defer |
| Zero goroutine leaks | WaitGroup + ctx.Done() selection on every SSE goroutine |
| localhost-only binding | `127.0.0.1:7731` hardcoded in server.go |
| No telemetry | No outbound HTTP; read-only file access |
| CGo-free binary | modernc.org/sqlite; CGO_ENABLED=0 in goreleaser |
| XDG Base Directory paths | All runtime/config paths use XDG env vars |
| Linux/amd64 + Linux/arm64 | goreleaser targets both |

### Implementation Readiness Validation ✅

**Decision Completeness:** All critical decisions documented with rationale and versions. Go 1.21+ constraint stated and explained. Viper rejection documented with root cause (key normalization). Package layout decisions each have explicit rationale. Anti-patterns have concrete code examples, not just descriptions.

**Structure Completeness:** Every file in the directory tree has a one-line description of its responsibility. All integration points have explicit data-flow diagrams. All runtime files that live outside the repo are documented. Test data placement is specified per package.

**Pattern Completeness:** All six pattern categories defined (naming, structure, format, communication, process, enforcement). Each enforcement rule has a matching anti-pattern code block. Singleflight key design is explicit (`"refresh:claude"` vs single key). SSE goroutine contract has a canonical code template.

### Gap Analysis Results

**Critical Gaps:** None.

**Important Gaps:**
- `pakai provider debug` (FR41): lives as a cobra subcommand in `cmd/pakai/main.go`; requires a `GET /debug` route added to `daemon/handlers.go` returning raw per-provider state including error context. Non-blocking — self-contained addition.

**Nice-to-Have:**
- Makefile target list not enumerated (infer: `make build`, `make test`, `make lint`, `make run`)
- `.goreleaser.yaml` internal structure not documented (standard goreleaser config)

### Architecture Completeness Checklist

**Requirements Analysis**
- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed (medium-high, 8–10 packages)
- [x] Technical constraints identified (CGo-free, polling over inotify, SQLITE_BUSY, singleflight)
- [x] Cross-cutting concerns mapped (5 concerns: daemon lifecycle, concurrency, provider isolation, config observability, observability)

**Architectural Decisions**
- [x] Critical decisions documented with versions (Go 1.21+, all deps with rationale)
- [x] Technology stack fully specified (final dep list with dropped alternatives)
- [x] Integration patterns defined (REST+SSE API table, SSE wire format)
- [x] Performance considerations addressed (cache TTL, singleflight, in-memory reads)

**Implementation Patterns**
- [x] Naming conventions established (PascalCase, camelCase, file naming, test variables)
- [x] Structure patterns defined (test co-location, dependency injection, config access)
- [x] Communication patterns specified (SSE goroutine contract, singleflight keys, logging)
- [x] Process patterns documented (stale vs error, SQLITE_BUSY, context propagation)

**Project Structure**
- [x] Complete directory structure defined (every file annotated)
- [x] Component boundaries established (API, Provider, Cache, Config boundaries)
- [x] Integration points mapped (three data-flow diagrams)
- [x] Requirements to structure mapping complete (FR-to-file table)

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** High — all 16 checklist items confirmed; no critical gaps; one minor gap (debug endpoint) is self-contained and non-blocking.

**Key Strengths:**
- Import cycle risk eliminated at design time via `internal/model/` hub pattern
- Cache testability guaranteed by removing goroutines from the cache package
- Live reload correctness guaranteed by `config.Config()` accessor pattern — no snapshot drift possible
- Consistent anti-pattern examples give implementation agents explicit rejection criteria
- Full FR-to-file mapping means no agent needs to decide where a feature belongs

**Areas for Future Enhancement:**
- `pakai dashboard` Bubble Tea TUI (post-MVP)
- Rust + Tauri widget (post-MVP — separate `widget/` subtree)
- AUR PKGBUILD automation (currently manual on release)
- Remote access / bearer token auth (out of scope MVP)

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all architectural decisions exactly as documented — no re-debating decided items
- Use implementation patterns consistently; reject any code matching the documented anti-patterns
- Respect package boundaries — concrete adapters wired only in `cmd/pakai/main.go`
- Refer to the FR-to-file mapping table when determining where a feature belongs

**Implementation Start:**
```bash
go mod init github.com/dhanifudin/pakai
# implement internal/model/model.go first — everything else imports it
```
