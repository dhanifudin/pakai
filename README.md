# PakAI (`pakai`)

> **Disclaimer**: This project was written 100% using AI.

Unified AI subscription usage tracker. Surface Claude, OpenAI Codex, and OpenCode Go usage — per provider, per window (5h / weekly / monthly) — directly in your tmux status bar, Waybar panel, CLI, or a TUI dashboard.

## Features

- **Multi-provider**: Claude (OAuth), OpenAI Codex (OAuth), OpenCode Go (local DB)
- **Per-window tracking**: 5 hour, weekly, monthly usage per provider
- **Multiple surfaces**: tmux, Waybar, `pakai status`, `pakai dashboard` (TUI)
- **Percentage-first**: worst-available percentage shown compactly; raw detail on hover/tooltip
- **Color-coded severity**: green → yellow → red per provider
- **Live daemon**: polls providers, serves `/status`/`/events` HTTP API
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

This detects installed providers, writes a systemd user unit, and prints ready-to-paste config for tmux and Waybar.

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
    "interval": 30,
    "format": "{}",
    "tooltip": true,
    "on-click": "ghostty -e /home/user/go/bin/pakai dashboard",
    "on-click-right": "ghostty -e /home/user/go/bin/pakai status"
}
```

Hover for per-window detail with progress bars. Click for full TUI dashboard.

## Supported providers

| Provider | Source | Windows | Setup required |
|----------|--------|---------|---------------|
| Claude | `~/.claude/stats-cache.json` + OAuth API | 5h, weekly, monthly | `claude login` |
| OpenAI Codex | `~/.codex/auth.json` OAuth | 5h, weekly | `codex login` |
| OpenCode Go | `~/.local/share/opencode/opencode-stable.db` (or `opencode.db`) | 5h, weekly, monthly | Subscribe at [opencode.ai](https://opencode.ai/auth) |

OpenCode Go per-window limits are automatically applied from the [official docs](https://opencode.ai/docs/go/): 5h=$12, weekly=$30, monthly=$60.

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
```

Config file: `$XDG_CONFIG_HOME/pakai/config.toml`

## Commands

| Command | Description |
|---------|------------|
| `pakai tmux` | Compact string for tmux status-right |
| `pakai waybar` | JSON (text + tooltip + class + percentage) for Waybar |
| `pakai status` | Human-readable per-provider breakdown |
| `pakai dashboard` | Interactive TUI with live updates |
| `pakai setup` | Detect providers, write systemd unit, print snippets |
| `pakai config set/get/list` | Manage configuration |
| `pakai daemon start/stop/status` | Control background daemon |
| `pakai provider debug <id>` | Inspect raw provider data |

## How it works

1. `pakai daemon` starts a background HTTP server (port 7731)
2. Polls each provider every 30s (adaptive: faster near limits)
3. `pakai tmux` / `pakai waybar` / `pakai status` query the daemon cache (<200ms)
4. Non-Claude providers with no native percent data use configurable dollar limits

## Development

```bash
git clone https://github.com/dhanifudin/pakai
cd pakai
go test ./...
go install ./cmd/pakai
```

## License

MIT
