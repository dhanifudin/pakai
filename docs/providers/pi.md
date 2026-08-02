# Pi

Track local token usage recorded by the [Pi coding agent](https://pi.dev) across all provider backends it uses.

## Overview

| | |
|---|---|
| **Provider ID** | `pi/<provider>` |
| **Data source** | `~/.pi/agent/sessions/**/*.jsonl` |
| **Windows** | Monthly (local token count) |
| **Unit** | tokens |

Pi reads session JSONL logs and exposes each backend as a separate sub-provider:

| Sub-provider | Backend |
|---|---|
| `pi/opencode` | OpenCode Zen via Pi |
| `pi/kosyayuk` | Kosyayuk model via Pi |
| `pi/<other>` | Any other model Pi has used |

> **Note**: This is local token usage recorded by Pi, not server-side billing balance or quota.

## Prerequisites

Pi must be installed and have been used at least once (to create session logs at `~/.pi/agent/sessions/`). No additional configuration is needed.

## Enable / setup

Pi sub-providers are auto-discovered when `~/.pi/agent/sessions/` exists. Run:

```bash
pakai setup
```

## Configuration

Pi providers have no server-side limit, so percentages require a manual limit:

```bash
# Set token limits per sub-provider to show usage as a percentage
pakai config set provider."pi/opencode".limit 1000000
pakai config set provider."pi/kosyayuk".limit 1000000

# Disable a specific Pi sub-provider
pakai config set provider."pi/opencode".enabled false
```

Config file: `$XDG_CONFIG_HOME/pakai/config.toml`

## Result

```
  pi/opencode
     month  234,512 tokens  (no limit set)
  pi/kosyayuk
     month   89,021 tokens  (no limit set)
```

Set limits to see percentages instead of raw token counts.

## Troubleshooting

| Error | Fix |
|---|---|
| Pi providers not appearing | Check `ls ~/.pi/agent/sessions/` — logs must exist |
| Token count seems wrong | pakai sums all `.jsonl` files in the sessions tree for the current month |
| `no usage` | No Pi sessions this month |
