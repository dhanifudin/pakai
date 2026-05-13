---
stepsCompleted: [step-01-init, step-02-discovery, step-02b-vision, step-02c-executive-summary, step-03-success, step-04-journeys, step-05-domain, step-06-innovation, step-07-project-type, step-08-scoping, step-09-functional, step-10-nonfunctional, step-11-polish]
releaseMode: phased
inputDocuments: ['_bmad-output/brainstorming/brainstorming-session-2026-05-10-1200.md']
workflowType: 'prd'
classification:
  projectType: 'background_service + cli_control_plane'
  domain: 'AI Productivity / Power User Tooling'
  complexity: 'Medium-High'
  projectContext: 'greenfield'
---

# Product Requirements Document - pakai

**Author:** Dian
**Date:** 2026-05-11

## Executive Summary

pakai is a background service and CLI control plane that aggregates AI subscription usage data from multiple providers (Claude, OpenCode Go) and surfaces it as ambient status indicators across terminal-native surfaces — tmux status bar, waybar (Linux) — with a post-MVP Bubble Tea TUI dashboard and Tauri floating card widget for Windows. It targets power users of AI tools who live in the terminal and currently have no visibility into their subscription consumption without leaving their workflow.

The core problem: AI subscriptions are financially opaque. Users paying $20–$200/month across multiple providers have no ambient awareness of approaching limits, spending patterns, or cross-provider usage — until they hit a wall. Existing solutions are dashboard-first, requiring deliberate navigation. pakai is workflow-first: the data is always visible where the user already looks.

### What Makes This Special

pakai differs from dashboard alternatives on three axes:

**Ambient, not intentional.** Usage data lives in the status bar and waybar — surfaces the user checks dozens of times per day without deciding to. No context switch required.

**Honest by design.** Usage exceeding limits is displayed as-is (e.g. `claude:105%`), provider errors surface as explicit degraded states rather than silence, and the data layer owns no permanent state — pakai reads from where providers write, never from its own database.

**Zero-friction integration.** `pakai setup` auto-detects installed providers, writes the systemd user unit, and prints copy-pasteable config snippets for each surface. A new user is operational in under 30 seconds. The binary is self-contained — one `go install` or pre-built download, no runtime dependencies.

## Project Classification

| Attribute | Value |
|---|---|
| Project Type | Background service + CLI control plane |
| Domain | AI Productivity / Power User Tooling |
| Complexity | Medium-High |
| Context | Greenfield |
| Stack | Go (daemon + CLI), Rust + Tauri (post-MVP widget), monorepo |
| MVP Surfaces | CLI (`pakai status`), tmux, waybar |
| Post-MVP | Bubble Tea TUI dashboard (`pakai dashboard`), Tauri floating card widget (Windows) |

## Success Criteria

### User Success

- A new user running `pakai setup` reaches a working tmux or waybar integration in under 30 seconds, with no manual config file editing required
- Status bar accurately reflects provider usage within one daemon refresh cycle of install
- When a provider is unreachable or returns stale data, the surface shows an explicit indicator (e.g. `claude:??`) — users are never silently misled
- Users can shorten provider labels and customize separators to fit their existing status bar layout without breaking the data contract
- The "aha moment": glancing at the status bar and immediately knowing which AI subscription is running hot — without opening a browser or switching context

### Business Success

Since pakai is open source developer tooling, success is measured by adoption and community health:

- Listed in AUR within 4 weeks of first stable release
- Provider adapter PRs from community contributors — the plugin registry design enables this
- Users sharing their waybar/tmux configs publicly referencing pakai (organic distribution)
- GitHub issues skew toward feature requests and new providers, not bugs — indicating reliability

### Technical Success

- Daemon runs continuously for 7+ days without crash or memory leak on a standard Linux desktop
- `pakai tmux` / `pakai waybar` subcommand response time < 200ms (reading from daemon cache, not live fetch)
- Claude adapter correctly reads from `~/.claude/stats-cache.json` and surfaces meaningful activity data
- OpenCode Go adapter correctly queries `~/.local/share/opencode/opencode.db` and returns per-provider token usage and cost
- `pakai setup` successfully writes a functioning systemd user unit on first run

### Measurable Outcomes

| Metric | Target |
|---|---|
| Setup time (new user) | < 30 seconds |
| Subcommand response time | < 200ms |
| Daemon continuous uptime | 7+ days |
| Providers at launch | 2 (Claude, OpenCode Go) |
| Surfaces at launch | 3 (CLI, tmux, waybar) |

## Product Scope

### MVP — Minimum Viable Product

- Go daemon: HTTP server (`localhost:7731`), `GET /status`, `GET /events` SSE, `GET /health`, TTL cache, auto-spawn
- Claude provider adapter: reads `~/.claude/stats-cache.json` (messageCount, sessionCount per day), local-first
- OpenCode Go provider adapter: queries `~/.local/share/opencode/opencode.db`, aggregates tokens + cost by providerID per month
- Subcommands: `pakai status`, `pakai tmux`, `pakai waybar`
- `pakai setup [surface]`: auto-detect providers, write systemd user unit, print copy-pasteable surface config
- `pakai config set/get/list`: live config changes, no restart
- `pakai provider mock/unmock`: built-in testing and demo mode
- Alert system: configurable thresholds driving both visual alerts and adaptive refresh intervals
- Provider-labeled output: `claude:82% opencode-go:$1.12`, configurable labels and separator
- Graceful degradation: per-provider `ok`/`error`/`stale` status, never silent failure
- Over-limit display: shown as-is, never capped at 100%
- Single Go binary: `go install` + pre-built GitHub releases + AUR PKGBUILD

### Growth Features (Post-MVP)

- Tauri widget: floating always-on-top card (Windows), system tray icon, daemon supervisor
- Shell prompt integration documentation (starship, oh-my-zsh, pure)
- Background update notifications (daily check, surfaces only in `pakai status` and Tauri tray)
- Additional provider adapters (community-contributed via plugin registry)
- `pakai setup waybar` printing complete waybar config + CSS block

### Vision (Future)

- Push-based provider updates: `POST /push/{provider}` endpoint for when AI tools support webhooks — eliminates polling entirely
- Browser-based dashboard consuming `localhost:7731` — zero additional pakai server work required
- Neovim plugin, Fish shell integration via the open HTTP API
- Windows-native daemon lifecycle independent of Tauri widget

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — deliver the minimum that eliminates the core pain (AI subscription opacity) and makes users say "this is genuinely useful." The criterion is simple: can Kai glance at his status bar and know which provider is running hot, without leaving his terminal? If yes, MVP ships.

**Resource Requirements:** Single experienced Go developer. 6 weeks to full MVP. Drop-dead shippable at week 3: daemon + Claude adapter + `pakai tmux`. If anything slips, the week-3 checkpoint is a real release — not a demo.

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:** First Light (Journey 1 — happy path setup), 95% Wall (Journey 2 — threshold alerts and adaptive refresh), Provider Down (Journey 3 — graceful degradation). Journey 4 (Windows Isolation) partially supported via daemon-only on Windows.

**Must-Have Capabilities:**

- Go daemon: HTTP server at `localhost:7731`, `GET /status`, `GET /events` (SSE), `GET /health`, TTL cache with singleflight deduplication, auto-spawn on first subcommand call
- Claude provider adapter: reads `~/.claude/stats-cache.json`, extracts `dailyActivity`, `messageCount`, `sessionCount` per day, local-first
- OpenCode Go provider adapter: queries `~/.local/share/opencode/opencode.db` via pure-Go SQLite driver (`modernc.org/sqlite`, read-only), aggregates token usage and cost by `providerID` for the current billing month
- Surface subcommands: `pakai status` (human-readable), `pakai tmux` (plain string per provider), `pakai waybar` (JSON with CSS class field)
- `pakai setup [surface]`: auto-detect installed providers, write systemd user unit, print copy-pasteable config snippets for requested surface
- `pakai config set/get/list`: live config changes without daemon restart
- `pakai provider mock/unmock`: built-in demo and testing mode
- Alert system: configurable thresholds (`[50, 80, 95]` default) driving both CSS class changes and adaptive refresh intervals: `< 50% → 300s`, `50–80% → 120s`, `> 80% → 30s`, `> 95% → 10s`
- Provider-labeled output: `claude:82% opencode-go:$1.12`, configurable labels and separator
- Graceful degradation: per-provider `ok` / `error` / `stale` state machine, explicit `??` indicator on failure, never silent
- Over-limit display: shown as-is (e.g. `claude:105%`), never capped
- Signal handling: SIGTERM/SIGINT via `signal.NotifyContext`, 5s drain timeout, exit 0 clean / exit 1 timeout
- Single Go binary: `go install` + pre-built GitHub releases + AUR PKGBUILD

### Post-MVP Features (Phase 2)

- `pakai dashboard`: Bubble Tea TUI with live SSE subscription, per-provider breakdown, and `[d]` debug mode showing raw provider data — deferred because it solves a different job (deliberate inspection) vs ambient awareness, and adds significant surface complexity
- Tauri widget: floating always-on-top card (Windows), system tray icon, daemon supervisor — deferred to unblock Linux MVP without Rust build chain
- Shell prompt integration documentation (starship, oh-my-zsh, pure)
- Background update notifications (daily check, surfaces only in `pakai status` and Tauri tray)
- Additional provider adapters (community-contributed via plugin registry)
- `pakai setup waybar`: print complete waybar config JSON block + matching CSS snippet
- `pakai provider add/debug`: provider registration workflow

### Risk Mitigation Strategy

**Technical Risks:**
- *CGo-free SQLite*: `modernc.org/sqlite` eliminates the CGo build requirement — pure Go, cross-compiles cleanly. Risk: performance overhead vs. maturity — acceptable for read-only queries at 30s intervals.
- *SSE goroutine leaks*: mitigated by `context.Done()` select in every SSE handler, `sync.WaitGroup` drain on shutdown, and max 10 concurrent client cap enforced at connection accept time.
- *Port conflicts*: daemon detects `localhost:7731` already bound at startup and exits with a clear error rather than silently failing.
- *File not found*: each provider adapter returns an `error`/`stale` state when the source file/DB is missing — never a panic or zero-value silently treated as 0%.

**Market Risks:**
- *Niche audience*: pakai targets power users who already run tmux/waybar. Mitigation: AUR listing and community adapter PRs are the validation signal. If nobody contributes adapters within 3 months of launch, the plugin model design needs revisiting.
- *Provider API changes*: local-file strategy means pakai doesn't break when providers change their billing APIs — only when they change their local cache format. Risk is low; mitigation is schema validation on read with explicit `stale` fallback.

**Resource Risks:**
- *Single developer*: week-3 checkpoint (daemon + Claude adapter + `pakai tmux`) is a shippable release. If capacity shrinks, ship week 3 and keep growing.
- *Scope creep*: dashboard, Tauri widget, and additional providers are explicitly Post-MVP. Any pressure to include them in Phase 1 requires explicit PRD revision.

## User Journeys

### Journey 1: First Light (Primary User — Happy Path)

**Meet Kai**, a senior backend engineer who pays for Claude Pro and OpenCode Go. Every month, around the 25th, Kai gets a vague anxiety — "am I close to my limit?" — and opens the Anthropic billing page to check. It takes 30 seconds, breaks flow, and the number he sees is two days stale anyway.

**Opening scene:** Kai finds pakai on GitHub. Single `go install`. Thirty seconds later:

```
$ pakai setup
→ Detected Claude Code at ~/.claude/ ✓
→ Detected OpenCode Go at ~/.local/share/opencode/ ✓
→ Written ~/.config/systemd/user/pakai.service ✓
→ Daemon started ✓
→ Add to waybar: see `pakai setup waybar`
```

**Rising action:** He runs `pakai setup waybar`, pastes the printed JSON block into his waybar config and CSS snippet into his stylesheet, restarts waybar. His bar shows: `claude:68% opencode-go:$0.84`. No browser tab. No context switch.

**Climax:** Three days later, nearing end of month, the waybar text turns amber: `claude:83%`. He didn't configure anything — the threshold was a sensible default. He glances, paces himself, keeps coding.

**Resolution:** Kai never opens the billing page again. He files a GitHub PR to add a Gemini adapter.

**Requirements revealed:** `pakai setup`, provider auto-detection, systemd unit writing, waybar JSON output with CSS classes, default alert thresholds, ambient color change.

---

### Journey 2: The 95% Wall (Primary User — Alert Path)

**Meet Priya**, a data scientist who burns through 80% of her Claude Pro limit in three days during crunch periods.

**Opening scene:** Thursday. Mid-analysis. Tmux status bar shows `claude:95%` in red. She set `alert.thresholds 50,80,95` last week. The adaptive refresh has tightened to every 10 seconds automatically.

**Rising action:** She runs `pakai status` — full output shows usage bar, period end date, last refresh 12 seconds ago. She switches to a cheaper OpenCode Go model for the week.

**Climax:** When usage ticks to 98%, the status bar updates within 15 seconds. She sees it immediately. She doesn't hit her limit.

**Resolution:** The cross-provider visibility — both numbers in one bar — is what made the difference. She would have missed it with a single-provider tool.

**Requirements revealed:** Adaptive refresh intervals, unified threshold config, `pakai status` detailed view, multi-provider tmux output, per-provider independent display.

---

### Journey 3: Something's Wrong (Edge Case — Provider Error Path)

**Meet Dev**, a DevOps engineer. He updated OpenCode Go overnight. Morning: waybar shows `claude:41% opencode-go:??`.

**Opening scene:** He notices `??` immediately. Runs `pakai dashboard`, hits `[d]` for debug mode. Raw view: Claude `ok`, OpenCode Go `error: database schema mismatch`.

**Climax:** Dev opens a GitHub issue with the exact error from the debug view — schema version, table name, column that changed. Everything the maintainer needs. A patch lands within a day.

**Resolution:** The system stayed honest — it showed him a problem rather than showing stale data as if it were current.

**Requirements revealed:** Per-provider error states, `??` display for error/stale, `pakai dashboard` debug mode showing raw JSON + error messages.

---

### Journey 4: The New Machine (Secondary User — Partial Environment)

**Meet Sam**, a technical PM with pakai in their dotfiles. Just reimaged laptop — only Claude Code installed, not OpenCode Go yet.

**Opening scene:** `pakai setup` on fresh machine. OpenCode Go not found — pakai detects Claude only, writes systemd unit, starts daemon. Waybar shows `claude:12%`. No crash, no error.

**Climax:** Two days later Sam installs OpenCode Go, runs `pakai setup` again. OpenCode Go detected, adapter registered, daemon restarted. Waybar: `claude:31% opencode-go:$0.21`.

**Resolution:** `pakai setup` is idempotent. Running it twice adds new providers without disrupting existing ones.

**Requirements revealed:** Partial provider detection, `pakai setup` idempotency, re-running setup adds providers without disruption, graceful single-provider operation.

---

### Journey Requirements Summary

| Capability | Revealed By |
|---|---|
| `pakai setup` auto-detection + idempotency | Journey 1, 4 |
| Systemd user unit generation | Journey 1 |
| Waybar JSON with CSS classes | Journey 1 |
| Default alert thresholds (50/80/95%) | Journey 2 |
| Adaptive refresh (tightens near limit) | Journey 2 |
| `pakai status` detailed multi-provider view | Journey 2 |
| Per-provider error/stale states with `??` | Journey 3 |
| `pakai dashboard` debug mode | Journey 3 |
| Partial provider support (1 of N missing) | Journey 4 |
| Graceful degradation without crash | Journey 3, 4 |

## Domain-Specific Requirements

The user journeys above reveal specific constraints that shape the implementation: local file access patterns, cross-distribution compatibility, polling reliability, and graceful handling of provider schema changes. These requirements are binding on any implementation.

### Compliance & Regulatory

None applicable. pakai is a local-only tool — all data remains on the user's machine. No network transmission of usage data, no cloud storage, no third-party data sharing.

### Technical Constraints

**Local file access:**
- Claude adapter depends on `~/.claude/stats-cache.json` — format controlled by Anthropic, subject to change without notice. Adapter must degrade gracefully (return stale cache or `error` signal) rather than crashing the daemon
- OpenCode Go adapter depends on `~/.local/share/opencode/opencode.db` — must open in read-only mode (`_mode=ro&_query_only=true` DSN) with explicit acceptance of eventual consistency, not point-in-time
- All paths follow XDG Base Directory spec on Linux. macOS is not a stated target — out of scope for MVP

**SQLite driver:**
- Must use a pure-Go SQLite driver (`modernc.org/sqlite`) — no CGo. Avoids C toolchain requirement for contributors and CI. Write performance penalty is irrelevant (read-only workload)

**Data freshness — polling over file watching:**
- Adapters MUST use polling with a configurable interval (default 30s), NOT `fsnotify`/inotify. Rationale: inotify silently fails on NFS and WSL1, has kernel-level watch limits, and is non-mockable. Polling with an injected `FileReader func(path string) ([]byte, error)` is fully testable without touching the filesystem

**Cache singleflight:**
- Daemon MUST use a singleflight pattern on cache refresh. When multiple clients hit the daemon simultaneously at TTL boundary, only one file read executes; all callers receive the same result. Prevents thundering herd on concurrent poll boundaries

**Port management:**
- Daemon binds to `localhost:7731` (configurable). Auto-spawn logic MUST probe the port before attempting to bind — if occupied, fail with a clear error message. Silent port conflict failure is not acceptable

**SSE connection management:**
- SSE handlers MUST select on `{r.Context().Done(), dataCh}` — no blocking sends. Active SSE goroutines tracked via `sync.WaitGroup`. Maximum concurrent SSE clients: 10 (configurable). Daemon must not silently degrade under multi-surface setups (tmux + waybar + TUI simultaneously)

**Signal handling:**
- Daemon MUST handle SIGTERM and SIGINT via `signal.NotifyContext`
- On signal: stop accepting new SSE connections → drain active goroutines via `WaitGroup.Wait()` with 5s hard timeout → remove PID file (in `defer`, not signal handler) → exit 0 on clean shutdown, exit 1 on timeout-forced exit

**Config file:**
- Location: `$XDG_CONFIG_HOME/pakai/config.toml`
- Format: TOML (`BurntSushi/toml`)
- Required keys: daemon port (default 7731), poll interval (default 30s), per-provider enable/disable flags, `version = 1`
- Behavior on missing file: use defaults silently — do NOT error on startup

**Provider interface contract:**
- All provider adapters MUST implement `Provider` interface: `Fetch(ctx context.Context) (Usage, error)`
- Adapters MUST accept data sources as injected dependencies (`io.Reader` or `*sql.DB`) — path resolution at call site, not inside the adapter
- Unit tests must be runnable without real files or a real SQLite database on disk

### Risk Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Provider changes local data format | Medium | Graceful degradation to `error` state; `pakai provider debug` exposes raw parse errors |
| Daemon fails to start / port conflict | Medium | Port probe before bind; clear error output; subcommands verify daemon health before returning |
| OpenCode Go schema drift | Medium | Adapter pins to known schema version; `??` display on parse failure rather than crash |
| SSE goroutine leak on client disconnect | Medium | `Context.Done()` select in all SSE handlers; `WaitGroup` tracking; enforced connection cap |
| Cache thundering herd at TTL boundary | Low-Medium | Singleflight pattern on adapter refresh layer |
| systemd not available (NixOS, Alpine) | Low | Auto-spawn as fallback; known non-target environment |
| SQLite CGo complications | Low | Pure-Go driver mandated from day one |

## Innovation & Novel Patterns

> This section documents novel design patterns in pakai that differ meaningfully from existing solutions. Implementers and architects should read this before making tradeoffs in the functional and non-functional requirements — these patterns explain *why* certain constraints exist.

### Detected Innovation Areas

**Ambient awareness vs. intentional dashboards**
Every existing AI usage tool is pull-based — users must decide to check. pakai is push-to-ambient: usage data lives in tmux and waybar, surfaces the user checks dozens of times per day as part of existing workflow. This is not a UX improvement to an existing pattern — it's a different interaction model. The information finds the user, not the other way around.

**Accidental extension platform**
The local HTTP daemon at `localhost:7731` was designed for pakai's own surfaces. Because the protocol is plain HTTP with JSON and SSE, any tool — a Neovim plugin, a Fish shell prompt, a custom script — can consume it with zero additional pakai work. pakai becomes a local data bus for AI usage information without intending to. Extensibility as a side effect of simplicity.

**Honest-by-default data rendering**
Existing tools smooth over bad states — cap at 100%, hide stale data, show last-known values without staleness markers. pakai's "Honest by Design" principle inverts this: show `claude:105%` when usage exceeds limits, show `claude:??` when the provider is unreachable, show staleness timestamps prominently. A conscious break from dashboard conventions and a genuine differentiator for technically sophisticated users.

### Market Context & Competitive Landscape

| Solution | Model | Gap |
|---|---|---|
| Anthropic billing page | Web dashboard, pull-based | Requires browser, breaks flow, data 24h stale |
| robinebers/openusage | CLI query, single provider | No ambient surface, no daemon, single provider |
| Provider CLIs | Per-provider, no aggregation | No unified view, no status bar integration |
| pakai | Ambient daemon, multi-provider, open API | — |

### Validation Approach

- **Ambient value:** Users look at the status bar before opening any browser tab to check usage
- **Extension platform:** Community-built consumers of `localhost:7731` — third-party integrations without pakai building them
- **Honest rendering:** User retention despite (or because of) showing uncomfortable data like `105%`

### Risk Mitigation

| Innovation Risk | Mitigation |
|---|---|
| Status bar blindness (ambient ignored) | Alert thresholds change color/icon to break habituation at critical levels |
| Open API attracts malicious local consumers | Localhost is user-scoped; revisit auth if remote access use cases emerge |
| Honest displays frustrate less technical users | Post-MVP Tauri widget can soften display for non-terminal audiences |

## CLI Tool + Background Service Specific Requirements

### Project-Type Overview

pakai is a scriptable-first CLI tool backed by a long-running background service. The CLI is the control plane; the daemon is the product. Users interact with pakai in three modes:

1. **Scriptable/embedded** — `pakai tmux` and `pakai waybar` called by external processes on a poll interval, expected to return instantly
2. **Interactive** _(post-MVP)_ — `pakai dashboard` opens a full-terminal TUI (Bubble Tea) subscribing to the daemon's SSE stream
3. **Administrative** — `pakai setup`, `pakai config`, `pakai provider` — one-shot commands that configure and manage the daemon

### Command Structure

```
pakai
├── status                    # human-readable multi-provider usage summary
├── tmux                      # compact string for tmux status-right
├── waybar                    # JSON in waybar custom module format
├── dashboard                 # [post-MVP] interactive Bubble Tea TUI with SSE
├── setup [surface]           # first-run and surface-specific setup
│   ├── (no arg)              # full setup: detect + daemon + systemd + snippets
│   ├── waybar                # print waybar config + CSS block
│   └── tmux                  # print tmux config line
├── config
│   ├── set <key> <value>     # live config update, no restart
│   ├── get <key>             # read single config value
│   └── list                  # show all config key-value pairs
├── provider
│   ├── add <name> [flags]    # register a provider
│   ├── debug <name>          # show scan paths, parse result, error state
│   ├── mock <name> [flags]   # inject synthetic usage data
│   └── unmock [name]         # remove mock data (all if no name)
└── daemon
    ├── start                 # explicit daemon start (also auto-spawned)
    ├── stop                  # graceful shutdown
    └── status                # daemon health, uptime, port, active connections
```

### Output Formats

| Subcommand | Format | Consumer |
|---|---|---|
| `pakai tmux` | Plain string: `claude:82% opencode-go:$1.12` | tmux `status-right` |
| `pakai waybar` | JSON: `{"text":"...","tooltip":"...","class":"warning","percentage":82}` | waybar custom module |
| `pakai status` | Human-readable multi-line with progress bars | Terminal, human eyes |
| `pakai dashboard` _(post-MVP)_ | Interactive TUI (Bubble Tea), SSE-driven live updates | Terminal, interactive |
| `GET /status` | JSON array of provider Usage objects | HTTP clients, Tauri widget |
| `GET /events` | SSE stream of Usage update events | SSE consumers |

All compact outputs respect configurable provider labels and separator. Waybar output includes CSS class (`ok`, `warning`, `critical`, `over-limit`, `error`, `stale`) matching `alert.thresholds`.

### Config Schema

Location: `$XDG_CONFIG_HOME/pakai/config.toml`

```toml
version = 1

[daemon]
port = 7731
poll_interval = "30s"
max_sse_clients = 10

[alert]
thresholds = [50, 80, 95]  # drives both color changes and adaptive refresh intervals

[format]
separator = " "            # between providers in compact surfaces

[provider.claude]
label = "claude"
enabled = true

[provider.opencode]
label = "opencode-go"
enabled = true
# limit = 10.00           # optional manual USD limit
```

Config changes via `pakai config set <dotted.key> <value>` take effect immediately — no restart required.

### Scripting Support

- All subcommands exit 0 on success, non-zero on error — safe in shell conditionals
- `pakai tmux` and `pakai waybar` must return in < 200ms — designed for repeated poll invocation
- `--format json` flag on `pakai status` outputs machine-readable JSON
- `pakai provider mock` enables CI and demo usage without real credentials or local data files
- Auto-spawn: any subcommand transparently starts the daemon if not running

### Shell Completion

- `pakai completion bash|zsh|fish` via cobra built-in
- `pakai setup` prints completion installation instructions as part of onboarding output

### Implementation Considerations

- CLI framework: `cobra` — standard Go CLI, completion generation, subcommand grouping
- Config management: `viper` — TOML support, live reload, dotted key access
- Daemon protocol: HTTP (not Unix socket) — consistent for CLI, TUI, and Tauri widget
- Subcommands verify daemon health via `GET /health` before executing — explicit error if unreachable after auto-spawn attempt

## Functional Requirements

### Provider Data Reading

- **FR1:** The system can read Claude usage data from a local stats cache file without making network requests
- **FR2:** The system can query OpenCode Go usage from a local SQLite database without making network requests
- **FR3:** The system can aggregate OpenCode Go usage by provider and calendar billing month from per-message records
- **FR4:** The system can express Claude usage as a percentage of a subscription limit
- **FR5:** The system can express OpenCode Go usage as an absolute USD cost when no limit is configured
- **FR6:** The system can express OpenCode Go usage as a percentage when a manual limit is configured
- **FR7:** The system can display usage exceeding a limit as-is (e.g. `105%`) without capping at 100%
- **FR8:** The system can distinguish between a provider data source being inaccessible versus returning data that has not changed within a configured staleness threshold — these are separate, named states
- **FR9:** The system can indicate when provider data was last successfully refreshed

### Daemon Management

- **FR10:** The daemon can serve current provider usage data on a locally accessible HTTP endpoint
- **FR11:** The daemon can stream live usage updates to connected clients via server-sent events
- **FR12:** The daemon can report its operational health, uptime, active connection count, and listening port
- **FR13:** Any CLI subcommand can automatically start the daemon by probing the configured port for liveness; if the port is not responding, the CLI spawns the daemon and waits for it to become healthy before proceeding
- **FR14:** The daemon can detect when its configured port is already in use by another process and report the conflict explicitly
- **FR15:** Users can explicitly start, stop, and query the daemon from the CLI
- **FR16:** The daemon can perform an orderly shutdown on receiving a termination signal
- **FR17:** A new SSE subscriber receives the current provider usage state immediately on connection, without waiting for the next poll cycle

### Status Surface Delivery

- **FR18:** Users can retrieve a compact plain-text usage string for embedding in a tmux status bar
- **FR19:** Users can retrieve a structured JSON payload for embedding in a waybar custom module
- **FR20:** Users can retrieve a human-readable multi-provider usage summary for direct terminal display
- **FR21:** Users can configure custom per-provider display labels in all surfaces
- **FR22:** Users can configure the separator character between provider entries in compact surfaces
- **FR23:** The waybar payload includes a CSS class field with one of the following values: `ok`, `warning`, `critical`, `over-limit`, `error`, or `stale` — corresponding to the provider's current state and threshold zone
- **FR24:** `pakai status` supports a machine-readable JSON output mode
- **FR25:** Users can configure the display order of providers in compact surface output
- **FR26:** The setup command displays a live preview of what each configured surface will output after setup completes, confirming that data is being read correctly

### Setup & Configuration

- **FR27:** Users can run a setup command that detects which providers are installed based on known local file paths
- **FR28:** Users can run a setup command that creates a systemd user service unit for automatic daemon startup
- **FR29:** Users can run a setup command that prints a copy-pasteable configuration snippet for a named surface (tmux or waybar)
- **FR30:** Running setup again on an existing installation adds newly detected providers to the configuration without modifying existing provider configuration entries or disrupting a running daemon
- **FR31:** Users can read, write, and list individual configuration values from the CLI
- **FR32:** Configuration changes written via the CLI take effect within one daemon poll cycle, without restarting the daemon
- **FR33:** Individual providers can be enabled or disabled in configuration
- **FR34:** The system operates with default configuration when no config file is present
- **FR35:** The system validates configuration values on write and rejects invalid types or out-of-range values with an explicit error

### Alert & Adaptive Refresh

- **FR36:** Users can configure percentage thresholds that define alert zones (warning, critical, over-limit)
- **FR37:** The daemon automatically adjusts its provider poll interval based on current usage relative to configured thresholds
- **FR38:** Surface outputs reflect the current threshold zone using the CSS class values defined in FR23
- **FR39:** Alert threshold configuration applies uniformly across all enabled providers

### Developer & Integration Support

- **FR40:** Users can inject synthetic usage data for a provider to simulate arbitrary usage states; injected data is held in memory and does not persist across daemon restarts
- **FR41:** Users can remove injected synthetic data to restore live provider readings
- **FR42:** Third-party tools can query current usage data from the daemon's local HTTP endpoint
- **FR43:** Third-party tools can subscribe to live usage updates via the daemon's SSE endpoint
- **FR44:** Users can generate shell completion scripts for bash, zsh, and fish
- **FR45:** All CLI subcommands exit with status code 0 on success and a non-zero code on error

### Graceful Degradation

- **FR46:** Each provider maintains an independent health state (`ok`, `error`, or `stale`) that does not affect other providers
- **FR47:** When a provider's data source is inaccessible, the surface shows an explicit unavailability indicator (e.g. `??`) rather than zero or silence
- **FR48:** The system operates normally with a subset of configured providers when others are unreachable
- **FR49:** Users can view per-provider raw data and error details to diagnose adapter failures
- **FR50:** When a provider's data source becomes accessible again after an error or stale state, the daemon automatically restores the provider to an `ok` state without user intervention

## Non-Functional Requirements

### Performance

- `pakai tmux` and `pakai waybar` must return within **200ms** when the daemon is running and the cache is warm — these are invoked on every status bar refresh cycle
- End-to-end response (including auto-spawn) must complete within **3 seconds** on first call from a cold start
- The daemon's `/status` and `/health` HTTP endpoints must respond within **50ms** under normal operating load
- The daemon must serve up to **10 concurrent SSE clients** without throughput degradation to any individual subscriber
- Cache reads serving surface subcommands and background provider polling must run on separate goroutines — a slow provider fetch never blocks a `pakai tmux` call

### Reliability

- The daemon must run continuously for **7+ days** on a standard Linux desktop without crash or measurable heap growth
- A crashed daemon must be automatically restarted by systemd within **5 seconds** (via `Restart=on-failure` in the unit file)
- Provider adapter failures must not cause daemon exit — failed adapters return an error state; panics in adapters are recovered and logged
- The daemon must not leak goroutines on SSE client disconnect — every SSE goroutine exits cleanly when the client's request context is cancelled
- The daemon must not leave a stale PID file on unclean exit that prevents subsequent auto-spawn

### Security

- The HTTP endpoint binds exclusively to **127.0.0.1** (never `0.0.0.0` or any external interface) — pakai is explicitly a local-only service
- No usage data leaves the local machine — no telemetry, no analytics, no external API calls of any kind
- Provider source files are opened **read-only** — the daemon makes no writes to provider data files or databases
- All file access runs under the invoking user's permissions — no privilege escalation required or attempted

### Portability

- The Go binary must compile **without CGo** — no C toolchain required for contributors or CI runners
- Pre-built releases target **Linux/amd64** and **Linux/arm64**; daemon-only Windows/amd64 for pre-Tauri use
- All config, cache, and runtime paths follow the **XDG Base Directory Specification** (`$XDG_CONFIG_HOME`, `$XDG_DATA_HOME`, `$XDG_RUNTIME_DIR`)
- The binary must install via `go install github.com/dhanifudin/pakai/cmd/pakai@latest` with no additional setup steps

### Observability

- All provider parse errors are captured with full context (file path, field name, raw value) and surfaced through `pakai provider debug` — never swallowed silently
- Daemon log output goes to **stderr** at configurable verbosity; default level is silent during normal operation (no noise in systemd journal)
- The daemon's `/health` response includes structured fields for uptime, active connection count, port, and per-provider health state
- `pakai provider debug <name>` must display the exact file or database path being read, the raw data received, and any parse error — sufficient to file a useful bug report without running a debugger

### Accessibility

- Alert threshold states must be communicated through **both** a visual state (CSS class for color) and a textual/numeric indicator (the percentage or `??` value) — pakai must not rely on color as the sole signal of a critical state
