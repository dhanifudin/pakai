# tmux

Compact one-line usage string for the tmux `status-right`.

## Setup

Add to `~/.tmux.conf`:

```
set -g status-right "#(/home/user/go/bin/pakai tmux)"
set -g status-interval 30
```

Or generate the snippet automatically:

```bash
pakai setup tmux
```

## Output format

```
󰚩 61% | 󱢆 100% | 󰘦 35%
```

Provider icons + worst-window percentage, separated by `|`. Color-coded:
- **Green** — under warning threshold
- **Yellow** — warning
- **Red** — critical or over-limit

## Result

<img src="../images/surfaces/tmux.svg" alt="pakai tmux output">

## Configuration

```bash
# Change the separator character
pakai config set display.separator " · "

# Change color thresholds
pakai config set thresholds.warning 50
pakai config set thresholds.critical 80

# Hide a provider from compact outputs
pakai config set widget.hidden openai
```
