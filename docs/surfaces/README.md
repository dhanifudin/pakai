# Output Surfaces

pakai can surface provider usage in several places. Each surface has its own page.

| Surface | Command / component | Best for |
|---|---|---|
| [CLI status](cli-status.md) | `pakai status` | Terminal, scripting |
| [tmux](tmux.md) | `pakai tmux` | tmux status bar |
| [Waybar](waybar.md) | `pakai waybar` | Sway / Hyprland / i3 panel |
| [TUI dashboard](dashboard.md) | `pakai dashboard` | Full-screen terminal view |
| [KDE Plasma widget](plasma.md) | Plasma 6 applet | KDE desktop panel |
| [DankMaterialShell (DMS)](dms.md) | DankBar plugin | DMS desktop |

## Common pattern

Every surface reads from the daemon cache (response < 200 ms). If the daemon is not running, the command auto-spawns it. You can also manage it explicitly:

```bash
pakai daemon start    # start in background
pakai daemon stop     # stop
pakai daemon status   # check pid / port
```

## Filtering what's shown

```bash
# Hide providers from widget/compact outputs (they're still fetched)
pakai config set widget.hidden "openai,opencode-go"

# Pin a single provider to show its percentage first
pakai config set widget.pinned claude
```
