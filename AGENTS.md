# AGENTS.md — AI Coding Guidelines for pakai

## Project overview

pakai (PakAI) is a unified AI subscription usage tracker written in Go.
It surfaces Claude, OpenAI Codex, and OpenCode usage in tmux, Waybar, CLI status, and a TUI dashboard.

## Tech stack

- **Language**: Go 1.22+
- **CLI framework**: Cobra (`github.com/spf13/cobra`)
- **Config format**: TOML (`github.com/BurntSushi/toml`)
- **Database**: SQLite for OpenCode provider (`modernc.org/sqlite`)
- **HTTP**: stdlib `net/http`

## Architecture

```
cmd/pakai/main.go         — Cobra CLI entry point, all command definitions
internal/config/          — TOML config read/write, XDG-aware paths
internal/daemon/          — Background HTTP server (port 7731), PID management
internal/providers/       — One package per provider
  claude/                 — OAuth + stats-cache.json + API
  codex/                  — OpenAI Codex OAuth (chatgpt.com API)
  opencode/               — Local SQLite DB read
  mock/                   — Testing/demo provider
internal/client/          — HTTP client for daemon API
internal/renderer/        — tmux/waybar/status string formatters
internal/schema/          — Shared types (Usage, UsageWindow, Status)
internal/detect/          — Provider auto-detection logic
internal/systemd/         — Systemd unit generation
```

## Analyzing the codebase (Graphify)

A prebuilt knowledge graph lives in `graphify-out/` (4 k+ nodes, 4 k+ edges). Use it to
orient yourself before browsing raw files.

```bash
graphify query "what connects the daemon to the providers?"   # scoped subgraph
graphify path "daemon" "claude"                               # shortest path between two nodes
graphify explain "renderer"                                   # focused concept + neighbors
graphify update .                                             # refresh after code changes (AST-only, no API key)
```

Read `graphify-out/GRAPH_REPORT.md` for the broad subsystem / god-node overview.

**New machine setup** (graphify is already installed in `~/.local/bin` via uv):
```bash
uv tool install graphifyy        # PyPI package is graphifyy (double-y); CLI is graphify
graphify install --project       # register skill for this repo
graphify update .                # build code graph (no API key needed)
```

## Key patterns

- **Daemon-first**: commands try daemon cache first, fall back to direct fetch, then auto-spawn
- **Provider interface**: each provider implements `ID() string` and `Fetch(ctx) (*schema.Usage, error)`
- **Error surface**: provider errors are returned as `schema.Usage{Status: StatusError, Error: msg}`, not propagated as Go errors from `Fetch()`
- **File paths**: config in `$XDG_CONFIG_HOME/pakai/`, cache in `~/.cache/pakai/`, credentials from provider-native locations
- **Atomic writes**: config changes use temp file + rename to avoid corruption

## Development commands

```bash
go test ./...           # run all tests
go build ./cmd/pakai    # build binary
go install ./cmd/pakai  # install to $GOPATH/bin
```

## Code conventions

- No comments unless the WHY is non-obvious
- Wrap OS errors with context: `fmt.Errorf("description: %w", err)`
- Distinguish file-not-found from other IO errors using `os.IsNotExist(err)`
- File-not-found errors must be actionable — tell the user what to install or which command to run
- No feature flags or backwards-compat shims — just change the code
- Provider packages are self-contained; don't import one provider from another
- `schema.Usage.Error` carries user-facing error strings from providers
