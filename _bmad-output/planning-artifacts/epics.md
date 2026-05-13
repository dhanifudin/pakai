---
stepsCompleted: [1, 2, 3, 4]
status: 'complete'
completedAt: '2026-05-12'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
---

# pakai - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for pakai, decomposing the requirements from the PRD and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

**Provider Data Reading**
- FR1: The system can read Claude usage data from a local stats cache file without making network requests
- FR2: The system can query OpenCode Go usage from a local SQLite database without making network requests
- FR3: The system can aggregate OpenCode Go usage by provider and calendar billing month from per-message records
- FR4: The system can express Claude usage as a percentage of a subscription limit
- FR5: The system can express OpenCode Go usage as an absolute USD cost when no limit is configured
- FR6: The system can express OpenCode Go usage as a percentage when a manual limit is configured
- FR7: The system can display usage exceeding a limit as-is (e.g. `105%`) without capping at 100%
- FR8: The system can distinguish between a provider data source being inaccessible versus returning data that has not changed within a configured staleness threshold — these are separate, named states
- FR9: The system can indicate when provider data was last successfully refreshed

**Daemon Management**
- FR10: The daemon can serve current provider usage data on a locally accessible HTTP endpoint
- FR11: The daemon can stream live usage updates to connected clients via server-sent events
- FR12: The daemon can report its operational health, uptime, active connection count, and listening port
- FR13: Any CLI subcommand can automatically start the daemon by probing the configured port for liveness; if the port is not responding, the CLI spawns the daemon and waits for it to become healthy before proceeding
- FR14: The daemon can detect when its configured port is already in use by another process and report the conflict explicitly
- FR15: Users can explicitly start, stop, and query the daemon from the CLI
- FR16: The daemon can perform an orderly shutdown on receiving a termination signal
- FR17: A new SSE subscriber receives the current provider usage state immediately on connection, without waiting for the next poll cycle

**Status Surface Delivery**
- FR18: Users can retrieve a compact plain-text usage string for embedding in a tmux status bar
- FR19: Users can retrieve a structured JSON payload for embedding in a waybar custom module
- FR20: Users can retrieve a human-readable multi-provider usage summary for direct terminal display
- FR21: Users can configure custom per-provider display labels in all surfaces
- FR22: Users can configure the separator character between provider entries in compact surfaces
- FR23: The waybar payload includes a CSS class field with one of the following values: `ok`, `warning`, `critical`, `over-limit`, `error`, or `stale` — corresponding to the provider's current state and threshold zone
- FR24: `pakai status` supports a machine-readable JSON output mode
- FR25: Users can configure the display order of providers in compact surface output
- FR26: The setup command displays a live preview of what each configured surface will output after setup completes, confirming that data is being read correctly

**Setup & Configuration**
- FR27: Users can run a setup command that detects which providers are installed based on known local file paths
- FR28: Users can run a setup command that creates a systemd user service unit for automatic daemon startup
- FR29: Users can run a setup command that prints a copy-pasteable configuration snippet for a named surface (tmux or waybar)
- FR30: Running setup again on an existing installation adds newly detected providers to the configuration without modifying existing provider configuration entries or disrupting a running daemon
- FR31: Users can read, write, and list individual configuration values from the CLI
- FR32: Configuration changes written via the CLI take effect within one daemon poll cycle, without restarting the daemon
- FR33: Individual providers can be enabled or disabled in configuration
- FR34: The system operates with default configuration when no config file is present
- FR35: The system validates configuration values on write and rejects invalid types or out-of-range values with an explicit error

**Alert & Adaptive Refresh**
- FR36: Users can configure percentage thresholds that define alert zones (warning, critical, over-limit)
- FR37: The daemon automatically adjusts its provider poll interval based on current usage relative to configured thresholds
- FR38: Surface outputs reflect the current threshold zone using the CSS class values defined in FR23
- FR39: Alert threshold configuration applies uniformly across all enabled providers

**Developer & Integration Support**
- FR40: Users can inject synthetic usage data for a provider to simulate arbitrary usage states; injected data is held in memory and does not persist across daemon restarts
- FR41: Users can remove injected synthetic data to restore live provider readings
- FR42: Third-party tools can query current usage data from the daemon's local HTTP endpoint
- FR43: Third-party tools can subscribe to live usage updates via the daemon's SSE endpoint
- FR44: Users can generate shell completion scripts for bash, zsh, and fish
- FR45: All CLI subcommands exit with status code 0 on success and a non-zero code on error

**Graceful Degradation**
- FR46: Each provider maintains an independent health state (`ok`, `error`, or `stale`) that does not affect other providers
- FR47: When a provider's data source is inaccessible, the surface shows an explicit unavailability indicator (e.g. `??`) rather than zero or silence
- FR48: The system operates normally with a subset of configured providers when others are unreachable
- FR49: Users can view per-provider raw data and error details to diagnose adapter failures
- FR50: When a provider's data source becomes accessible again after an error or stale state, the daemon automatically restores the provider to an `ok` state without user intervention

### NonFunctional Requirements

**Performance**
- NFR1: `pakai tmux` and `pakai waybar` must return within 200ms when the daemon is running and the cache is warm
- NFR2: End-to-end response (including auto-spawn) must complete within 3 seconds on first call from a cold start
- NFR3: The daemon's `/status` and `/health` HTTP endpoints must respond within 50ms under normal operating load
- NFR4: The daemon must serve up to 10 concurrent SSE clients without throughput degradation
- NFR5: Cache reads serving surface subcommands and background provider polling must run on separate goroutines — a slow provider fetch never blocks a `pakai tmux` call

**Reliability**
- NFR6: The daemon must run continuously for 7+ days without crash or measurable heap growth
- NFR7: A crashed daemon must be automatically restarted by systemd within 5 seconds
- NFR8: Provider adapter failures must not cause daemon exit — failed adapters return an error state; panics are recovered and logged
- NFR9: The daemon must not leak goroutines on SSE client disconnect
- NFR10: The daemon must not leave a stale PID file on unclean exit

**Security**
- NFR11: The HTTP endpoint binds exclusively to 127.0.0.1 — never 0.0.0.0 or any external interface
- NFR12: No usage data leaves the local machine — no telemetry, no analytics, no external API calls
- NFR13: Provider source files are opened read-only
- NFR14: All file access runs under the invoking user's permissions — no privilege escalation

**Portability**
- NFR15: The Go binary must compile without CGo
- NFR16: Pre-built releases target Linux/amd64 and Linux/arm64
- NFR17: All config, cache, and runtime paths follow the XDG Base Directory Specification
- NFR18: The binary must install via `go install github.com/dhanifudin/pakai/cmd/pakai@latest`

**Observability**
- NFR19: All provider parse errors are captured with full context and surfaced through `pakai provider debug`
- NFR20: Daemon log output goes to stderr at configurable verbosity; default is silent during normal operation
- NFR21: The `/health` response includes structured fields for uptime, active connections, port, and per-provider health
- NFR22: `pakai provider debug <name>` displays the exact path, raw data, and parse error

**Accessibility**
- NFR23: Alert threshold states must be communicated through both a visual state (CSS class) and a textual/numeric indicator — not color alone

### Additional Requirements

From Architecture — technical requirements that affect implementation:

- **Repo init**: `go mod init github.com/dhanifudin/pakai` with exact dependency set (cobra, BurntSushi/toml, modernc.org/sqlite, golang.org/x/sync, fsnotify/fsnotify, stretchr/testify)
- **Package order constraint**: `internal/model/` must be implemented first — it is the import root; all other packages depend on it
- **SQLite DSN**: OpenCode adapter must use `?_mode=ro&_query_only=true&_busy_timeout=5000` — no other DSN form is acceptable
- **SSE goroutine contract**: every SSE handler goroutine must select on `r.Context().Done()` and register/deregister with the hub `sync.WaitGroup`
- **Signal handling**: daemon shutdown uses `signal.NotifyContext` for SIGTERM/SIGINT; PID file removed in `defer`, not in signal handler; 5s drain timeout
- **Config accessor**: `config.Config()` must be called fresh at each use site — never captured at startup (enables live reload)
- **Singleflight keys**: per-provider keys `"refresh:claude"`, `"refresh:opencode"` — never a single global key
- **CI/CD**: GitHub Actions — `go test ./...` + `go vet ./...` + `staticcheck` on PR; goreleaser on `v*` tag push for Linux/amd64 + arm64
- **Adapter wiring**: concrete adapters wired only in `cmd/pakai/main.go` — `daemon/` package sees only the `Provider` interface
- **`/debug` endpoint**: `GET /debug` route needed in `daemon/handlers.go` for `pakai provider debug` (FR49)

### UX Design Requirements

_Not applicable — pakai is a terminal-native CLI/daemon with no interactive UI surfaces requiring UX design specification. Surface output formatting (tmux string, waybar JSON) is fully specified in FR18–FR25 and the Architecture renderer contracts._

### FR Coverage Map

| FR | Epic | Summary |
|---|---|---|
| FR1 | 1 | Claude stats-cache.json reader |
| FR2 | 2 | OpenCode SQLite reader |
| FR3 | 2 | OpenCode aggregation by provider/month |
| FR4 | 1 | Claude % of subscription limit |
| FR5 | 2 | OpenCode absolute USD display |
| FR6 | 2 | OpenCode % with manual limit |
| FR7 | 1 | Over-limit display (105%) without capping |
| FR8 | 1 | Stale vs inaccessible distinct states |
| FR9 | 1 | Last-refreshed timestamp |
| FR10 | 1 | HTTP /status endpoint |
| FR11 | 1 | SSE /events endpoint with hub fan-out |
| FR12 | 1 | /health endpoint (uptime, connections, port) |
| FR13 | 1 | Auto-spawn on port refusal |
| FR14 | 1 | Port conflict detection |
| FR15 | 1 | `pakai daemon start/stop/status` |
| FR16 | 1 | Graceful shutdown on SIGTERM/SIGINT |
| FR17 | 1 | Initial snapshot on SSE connect |
| FR18 | 1 | `pakai tmux` plain-text output |
| FR19 | 1 | `pakai waybar` JSON output |
| FR20 | 1 | `pakai status` human-readable output |
| FR21 | 3 | Custom per-provider display labels |
| FR22 | 3 | Configurable separator character |
| FR23 | 1+4 | CSS classes: ok/error/stale/over-limit (Epic 1); warning/critical (Epic 4) |
| FR24 | 1 | `pakai status --json` machine-readable mode |
| FR25 | 3 | Configurable provider display order |
| FR26 | 3 | Live preview after setup completes |
| FR27 | 3 | Provider auto-detection by file path |
| FR28 | 3 | systemd user service unit generation |
| FR29 | 3 | Copy-pasteable surface config snippet |
| FR30 | 3 | Idempotent re-run of setup |
| FR31 | 3 | `pakai config get/set/list` |
| FR32 | 3 | Config live reload (one poll cycle, no restart) |
| FR33 | 3 | Per-provider enable/disable |
| FR34 | 3 | Default config when no file present |
| FR35 | 3 | Config validation on write |
| FR36 | 4 | Threshold zone configuration |
| FR37 | 4 | Adaptive poll interval |
| FR38 | 4 | Threshold zone reflected in surface output |
| FR39 | 4 | Uniform threshold config across all providers |
| FR40 | 5 | Inject synthetic usage data (`/mock/:name` PUT) |
| FR41 | 5 | Remove injected data (`/mock/:name` DELETE) |
| FR42 | 1 | Third-party /status access (delivered with endpoint) |
| FR43 | 1 | Third-party SSE access (delivered with endpoint) |
| FR44 | 3 | Shell completions (bash, zsh, fish) |
| FR45 | 1 | Exit code 0/non-zero contract (established from start) |
| FR46 | 1 | Per-provider independent health state |
| FR47 | 1 | Unavailability indicator (`??`) |
| FR48 | 1 | Partial operation with subset of providers |
| FR49 | 5 | `pakai provider debug` raw adapter output |
| FR50 | 1 | Auto-restore when source becomes accessible |

## Epic List

### Epic 1: Core Daemon & Claude Usage Visibility

Users can run `pakai tmux`, `pakai waybar`, and `pakai status` to see real Claude usage in their terminal status bars. The daemon starts automatically on first use. When Claude data is unavailable or stale, surfaces show an explicit indicator rather than silence. The HTTP API is accessible to third-party tools.

**FRs covered:** FR1, FR4, FR7, FR8, FR9, FR10, FR11, FR12, FR13, FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR23 (partial: `ok`, `error`, `stale`, `over-limit` classes), FR24, FR42, FR43, FR45, FR46, FR47, FR48, FR50

---

### Epic 2: OpenCode Go Provider

Users can see both Claude and OpenCode Go usage side by side in all surfaces. OpenCode usage displays as absolute USD cost when no limit is configured, or as a percentage when a manual limit is set.

**FRs covered:** FR2, FR3, FR5, FR6

---

### Epic 3: Setup, Configuration & CLI Polish

Users can go from zero to a working tmux or waybar integration in under 30 seconds using `pakai setup`. They can customize display labels, separators, and provider order. Config changes take effect within one poll cycle without restarting the daemon. Shell completions are available for bash, zsh, and fish.

**FRs covered:** FR21, FR22, FR25, FR26, FR27, FR28, FR29, FR30, FR31, FR32, FR33, FR34, FR35, FR44

---

### Epic 4: Adaptive Refresh & Threshold Alerting

The daemon automatically polls more frequently as usage approaches limits. Waybar output gains `warning` and `critical` CSS classes based on user-configured thresholds. Users can configure custom alert zones.

**FRs covered:** FR23 (completes: `warning`, `critical` classes), FR36, FR37, FR38, FR39

---

### Epic 5: Developer & Integration Tooling

Developers can inject synthetic usage data to test arbitrary states without live providers. Users can inspect raw adapter data and error details to diagnose failures.

**FRs covered:** FR40, FR41, FR49

---

## Epic 1: Core Daemon & Claude Usage Visibility

Users can run `pakai tmux`, `pakai waybar`, and `pakai status` to see real Claude usage in their terminal status bars. The daemon starts automatically on first use. When Claude data is unavailable or stale, surfaces show an explicit indicator rather than silence.

### Story 1.1: Installable Binary & CI/CD

As a developer,
I want to install pakai with `go install github.com/dhanifudin/pakai/cmd/pakai@latest` and run `pakai version`,
So that I can verify the tool is installed and working before any providers are configured.

**Acceptance Criteria:**

**Given** the Go toolchain is installed with no C compiler,
**When** I run `go install github.com/dhanifudin/pakai/cmd/pakai@latest`,
**Then** the binary builds successfully with `CGO_ENABLED=0` and is placed in `$GOPATH/bin/pakai`

**Given** pakai is installed,
**When** I run `pakai version`,
**Then** it prints the version string and exits with code 0

**Given** pakai is installed,
**When** I run `pakai --help`,
**Then** it lists available subcommands with descriptions and exits with code 0

**Given** any pakai subcommand encounters a fatal error,
**When** the command exits,
**Then** it exits with a non-zero status code and prints the error to stderr

**Given** a PR is opened or a push lands on main,
**When** GitHub Actions runs the CI workflow,
**Then** `go test ./...`, `go vet ./...`, and `staticcheck` all pass on Linux/amd64

**Given** a tag matching `v*` is pushed,
**When** goreleaser runs the release workflow,
**Then** Linux/amd64 and Linux/arm64 binaries are produced and attached to a GitHub release with `CGO_ENABLED=0`

---

### Story 1.2: Daemon Lifecycle & Health Endpoint

As a user,
I want to start, stop, and check the status of the pakai daemon,
So that I can manage the background service that powers my status bar integrations.

**Acceptance Criteria:**

**Given** no daemon is running,
**When** I run `pakai daemon start`,
**Then** the daemon starts, binds exclusively to `127.0.0.1:7731`, writes its PID to `$XDG_RUNTIME_DIR/pakai/daemon.pid`, and the foreground process exits with code 0

**Given** the daemon is running,
**When** I run `pakai daemon status`,
**Then** it prints the daemon's PID, uptime, and listening port and exits with code 0

**Given** the daemon is running,
**When** I run `pakai daemon stop`,
**Then** the daemon receives SIGTERM, drains in-flight requests within 5 seconds, removes the PID file (via `defer`, not signal handler), and exits cleanly

**Given** the daemon receives SIGTERM directly (e.g. `kill <pid>`),
**When** shutdown completes,
**Then** the PID file is removed and the process exits — no stale PID file remains

**Given** the daemon is running,
**When** `GET /health` is called,
**Then** it returns HTTP 200 with JSON containing `status`, `uptime_seconds`, `connections`, `port`, and `providers` within 50ms

**Given** port 7731 is already in use by another process,
**When** I run `pakai daemon start`,
**Then** it prints an explicit message ("port 7731 is already in use by another process") and exits with a non-zero code

**Given** the daemon is running,
**When** any external interface (not 127.0.0.1) attempts a connection,
**Then** the connection is refused — the server never binds to 0.0.0.0 or any non-loopback interface

**Given** the daemon crashes unexpectedly and systemd restarts it,
**When** the new daemon starts,
**Then** it starts cleanly — it does not fail due to a stale PID file from the previous run

---

### Story 1.3: Claude Adapter & /status Endpoint

As a user,
I want the daemon to read my Claude usage from `~/.claude/stats-cache.json` and expose it at `/status`,
So that my real Claude subscription usage is available to status bar integrations.

**Acceptance Criteria:**

**Given** `~/.claude/stats-cache.json` exists and is readable,
**When** the daemon's poll loop fires (default 30s interval),
**Then** `GET /status` returns a JSON array with a Claude `Usage` object containing `state: "ok"`, a non-nil `percent` value, and a `refreshedAt` RFC3339 timestamp

**Given** usage exceeds the subscription limit (e.g. 105%),
**When** `GET /status` is called,
**Then** the Claude usage object has `percent: 105.0` — not capped at 100 — and `state: "ok"`

**Given** `~/.claude/stats-cache.json` does not exist or cannot be read,
**When** the poll loop fires,
**Then** `GET /status` returns the Claude usage object with `state: "error"` and a non-empty `errorMsg` describing the failure — the daemon continues running

**Given** the daemon last successfully read Claude data more than (poll\_interval + 10s) ago,
**When** `GET /status` is called,
**Then** the Claude usage object has `state: "stale"` and `errorMsg` includes the last successful refresh time — distinct from `state: "error"` and must not be collapsed

**Given** a panic occurs inside the Claude adapter during polling,
**When** the poll loop's `recover()` catches it,
**Then** the Claude usage object is set to `state: "error"`, the panic is logged via `slog` with full context, and the daemon continues running

**Given** any provider data is read,
**When** the adapter accesses the file,
**Then** it opens the file read-only — no writes to `~/.claude/` under any circumstances

**Given** `GET /status` is called during a background poll,
**When** the response is returned,
**Then** it completes within 50ms — it reads from the in-memory cache, never from the filesystem on demand

**Given** two providers eventually exist (claude + opencode),
**When** the claude adapter returns an error,
**Then** the opencode entry in `/status` is unaffected — provider states are fully independent

---

### Story 1.4: Surface Outputs & Auto-Spawn

As a terminal user,
I want to run `pakai tmux`, `pakai waybar`, and `pakai status` to get formatted Claude usage for my status bar,
So that I can embed live usage data into my terminal environment without manually managing the daemon.

**Acceptance Criteria:**

**Given** the daemon is running and Claude data has `state: "ok"`,
**When** I run `pakai tmux`,
**Then** it prints a compact plain-text string (e.g. `claude:72%`) and exits with code 0 within 200ms

**Given** the daemon is running and Claude data has `state: "ok"`,
**When** I run `pakai waybar`,
**Then** it prints a JSON object with `text`, `tooltip`, and `class` fields; `class` is `"ok"` and exits with code 0 within 200ms

**Given** the daemon is running and Claude data has `state: "ok"`,
**When** I run `pakai status`,
**Then** it prints a human-readable multi-line usage summary with provider name, usage value, and state, and exits with code 0 within 200ms

**Given** the daemon is running,
**When** I run `pakai status --json`,
**Then** it prints the raw `[]Usage` JSON array identical to `GET /status` and exits with code 0

**Given** the daemon is NOT running,
**When** I run `pakai tmux`,
**Then** the CLI dials localhost:7731, gets connection refused, spawns `pakai daemon start` as a detached background process, polls `GET /health` every 200ms up to 3 seconds, then prints the usage string — total elapsed time under 3 seconds

**Given** the daemon fails to become healthy within 3 seconds after auto-spawn,
**When** the timeout expires,
**Then** `pakai tmux` exits with a non-zero code and an explicit error message

**Given** Claude data has `state: "error"`,
**When** I run `pakai tmux`,
**Then** it prints `claude:??` — not silence, not zero, not an empty string

**Given** Claude data has `state: "stale"`,
**When** I run `pakai waybar`,
**Then** the JSON `class` field is `"stale"` and `tooltip` includes the last successful refresh time

**Given** Claude usage exceeds the limit (e.g. 105%),
**When** I run `pakai waybar`,
**Then** the JSON `class` field is `"over-limit"` and `text` shows the raw percentage (e.g. `claude:105%`)

**Given** the daemon was just auto-spawned and has not yet completed its first poll cycle,
**When** `GET /status` returns an empty array,
**Then** `pakai tmux` prints an empty string and exits with code 0 — empty cache is a valid transient state, not an error

**Given** a third-party tool sends `GET http://127.0.0.1:7731/status`,
**When** the request is received,
**Then** it returns HTTP 200 with the `[]Usage` JSON array — no special headers or auth required

---

### Story 1.5: SSE Live Updates

As a user with a live waybar or external tool integration,
I want the daemon to push usage updates via server-sent events,
So that my status bar reflects changes immediately without polling on a fixed interval.

**Acceptance Criteria:**

**Given** a client connects to `GET /events`,
**When** the connection is established,
**Then** the daemon immediately sends an `event: update` SSE event with the current `[]Usage` JSON as `data` — no waiting for the next poll cycle

**Given** a client is subscribed to `GET /events` and the poll loop produces new data,
**When** the cache is updated,
**Then** the daemon sends an `event: update` SSE event with the updated `[]Usage` JSON within one poll interval

**Given** a client is subscribed and no data changes for 30 seconds,
**When** the heartbeat timer fires,
**Then** the daemon sends `event: heartbeat\ndata: {}\n\n` to keep the connection alive

**Given** a subscribed client disconnects,
**When** the HTTP request context is cancelled,
**Then** the SSE goroutine exits cleanly via `ctx.Done()` selection; the hub's `sync.WaitGroup` count decrements; no goroutine leak

**Given** 10 clients are already subscribed,
**When** an 11th client connects to `GET /events`,
**Then** the daemon returns HTTP 503 and does not create a goroutine for the rejected client

**Given** a third-party tool subscribes to `GET /events`,
**When** usage data changes,
**Then** it receives the same SSE stream as pakai's own clients — no special headers or auth required

**Given** the daemon receives SIGTERM while SSE clients are connected,
**When** shutdown begins,
**Then** the daemon waits up to 5 seconds for all SSE goroutines to exit (via `WaitGroup.Wait()`) before completing shutdown

---

## Epic 2: OpenCode Go Provider

Users can see both Claude and OpenCode Go usage side by side in all surfaces. OpenCode usage displays as absolute USD cost when no limit is configured, or as a percentage when a manual limit is set.

### Story 2.1: OpenCode Adapter & Aggregation

As a user,
I want the daemon to read my OpenCode Go usage from `~/.local/share/opencode/opencode.db` and include it in `/status`,
So that my OpenCode spending is tracked alongside Claude in the same data stream.

**Acceptance Criteria:**

**Given** `~/.local/share/opencode/opencode.db` exists and is readable,
**When** the poll loop fires,
**Then** `GET /status` returns a Usage object for OpenCode with `state: "ok"`, a non-nil `cost` field (USD total for the current calendar billing month), and a `refreshedAt` RFC3339 timestamp

**Given** OpenCode has messages from multiple AI providers in the current month,
**When** the adapter queries the database,
**Then** cost is aggregated across all providers — the `cost` field reflects the total USD spent via OpenCode for the billing month

**Given** the database is queried,
**When** the adapter opens the connection,
**Then** it uses DSN `?_mode=ro&_query_only=true&_busy_timeout=5000` — read-only, no writes, 5s busy timeout

**Given** another process holds a write lock on the SQLite database,
**When** the adapter's busy timeout (5000ms) expires without acquiring the lock,
**Then** the OpenCode usage object is set to `state: "error"` with an `errorMsg` describing the SQLITE_BUSY condition — the daemon does not crash or block

**Given** `~/.local/share/opencode/opencode.db` does not exist or is inaccessible,
**When** the poll loop fires,
**Then** the OpenCode usage object has `state: "error"` and a non-empty `errorMsg` — the Claude entry in `/status` is unaffected

**Given** a panic occurs inside the OpenCode adapter during polling,
**When** the poll loop's `recover()` catches it,
**Then** the OpenCode usage object is set to `state: "error"`, the panic is logged via `slog` with provider, field, and raw value context, and the daemon continues running

**Given** the OpenCode adapter is implemented,
**When** unit tests run,
**Then** tests use `testutil.SeedDB(*sql.DB)` with a real in-memory SQLite database — no mocks of the database layer

---

### Story 2.2: OpenCode Surface Display

As a terminal user,
I want `pakai tmux`, `pakai waybar`, and `pakai status` to show OpenCode usage alongside Claude,
So that I can see my total AI spending across providers at a glance.

**Acceptance Criteria:**

**Given** no manual limit is configured for OpenCode (`provider.opencode.limit` is unset),
**When** I run `pakai tmux`,
**Then** it prints both providers (e.g. `claude:72% | opencode:$1.24`) — OpenCode shows absolute USD cost, not a percentage

**Given** a manual limit is configured (e.g. `provider.opencode.limit = 10.00`),
**When** I run `pakai tmux`,
**Then** OpenCode displays as a percentage of the configured limit (e.g. `opencode:12%`)

**Given** OpenCode usage exceeds the configured limit (e.g. $11.20 against a $10.00 limit),
**When** I run `pakai tmux`,
**Then** it shows `opencode:112%` — not capped at 100%

**Given** OpenCode has `state: "error"`,
**When** I run `pakai tmux`,
**Then** it prints `opencode:??` — Claude's entry is unaffected and shown normally

**Given** both providers have `state: "ok"`,
**When** I run `pakai waybar`,
**Then** the JSON `text` field includes both providers and `class` reflects the worst state across all providers (e.g. if one is `ok` and one is `over-limit`, `class` is `"over-limit"`)

**Given** both providers are present,
**When** I run `pakai status`,
**Then** the human-readable summary shows a row for each provider with name, current usage value, and state

**Given** OpenCode has `state: "ok"` with no configured limit,
**When** I run `pakai waybar`,
**Then** the `class` field is `"ok"` and `tooltip` shows the absolute cost (e.g. `OpenCode: $1.24 this month`)

---

## Epic 3: Setup, Configuration & CLI Polish

Users can go from zero to a working tmux or waybar integration in under 30 seconds using `pakai setup`. They can customize display labels, separators, and provider order. Config changes take effect within one poll cycle without restarting the daemon. Shell completions are available for bash, zsh, and fish.

### Story 3.1: Config File & Live Reload

As a user,
I want pakai to read configuration from `$XDG_CONFIG_HOME/pakai/config.toml` and reflect changes without restarting the daemon,
So that I can tune behavior without disrupting a running status bar integration.

**Acceptance Criteria:**

**Given** no config file exists at `$XDG_CONFIG_HOME/pakai/config.toml`,
**When** the daemon starts or any CLI subcommand runs,
**Then** it operates with built-in defaults (30s poll interval, port 7731, default labels) — no error, no missing-file warning

**Given** a valid config file exists,
**When** `config.Config()` is called from any package,
**Then** it returns the current on-disk values — it never returns a snapshot cached at startup

**Given** the config file is updated on disk (e.g. via `pakai config set`),
**When** the daemon's next poll cycle fires,
**Then** the new config values are used for that cycle — changes propagate within one poll interval without daemon restart

**Given** fsnotify detects a write to `config.toml`,
**When** the watcher fires,
**Then** the internal config state is invalidated so the next `config.Config()` call reads fresh values

**Given** fsnotify is unavailable or fails to watch,
**When** 5 seconds elapse,
**Then** the config is re-read via the fallback poll — live reload still works, just with up to 5s additional latency

**Given** the config file contains a syntax error (invalid TOML),
**When** `config.Config()` is called,
**Then** it returns the last known-good config and logs the parse error via `slog` — the daemon continues running with stale-but-valid config rather than crashing

---

### Story 3.2: Provider Auto-Detection & systemd Unit

As a new user,
I want to run `pakai setup` and have it detect my installed providers, create a systemd user service, and print integration snippets,
So that I can reach a working status bar integration in under 30 seconds without manually editing config files.

**Acceptance Criteria:**

**Given** `~/.claude/stats-cache.json` exists,
**When** I run `pakai setup`,
**Then** it detects the Claude provider, prints a confirmation ("Detected: Claude at ~/.claude/stats-cache.json"), and adds Claude to `config.toml` if not already present

**Given** `~/.local/share/opencode/opencode.db` exists,
**When** I run `pakai setup`,
**Then** it detects the OpenCode provider, prints a confirmation, and adds OpenCode to `config.toml` if not already present

**Given** neither provider file is found at the known paths,
**When** I run `pakai setup`,
**Then** it prints a clear message explaining which paths were checked and exits with a non-zero code

**Given** `pakai setup` has detected at least one provider,
**When** setup completes,
**Then** it writes a systemd user unit file to `~/.config/systemd/user/pakai.service` with `Restart=on-failure` and `RestartSec=5`, and prints instructions to enable it

**Given** the systemd unit file already exists,
**When** I run `pakai setup` again,
**Then** it overwrites the unit file with the current template and does not error

**Given** `pakai setup` has run successfully,
**When** setup completes,
**Then** it prints a copy-pasteable tmux config snippet and a waybar module JSON block — one for each surface

**Given** a provider is already configured in `config.toml`,
**When** I run `pakai setup` again,
**Then** the existing provider config entries are preserved unchanged — no duplicate entries, no value resets, running daemon is not disrupted

---

### Story 3.3: Config CLI & Provider Management

As a user,
I want to read, write, and list config values from the CLI and enable or disable individual providers,
So that I can tune pakai's behavior without manually editing the TOML file.

**Acceptance Criteria:**

**Given** a config key exists (e.g. `daemon.port`),
**When** I run `pakai config get daemon.port`,
**Then** it prints the current value and exits with code 0

**Given** a config key does not exist,
**When** I run `pakai config get nonexistent.key`,
**Then** it prints an error message and exits with a non-zero code

**Given** a valid key-value pair,
**When** I run `pakai config set daemon.poll_interval 60`,
**Then** it writes the value to `config.toml`, the daemon picks it up within one poll cycle, and the command exits with code 0

**Given** an invalid value is provided (wrong type or out of range),
**When** I run `pakai config set daemon.poll_interval abc`,
**Then** it prints a validation error ("expected integer, got 'abc'") and exits with a non-zero code — `config.toml` is not modified

**Given** any config write is requested,
**When** the value passes validation,
**Then** the write is atomic — a partially written TOML file is never left on disk

**When** I run `pakai config list`,
**Then** it prints all current config key-value pairs in `key = value` format and exits with code 0

**Given** a configured provider (e.g. opencode),
**When** I run `pakai config set provider.opencode.enabled false`,
**Then** the daemon stops polling OpenCode on the next cycle and omits it from `/status` responses

**Given** a disabled provider,
**When** I run `pakai config set provider.opencode.enabled true`,
**Then** the daemon resumes polling OpenCode on the next cycle without restart

---

### Story 3.4: Surface Customization

As a user,
I want to configure custom display labels, the separator character, and provider display order,
So that pakai's output fits my existing status bar layout without breaking the data contract.

**Acceptance Criteria:**

**Given** `provider.claude.label = "AI"` is set in config,
**When** I run `pakai tmux`,
**Then** the output shows `AI:72%` instead of `claude:72%`

**Given** `provider.opencode.label = "OC"` is set in config,
**When** I run `pakai waybar`,
**Then** the `text` field uses `OC` as the OpenCode label

**Given** `display.separator = " · "` is set in config,
**When** I run `pakai tmux` with two providers,
**Then** the output uses ` · ` between provider entries (e.g. `claude:72% · opencode:$1.24`)

**Given** `display.order = ["opencode", "claude"]` is set in config,
**When** I run `pakai tmux`,
**Then** OpenCode appears before Claude in the output string

**Given** no `display.order` is configured,
**When** I run `pakai tmux`,
**Then** providers appear in a deterministic default order (alphabetical by provider ID)

**Given** a label or separator config change is written via `pakai config set`,
**When** the daemon's next poll cycle fires and I run `pakai tmux`,
**Then** the new label or separator is used — no daemon restart required

---

### Story 3.5: Shell Completions & Setup Live Preview

As a user,
I want shell tab-completion for pakai subcommands and to see a live preview of surface output after setup,
So that I can use the CLI efficiently and verify my integration is working immediately after setup.

**Acceptance Criteria:**

**Given** I run `pakai completion bash`,
**Then** it prints a bash completion script to stdout and exits with code 0

**Given** I run `pakai completion zsh`,
**Then** it prints a zsh completion script to stdout and exits with code 0

**Given** I run `pakai completion fish`,
**Then** it prints a fish completion script to stdout and exits with code 0

**Given** a completion script is sourced in the shell,
**When** I type `pakai ` and press Tab,
**Then** available subcommands are listed as completions

**Given** `pakai setup` has successfully detected and configured at least one provider,
**When** setup completes its final step,
**Then** it starts the daemon (if not already running), fetches live data, and prints what `pakai tmux` and `pakai waybar` would output with the current data

**Given** the live preview fetch fails (e.g. daemon fails to start),
**When** setup prints the preview section,
**Then** it prints an explicit error explaining why the preview could not be shown — setup exits with code 0 since provider detection and config writing succeeded

---

## Epic 4: Adaptive Refresh & Threshold Alerting

The daemon automatically polls more frequently as usage approaches limits. Waybar output gains `warning` and `critical` CSS classes based on user-configured thresholds. Users can configure custom alert zones.

### Story 4.1: Threshold Configuration

As a user,
I want to configure percentage thresholds that define warning, critical, and over-limit alert zones,
So that pakai's alerting is calibrated to my personal risk tolerance for each subscription.

**Acceptance Criteria:**

**Given** no threshold config exists,
**When** the daemon evaluates provider usage,
**Then** it uses built-in defaults: `warning = 50`, `critical = 80`, `over_limit = 100` — applied uniformly across all enabled providers

**Given** I run `pakai config set thresholds.warning 60`,
**When** the config is written,
**Then** it is accepted and stored — the value 60 represents "60% usage triggers warning zone"

**Given** I run `pakai config set thresholds.critical 90`,
**When** the config is written,
**Then** it is accepted, and the next poll cycle uses 90 as the critical threshold

**Given** an out-of-range value is provided (e.g. `pakai config set thresholds.warning 150`),
**When** config validation runs,
**Then** it rejects the value with an explicit error ("thresholds.warning must be between 0 and 100") and does not modify `config.toml`

**Given** `thresholds.warning` is set to a value greater than or equal to `thresholds.critical`,
**When** config validation runs,
**Then** it rejects the write with an explicit error ("thresholds.warning must be less than thresholds.critical") and does not modify `config.toml`

**Given** threshold config is updated via `pakai config set`,
**When** the daemon's next poll cycle fires,
**Then** the new thresholds are applied — no daemon restart required

**Given** threshold config is set,
**When** a provider's `percent` value is nil (cost-only provider with no limit),
**Then** the threshold comparison is skipped for that provider — it remains in `ok` state regardless of thresholds

---

### Story 4.2: Adaptive Poll Interval

As a user approaching a subscription limit,
I want the daemon to poll more frequently as my usage rises,
So that my status bar stays current when it matters most — near the limit — without burning unnecessary reads when I'm well under.

**Acceptance Criteria:**

**Given** all providers have usage below the `warning` threshold (e.g. < 50%),
**When** the poll loop evaluates the current interval,
**Then** it uses the slow interval (300s default) — the ticker is reset to 300s for the next cycle

**Given** any provider has usage between `warning` and `critical` thresholds (e.g. 50–80%),
**When** the poll loop evaluates the current interval,
**Then** it uses the medium interval (120s default) — the ticker is reset to 120s

**Given** any provider has usage between `critical` and 95%,
**When** the poll loop evaluates the current interval,
**Then** it uses the fast interval (30s default) — the ticker is reset to 30s

**Given** any provider has usage above 95%,
**When** the poll loop evaluates the current interval,
**Then** it uses the urgent interval (10s default) — the ticker is reset to 10s

**Given** the interval changes between cycles (e.g. usage crosses from warning to critical zone),
**When** the poll loop resets the ticker,
**Then** the new interval takes effect immediately on the next tick — no residual wait from the old interval

**Given** threshold config values are updated via `pakai config set`,
**When** the daemon's next poll cycle fires,
**Then** the adaptive interval boundaries use the new threshold values — interval recalculation always reads fresh config

**Given** a provider has `state: "error"` or `state: "stale"`,
**When** the poll loop evaluates the interval,
**Then** the errored provider's `percent` is excluded from the worst-case comparison — interval is driven only by providers with valid data

---

### Story 4.3: Warning & Critical CSS Classes

As a waybar user,
I want the `class` field in `pakai waybar` output to reflect `warning` and `critical` threshold zones,
So that my status bar changes color automatically as I approach my subscription limit.

**Acceptance Criteria:**

**Given** a provider's usage is below the `warning` threshold,
**When** I run `pakai waybar`,
**Then** the `class` field is `"ok"` for that provider

**Given** a provider's usage is at or above the `warning` threshold but below `critical` (e.g. 65% with warning=50, critical=80),
**When** I run `pakai waybar`,
**Then** the `class` field is `"warning"`

**Given** a provider's usage is at or above the `critical` threshold but at or below 100% (e.g. 85%),
**When** I run `pakai waybar`,
**Then** the `class` field is `"critical"`

**Given** a provider's usage exceeds 100%,
**When** I run `pakai waybar`,
**Then** the `class` field is `"over-limit"` — this takes precedence over `"critical"`

**Given** multiple providers are present with different threshold zones,
**When** I run `pakai waybar`,
**Then** the top-level `class` field reflects the worst state across all providers (precedence: `over-limit` > `critical` > `warning` > `ok` > `stale` > `error`)

**Given** a provider has `state: "error"`,
**When** I run `pakai waybar`,
**Then** the `class` field is `"error"` for that provider — threshold comparison does not apply to errored providers

**Given** a provider has `state: "stale"`,
**When** I run `pakai waybar`,
**Then** the `class` field is `"stale"` — stale data does not trigger threshold-based classes

**Given** threshold values are updated via `pakai config set` and the daemon has run one poll cycle,
**When** I run `pakai waybar`,
**Then** the `class` reflects the new threshold boundaries — no daemon restart required

**Given** alert state is `warning` or `critical`,
**When** I run `pakai tmux`,
**Then** the text still shows the numeric percentage (e.g. `claude:78%`) — state is communicated through both the CSS class (color) and the number (text), never color alone

---

## Epic 5: Developer & Integration Tooling

Developers can inject synthetic usage data to test arbitrary states without live providers. Users can inspect raw adapter data and error details to diagnose failures.

### Story 5.1: Mock Provider

As a developer,
I want to inject synthetic usage data for a provider via the daemon's HTTP API and remove it to restore live readings,
So that I can test arbitrary usage states (warning, critical, over-limit, error) without needing real subscription activity.

**Acceptance Criteria:**

**Given** the daemon is running,
**When** I send `PUT /mock/claude` with body `{"percent": 85}`,
**Then** the daemon responds HTTP 200, subsequent `GET /status` returns the Claude usage object with `percent: 85.0` and `state: "ok"` — the live claude adapter is bypassed

**Given** a mock is active for a provider,
**When** I send `PUT /mock/claude` with body `{"percent": 105}`,
**Then** the response is HTTP 200 and `/status` returns `percent: 105.0` — over-limit states are injectable

**Given** a mock is active,
**When** I send `DELETE /mock/claude`,
**Then** the daemon responds HTTP 200 and subsequent `/status` calls return live data from the real claude adapter

**Given** mock data is active,
**When** the daemon is restarted,
**Then** the mock data is gone — injected state is in-memory only and does not persist across restarts

**Given** I send `PUT /mock/opencode` with body `{"cost": 7.50}`,
**When** `/status` is called,
**Then** the OpenCode usage object shows `cost: 7.50` with `state: "ok"` — the live opencode adapter is bypassed

**Given** an invalid mock body is sent (e.g. `{"percent": "high"}`),
**When** the daemon parses the request,
**Then** it responds HTTP 400 with `{"error": "invalid mock payload"}` and does not modify any provider state

**Given** a mock is active for provider "claude",
**When** I send `DELETE /mock/nonexistent`,
**Then** the daemon responds HTTP 404 with `{"error": "no mock active for provider: nonexistent"}`

**Given** a mock is active,
**When** the SSE hub pushes the next update,
**Then** the mocked usage data appears in the SSE `event: update` payload — mock state flows through all surfaces consistently

---

### Story 5.2: Provider Debug Output

As a user troubleshooting a broken provider integration,
I want to run `pakai provider debug <name>` to see the exact file path being read, the raw data received, and any parse errors,
So that I can file a useful bug report or fix my own configuration without running a debugger.

**Acceptance Criteria:**

**Given** the Claude provider is configured and its data file is readable,
**When** I run `pakai provider debug claude`,
**Then** it prints: the resolved file path (`~/.claude/stats-cache.json` expanded to absolute), the raw file contents, the parsed Usage values, and exits with code 0

**Given** the OpenCode provider is configured and the database is readable,
**When** I run `pakai provider debug opencode`,
**Then** it prints: the resolved database path, the raw rows returned by the aggregation query, the computed `cost` value, and exits with code 0

**Given** the provider data file does not exist,
**When** I run `pakai provider debug claude`,
**Then** it prints the path that was checked, an explicit "file not found" message, and exits with a non-zero code

**Given** the provider data file exists but contains malformed data,
**When** I run `pakai provider debug claude`,
**Then** it prints the raw content that failed to parse, the specific field that caused the failure, and the parse error — sufficient context to file a bug report without a debugger

**Given** the daemon is running and exposes a `GET /debug` endpoint,
**When** `pakai provider debug claude` is invoked,
**Then** it fetches raw debug data from `GET /debug/claude` rather than reading the file directly — debug output reflects the daemon's actual view of the provider state

**Given** a provider name is provided that is not configured,
**When** I run `pakai provider debug unknown`,
**Then** it prints "unknown provider: unknown" and exits with a non-zero code

**Given** `pakai provider debug claude` runs successfully,
**When** the output is printed,
**Then** it includes all of: resolved path, raw data excerpt, parsed field values, and current `state` — no omissions that would require running again to get a complete picture
