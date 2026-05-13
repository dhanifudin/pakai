---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'pakai - Unified AI Subscription Usage Tracker (CLI, tmux, waybar, Windows widget)'
session_goals: 'Brainstorm architecture, features, and implementation approach for a unified AI usage tracker similar to robinebers/openusage, supporting Claude and OpenCode Go, with CLI/tmux/waybar outputs and a post-MVP Tauri-based Windows widget'
selected_approach: 'progressive-flow'
techniques_used: ['What If Scenarios', 'Mind Mapping', 'SCAMPER Method', 'Decision Tree Mapping']
ideas_generated: [30]
workflow_completed: true
session_active: false
context_file: ''
---

# Brainstorming Session Results

**Facilitator:** Dian
**Date:** 2026-05-10

## Session Overview

**Topic:** pakai — Unified AI Subscription Usage Tracker
**Goals:** Design a unified tool that aggregates AI subscription usage (Claude, OpenCode Go) and surfaces it via CLI, tmux status bar, waybar (Linux), and a Tauri-based Windows widget (post-MVP)

### Session Setup

Inspired by robinebers/openusage. Key constraints and excitement points:
- No strong language preference — open to best fit
- Tauri for Windows widget brings genuine excitement (Rust + WebView)
- MVP focus: Claude + OpenCode Go usage tracking
- Multi-surface output: CLI, tmux, waybar, and later Windows widget

---

## Phase 1: Expansive Exploration — What If Scenarios

### Ideas Generated

**[Feature #1]**: Red-Line Alert System
_Concept:_ Real-time threshold alerts surfaced in tmux/waybar/widget. When usage crosses a configurable threshold (e.g. 80%, 90%, limit-in-N-days), the status indicator changes color/icon. Each surface can independently enable or disable alerts.
_Novelty:_ Unlike passive dashboards, this is ambient — it lives in your existing workflow without opening a new tool.

**[Feature #2]**: Config-as-CLI
_Concept:_ All configuration managed through `pakai config set/get/list` commands. Changes are immediately live across all output surfaces — no restart, no file editing required.
_Novelty:_ The CLI is the config UI. No separate config file the user has to hunt for.

**[Feature #3]**: Plugin-Style Provider Registry
_Concept:_ Providers are registered via CLI (`pakai provider add`) with a data-path and parsing schema. pakai discovers and reads them at runtime — no recompile needed. Ships with Claude and OpenCode Go as built-in defaults.
_Novelty:_ Turns pakai from a 2-provider tool into a community-extensible platform without framework overhead.

**[Architecture #4]**: Stateless Reader with Speed Cache
_Concept:_ pakai reads directly from provider sources (local files or API calls). API results are cached locally with a TTL (e.g. 60s) so tmux/waybar can refresh fast without hammering rate limits. No user-owned history — cache is ephemeral, not permanent.
_Novelty:_ The architecture stays honest — pakai is a viewer, not a database. Cache is an implementation detail, not a feature.

**[Architecture #5]**: Surface-Aware Subcommands
_Concept:_ Dedicated subcommands per output surface (`tmux`, `waybar`, `status`), each rendering the same underlying data in the format that surface expects natively. Shared data pipeline, divergent renderers.
_Novelty:_ Integration is copy-paste, not configuration. The binary knows what each consumer needs. waybar gets proper JSON schema, tmux gets a plain string.

**[Architecture #7]**: Lightweight Background Daemon
_Concept:_ `pakai daemon start` runs in background managing cache refresh and provider polling. Subcommands (`tmux`, `waybar`, `status`) read from daemon via local socket — instant response. Daemon also becomes the future subscription point for the Tauri widget.
_Novelty:_ Single source of truth for all surfaces. Cache is managed once, not per-process.

**[Architecture #8]**: Dual-Language Stack
_Concept:_ Go for daemon + CLI — static binary, low overhead, easy distribution. Rust + Tauri for the Windows widget, which talks to the Go daemon via local HTTP. Two binaries, one protocol.
_Novelty:_ Each language does what it's best at. Go owns the data layer, Rust/Tauri owns the rich UI layer.

**[Architecture #9]**: Local HTTP Daemon with SSE
_Concept:_ Go daemon runs a local HTTP server. REST endpoints for current status (`GET /status`), SSE endpoint for live updates (`GET /events`). All surfaces speak the same protocol.
_Novelty:_ Debuggable with curl. Tauri gets real-time push without custom IPC. Accidental open standard for third-party surfaces.

**[Architecture #10]**: Accidental Extension Platform
_Concept:_ The local HTTP API is undocumented but open — anyone can build Neovim plugins, shell prompt integrations, or scripts against `localhost:7731`. pakai becomes a platform without trying to be one.
_Novelty:_ Zero extra effort, infinite surface extensibility. The protocol is the plugin system.

**[Feature #11]**: Single Binary Distribution
_Concept:_ Everything in one Go binary with subcommands. `pakai daemon start` forks itself. Install once, get everything. Tauri widget is the only separate artifact, Windows-only.
_Novelty:_ Zero dependency install. `go install` or pre-built binary. The binary is its own process manager.

**[Architecture #12]**: Local-First Provider Strategy
_Concept:_ Each provider adapter checks for local data first (e.g. `~/.claude/` for Claude Code). Falls back to API call only when local data is absent or stale. Cache TTL prevents redundant API hits.
_Novelty:_ Offline-friendly. Respects rate limits naturally. Local reads are instantaneous.

**[Architecture #13]**: Minimal Normalized Usage Schema
_Concept:_ Each provider maps to a common struct: provider, plan, period, used, limit, unit, refreshed_at. No forced normalization across units — if Claude reports percent and OpenCode reports tokens, both are valid.
_Novelty:_ Honest about what providers actually expose. No lossy conversion. Schema extensible without changing core shape.

**[Feature #14]**: Zero-Config Onboarding
_Concept:_ `pakai init` auto-detects installed providers by scanning known config paths. Starts daemon. Prints ready-to-paste setup snippets for tmux and waybar. First-run under 30 seconds.
_Novelty:_ The tool meets the user where they are. Detection is best-effort, explicit override always available.

**[Feature #15]**: Auto-Spawning Daemon
_Concept:_ Any pakai subcommand transparently starts the daemon if it isn't running. Daemon can be configured to auto-exit after idle timeout or run persistently. `pakai daemon stop` for explicit shutdown.
_Novelty:_ The daemon is an implementation detail, not a user responsibility.

**[Feature #16]**: Floating Card Widget
_Concept:_ Tauri widget renders as a compact always-on-top floating card. Progress bars per provider, refresh timestamp, expand-on-click for details. Right-click context menu for settings and quit.
_Novelty:_ Not a system tray icon (too hidden) nor a full app (too heavy). Persistent ambient presence like a desktop clock widget.

**[Feature #17]**: Graceful Degradation Per Provider
_Concept:_ Each provider has an independent status field (`ok`, `error`, `stale`). One provider failing never breaks the others. All surfaces render a visible error indicator rather than silence. Last known good data shown with staleness timestamp.
_Novelty:_ Partial failure is the common case. pakai stays useful for the working providers.

**[Design Constraint #18]**: Provider-Labeled tmux Output
_Concept:_ tmux output always prefixes usage with a provider identifier: `claude:82% opencode:41%`. Provider label is non-negotiable — percentage without context is noise.
_Novelty:_ Forces legibility even under tight space constraints. Labels are configurable short-forms.

**[Feature #19]**: Configurable Provider Labels
_Concept:_ Each provider has a configurable short label for space-constrained surfaces (`pakai config set provider.claude.label cld`). Full name always used in `pakai status` and Tauri widget.
_Novelty:_ User decides the tradeoff between brevity and clarity for their own setup.

**[Feature #20]**: Configurable Output Separator
_Concept:_ The delimiter between providers in compact surfaces is user-configurable (`pakai config set format.separator " | "`). Default is a space. Applies to tmux and waybar text output.
_Novelty:_ Respects tmux power-user culture where status bar aesthetics are personal.

**[Feature #21]**: Local Bearer Token Auth
_Concept:_ Daemon generates a random token on first start, stored in `~/.config/pakai/token` (mode 0600). All HTTP clients must present it. CLI and Tauri read it automatically.
_Novelty:_ Security-by-default without user friction. Token rotation via `pakai daemon rotate-token`.

**[Architecture #22]**: Tauri as Daemon Supervisor on Windows
_Concept:_ On Windows, the Tauri widget spawns and owns the Go daemon process. Daemon lifetime is tied to widget lifetime. On Linux, daemon runs independently (auto-spawn). Same daemon binary, different supervisor per platform.
_Novelty:_ Sidesteps the Windows background process problem entirely. The widget IS the natural owner of the daemon.

**[Feature #23]**: System Tray Icon as Daemon Presence
_Concept:_ Tauri widget shows a system tray icon on Windows. Tray icon = daemon is running. Right-click: show/hide card, open settings, quit (stops daemon). Card can be closed without killing daemon — tray persists.
_Novelty:_ Visual affordance for daemon lifecycle that Windows users already understand.

**[Feature #24]**: Provider Mock Mode
_Concept:_ `pakai provider mock <name>` injects synthetic usage data into the daemon. Real provider adapters are bypassed. Mock state labeled in all outputs. Reset with `pakai provider unmock`.
_Novelty:_ Built-in demo and testing mode. No credentials needed in CI. Doubles as a way to show pakai to others without exposing real usage.

**[Architecture #25]**: Discoverable Provider Adapters
_Concept:_ Each provider adapter defines a list of candidate paths to scan. First match wins. Falls back to API. `pakai provider debug <name>` shows the scan order and which path matched.
_Novelty:_ Adapter is self-documenting about where it looks. Makes provider issues precisely reportable.

**[Feature #26]**: Over-Limit Display
_Concept:_ Usage exceeding 100% shown as-is — never capped. Each surface gets distinct visual treatment. waybar exposes a CSS class for over-limit state. Tauri shows overflow visual.
_Novelty:_ Honest rendering. Over-limit is important signal — capping at 100% would be actively misleading.

**[Feature #27]**: Background Update Notifications
_Concept:_ Daily version check against GitHub releases, result cached. Update notice surfaces only in `pakai status` and Tauri tray icon — never in tmux/waybar. Fully opt-out via config.
_Novelty:_ Respects the signal-to-noise contract of each surface. Status bar is sacred.

---

## Phase 2: Pattern Recognition — Mind Mapping

### Core Branches

- **Data Layer**: Local-first provider adapters → API fallback → TTL cache → normalized schema
- **Surfaces**: tmux (string) → waybar (JSON) → Tauri widget (SSE) → CLI status
- **Daemon**: HTTP server → SSE events → bearer auth → auto-spawn → Windows supervisor
- **Config/UX**: CLI config → provider labels → separator → init auto-detect → mock mode
- **Distribution**: Single Go binary (Linux) + Tauri widget (Windows-only)

### Emergent Core Values

**"Honest by Design"** — Over-limit display, provider labels, graceful degradation, stateless architecture. pakai never lies, hides, or smooths over bad data.

**"Zero Friction"** — Auto-spawn, `pakai init`, single binary, config-as-CLI. Every interaction is copy-paste simple.

**"Surface Contract"** — Each surface gets exactly what it needs and nothing more. tmux gets a string. waybar gets JSON. Tauri gets SSE. The daemon serves all identically.

---

## Phase 3: Idea Development — SCAMPER Method

### S — Substitute
**[SCAMPER-S]**: Daemon from Day One — Commit to HTTP daemon as foundation even for MVP. Avoids future refactor when Tauri arrives. One architecture, no migration tax.

### C — Combine
**[SCAMPER-C]**: Init + Systemd + Setup Combined — `pakai setup` replaces `pakai init`. Detects providers, writes systemd user unit, starts daemon, prints surface snippets. `pakai setup <surface>` prints copy-pasteable config on demand.

### A — Adapt
**[SCAMPER-A]**: Waybar-Style Format Strings — `{}` placeholder syntax familiar from waybar's own `format` field. `pakai config set format.tmux "{claude} | {opencode}"`. Zero new syntax to learn.

### M — Modify
**[SCAMPER-M #1]**: Adaptive Refresh Intervals — Daemon adjusts polling frequency by usage proximity to limit. <50%: 5min · 50–80%: 2min · >80%: 30s · >95%: 10s. Configurable per tier.

**[SCAMPER-M #2]**: Collapsed Setup Command — `pakai setup` as single onboarding entry point. Reduces top-level command surface.

### P — Put to Other Uses
**[SCAMPER-P #1]**: Shell Prompt Integration — `pakai tmux` works in starship/oh-my-zsh/pure with zero additional pakai code. Documentation is the feature.

**[SCAMPER-P #2]**: TUI Dashboard — `pakai dashboard` opens Bubble Tea full-terminal live view. Subscribes to SSE stream. Shows usage bars, period countdown, source path, daemon uptime. `[d]` debug mode shows raw provider JSON.

### E — Eliminate
**[SCAMPER-E]**: Drop Bearer Token for MVP — Localhost is already user-scoped on Linux. Auth adds complexity for minimal security gain. Revisit if remote access use cases emerge.

### R — Reverse
**[SCAMPER-R]**: Push-Based Provider Updates (Future) — Daemon exposes `POST /push/{provider}` for provider-initiated updates. Current polling remains fallback. Architecture already compatible.

### Deep Development
**[Feature #28]**: Dashboard Debug Mode — `[d]` in TUI toggles raw view: daemon JSON per provider, source path, cache age, HTTP response code. Built-in debugging, no separate command needed.

**[Feature #29]**: Unified Threshold Config — `alert.thresholds` drives both visual alert levels AND daemon refresh intervals. One config, coherent system behavior.

**[Feature #30]**: Complete Waybar Setup Output — `pakai setup waybar` prints full waybar module config + CSS with correct class names + `on-click` wired to `pakai dashboard`. Zero docs to read.

---

### Boundaries Established (Out of Scope for MVP)
- Cost estimation / pre-task budget check (#6)
- Usage profiles (work vs weekend modes)
- Permanent usage history / trend tracking

---

## Phase 4: Action Planning — Decision Tree Mapping

### Decisions Resolved

| Decision | Choice | Rationale |
|---|---|---|
| Repository structure | Monorepo | Shared versioning, API contract stays in sync across Go daemon and Tauri widget |
| MVP scope | CLI + daemon + tmux + waybar | All four surfaces ship together, immediately usable |
| Claude provider source | Local-first (`~/.claude/`), API fallback | Offline-friendly, no unnecessary API calls |
| OpenCode Go provider source | Local files | ⚠️ Path TBD — investigate source before writing adapter |
| Daemon lifecycle on Linux | Both: auto-spawn + systemd user unit | Auto-spawn = zero-config fallback; systemd = robust recommended path |
| Tauri widget release | Same monorepo, post-MVP cadence | Versioned together, shipped when ready |
| Bearer token auth | Eliminated for MVP | Localhost is already user-scoped; revisit for remote access |

### Monorepo Structure

```
github.com/dhanifudin/pakai/
├── cmd/pakai/          ← Go CLI entry point
├── internal/
│   ├── daemon/         ← HTTP server, SSE, adaptive refresh loop
│   ├── providers/
│   │   ├── claude/     ← local-first + API fallback adapter
│   │   └── opencode/   ← TBD pending source investigation
│   ├── cache/          ← TTL cache layer
│   └── renderer/       ← tmux, waybar, status, dashboard formatters
├── widget/             ← Tauri (post-MVP, Rust)
└── docs/
    └── setup/          ← waybar, tmux, starship integration guides
```

### Implementation Roadmap

**Week 1–2: Foundation**
- Monorepo scaffold + Go module
- Daemon: HTTP server, `GET /status`, `GET /events` SSE, `GET /health`
- Normalized schema, TTL cache
- Claude adapter (local-first, investigate `~/.claude/` structure)

**Week 3: OpenCode Go Adapter**
- Investigate OpenCode Go source → identify local file path → write adapter
- `pakai provider debug opencode` for scan-path visibility

**Week 4: Surfaces**
- `pakai tmux` — provider-labeled string, configurable labels + separator
- `pakai waybar` — JSON with CSS classes (warning/critical/over-limit)
- `pakai status` — human-readable multi-provider view
- Over-limit display (never cap at 100%)

**Week 5: UX + Polish**
- `pakai setup` — auto-detect + systemd unit + surface snippets
- `pakai setup waybar` — full config + CSS output
- `pakai config set/get/list`
- Alert system + adaptive refresh + unified threshold config
- Graceful degradation (provider `ok`/`error`/`stale` states)

**Week 6: Dashboard + Distribution**
- `pakai dashboard` (Bubble Tea + SSE subscription + debug mode)
- Provider mock mode (`pakai provider mock`)
- Single binary GitHub releases + AUR PKGBUILD

**Post-MVP:**
- Tauri widget (floating card + system tray + Windows daemon supervisor)
- Shell prompt integration documentation (starship, oh-my-zsh)
- Background update notifications
- Push-based provider webhook endpoint

### Open Blocker — RESOLVED

**OpenCode Go data source investigated 2026-05-11.**

- **File:** `~/.local/share/opencode/opencode.db` (SQLite)
- **Table:** `message`, column `data` (JSON), filter `role = 'assistant'`
- **Fields:** `providerID`, `modelID`, `cost` (USD float), `tokens.input`, `tokens.output`, `tokens.cache.read/write`, `time.created` (Unix ms)
- **Aggregate query:** SUM tokens + cost WHERE time.created >= month_start, GROUP BY providerID

**Schema impact:** OpenCode stores no subscription limits locally — only actual usage. The normalized schema `limit` field is optional. When absent, display degrades from percentage to absolute value (e.g. `opencode-go: $1.12` instead of `41%`). User can set a manual limit via `pakai config set provider.opencode.limit 10.00`.

**Adapter implementation:**
```sql
SELECT
  json_extract(data,'$.providerID') as provider,
  SUM(json_extract(data,'$.tokens.input'))  as input_tokens,
  SUM(json_extract(data,'$.tokens.output')) as output_tokens,
  SUM(json_extract(data,'$.cost'))          as total_cost_usd
FROM message
WHERE json_extract(data,'$.role') = 'assistant'
  AND json_extract(data,'$.time.created') >= ?  -- month start in Unix ms
GROUP BY provider;
```

---

## Session Summary and Insights

**Total Ideas Generated:** 30 across 4 phases
**Techniques Used:** What If Scenarios · Mind Mapping · SCAMPER · Decision Tree Mapping
**Session Duration:** Full progressive flow

### Key Achievements

- Complete architecture decided: Go daemon + HTTP/SSE + Rust/Tauri, monorepo
- MVP scope locked: CLI + daemon + tmux + waybar ship together
- 3 core values crystallized that shape every design decision
- All critical implementation decisions resolved except one (OpenCode Go source)
- Implementation roadmap with 6-week sequencing ready to execute

### Core Values (carry into all future decisions)

1. **Honest by Design** — Never cap, never hide, never smooth over bad data
2. **Zero Friction** — Copy-paste setup, auto-detection, single binary
3. **Surface Contract** — Each surface gets exactly what it needs, nothing more

### Breakthrough Insights

- The local HTTP daemon is an accidental extension platform — third parties can consume `localhost:7731` with zero extra work from pakai
- Unified threshold config (`alert.thresholds`) driving both visual alerts AND refresh intervals makes the system feel intelligent without extra configuration
- `pakai setup waybar` printing complete config + CSS closes the gap between pakai's output format and user's waybar config — documentation becomes unnecessary

### One Decision That Shaped Everything

Committing to the HTTP daemon from day one (not a cron/file approach) unlocks SSE for Tauri, makes the architecture consistent across all surfaces, and turns pakai into a debuggable, extensible local service. The complexity cost is front-loaded; the payoff compounds with every surface added.
