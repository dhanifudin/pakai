# PakAI (`pakai`)

> **Disclaimer**: This project was written 100% using AI.

Unified AI subscription usage tracker. Surface Claude, OpenAI Codex, and OpenCode usage — per provider, per window (5h / weekly / monthly) — directly in your tmux status bar, Waybar panel, KDE Plasma widget, CLI, or a TUI dashboard.

## Features

- **Multi-provider**: Claude (OAuth), OpenAI Codex (OAuth), OpenCode (local SQLite DB), OpenCode Go (web dashboard)
- **Per-window tracking**: 5 hour, weekly, monthly usage per provider
- **OpenCode sub-providers**: automatically discovers provider backends (Anthropic, OpenAI, etc.) from the local DB with per-sub cost estimation
- **Multiple surfaces**: tmux, Waybar, KDE Plasma widget, `pakai status`, `pakai dashboard` (TUI)
- **Percentage-first**: worst-available percentage shown compactly; raw detail on hover/tooltip
- **Color-coded severity**: green → yellow → red per provider
- **Live daemon**: polls providers, serves `/status`/`/events` HTTP API (SSE)
- **Auto-detection**: `pakai setup` finds installed providers and prints config snippets

## Quick start

### Install

```bash
go install github.com/dhanifudin/pakai/cmd/pakai@latest
```

### Setup

```bash
pakai setup
```

This detects installed providers, writes a systemd user unit, starts the daemon, and prints ready-to-paste config for tmux and Waybar. A live preview is shown at the end.

### tmux

Add to `~/.tmux.conf`:

```
set -g status-right "#(/home/user/go/bin/pakai tmux)"
set -g status-interval 30
```

Compact output: `󰚩 61% | 󱢆 100% | 󰘦 35%`

### Waybar

Add to your Waybar config:

```jsonc
"custom/pakai": {
    "exec": "/home/user/go/bin/pakai waybar",
    "return-type": "json",
    "interval": 60,
    "format": "{}",
    "tooltip": true
}
```

CSS classes: `ok` (green), `warning` (yellow), `critical` (red), `over-limit` (bold red). Generate with `pakai setup waybar`.

### KDE Plasma Widget

A native Plasma 6 applet for KDE desktops.

**Install:**

```bash
# Copy the widget to Plasma's local package directory
cp -r plasma/com.dhanifudin.pakai ~/.local/share/plasma/plasmoids/

# Reload Plasma shell
plasmashell --replace &
# or log out and back in
```

**Add to panel:** Right-click panel → "Add or Remove Widgets" → search "PakAI" → drag to panel.

**Features:**
- Compact mode: colored dot + percentage in the panel
- Pin a provider to show its percentage directly in the panel
- Expanded popup: provider cards with per-window progress bars (5h / weekly / monthly)
- Hide individual providers from the widget
- Reserve projection on monthly windows ("X% in reserve" or "Runs out in Xm")
- OpenCode sub-providers grouped with per-sub progress bars
- 30s auto-refresh with manual refresh button
- Error state with daemon connection diagnostics

### DankMaterialShell Widget

A native DankBar plugin that follows the active DMS light, dark, or custom theme.

```bash
mkdir -p ~/.config/DankMaterialShell/plugins
ln -s "$PWD/dms/com.dhanifudin.pakai" ~/.config/DankMaterialShell/plugins/pakai
dms ipc call plugin-scan scan
```

Enable **PakAI** in DMS Settings → Plugins, then add it to the DankBar layout. The popout shows provider windows, reset times, errors, and **Refresh** and **Settings** actions. Settings opens DMS's plugin configuration, where the daemon URL, cache refresh interval, and optional bar label can be changed without editing QML.

## Supported providers

| Provider | Source | Windows | Setup required |
|----------|--------|---------|---------------|
| Claude | OAuth API + `~/.claude/stats-cache.json` | 5h, weekly | `claude login` |
| OpenAI Codex | `~/.codex/auth.json` OAuth | 5h, weekly | `codex login` |
| OpenCode (local) | `~/.local/share/opencode/opencode-stable.db` (or `opencode.db`) | 5h, weekly, monthly | [opencode.ai](https://opencode.ai/auth) |
| OpenCode Go | Web dashboard scraping | 5h, weekly, monthly | Set `OPENCODE_COOKIE` and `OPENCODE_WORKSPACE_ID` env vars |
| Pi custom providers | `~/.pi/agent/sessions/**/*.jsonl` | Monthly local tokens | Use the provider in Pi |

### OpenCode (local DB)

Reads the SQLite database created by OpenCode. Automatically discovers provider backends used (e.g. `opencode/anthropic`, `opencode/openai`). Each sub-provider gets its own usage entry with cost tracking.

Per-window limits are auto-applied from the [official docs](https://opencode.ai/docs/go/): 5h=$12, weekly=$30, monthly=$60. Configured limits on `opencode-go` are shared across sub-providers.

### Pi custom providers

Reads Pi session logs and automatically exposes each provider as `pi/<provider>`, including `pi/opencode` (OpenCode Zen) and `pi/kosyayuk`. This is local token usage recorded by Pi, not server-side billing balance or quota.

Optional limits can turn token totals into percentages:

```bash
pakai config set provider."pi/opencode".limit 1000000
pakai config set provider."pi/kosyayuk".limit 1000000
```

### OpenCode Go (web dashboard)

Scrapes the [opencode.ai billing dashboard](https://opencode.ai/workspace) for percentage-based usage. Requires two environment variables:

```bash
export OPENCODE_COOKIE="your-session-cookie"
export OPENCODE_WORKSPACE_ID="your-workspace-id"
```

Supports `.env` file loading from the current directory.

## Configuration

```bash
# Set subscription limits
pakai config set provider.openai.limit 20
pakai config set provider.opencode-go.limit 60

# View all config
pakai config list

# Change poll interval or thresholds
pakai config set daemon.poll_interval 30
pakai config set thresholds.warning 50
pakai config set thresholds.critical 80

# Display options
pakai config set display.separator " | "
```

Config file: `$XDG_CONFIG_HOME/pakai/config.toml`

## Widget Configuration

The Plasma widget and TUI dashboard share the same config options:

```bash
# Pin a provider to show first (panel compact text / dashboard top)
pakai config set widget.pinned claude

# Disable providers you do not use (removes fetching and display)
pakai config set provider.openai.enabled false
pakai config set provider.opencode-go.enabled false

# Hide providers from widgets only (comma-separated)
pakai config set widget.hidden "openai"
pakai config set widget.hidden "openai,opencode-go"

# Reset — show all providers, no pin
pakai config set widget.pinned ""
pakai config set widget.hidden ""
```

**Dashboard keybindings:** `q` quit, `r` refresh, `d` toggle debug (raw JSON).

## Commands

| Command | Description |
|---------|------------|
| `pakai tmux` | Compact string for tmux status-right |
| `pakai waybar` | JSON (text + tooltip + class + percentage) for Waybar |
| `pakai status` | Human-readable per-provider breakdown |
| `pakai status --json` | JSON output |
| `pakai dashboard` | Interactive TUI with live updates |
| `pakai setup` | Detect providers, write systemd unit, start daemon, print snippets |
| `pakai setup tmux` | Print tmux config line |
| `pakai setup waybar` | Print waybar module config and CSS |
| `pakai config set/get/list` | Manage configuration |
| `pakai daemon start/stop/status` | Control background daemon |
| `pakai provider debug <id>` | Inspect raw provider data |
| `pakai provider mock <id>` | Create a mock provider for testing |
| `pakai provider unmock <id>` | Remove a mock provider |
| `pakai version` | Print version |
| `pakai completion <bash|zsh|fish>` | Generate shell completion script |

## How it works

1. `pakai daemon` starts a background HTTP server (port 7731)
2. Polls each provider every 30s (adaptive: faster near limits)
3. `pakai tmux` / `pakai waybar` / `pakai status` query the daemon cache (<200ms)
4. Commands auto-spawn the daemon if not running
5. OpenCode sub-providers with no native percent data use configurable dollar limits
6. OpenCode Go probes the Zen API for credit exhaustion detection

## Development

```bash
git clone https://github.com/dhanifudin/pakai
cd pakai
go test ./...
go install ./cmd/pakai
```

## License

MIT
