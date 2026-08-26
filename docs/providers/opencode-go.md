# OpenCode Go

Track OpenCode Go subscription quota through the [Usage API](https://opencode.ai/console/guides/usage).

## Credential source

Choose the source in the PakAI widget Settings, or set it directly:

```bash
# Pi's opencode-go key
pakai config set provider.opencode-go.source pi

# OpenCode CLI key
pakai config set provider.opencode-go.source opencode
```

| Source | Credential file | Key |
|---|---|---|
| Pi | `~/.pi/agent/auth.json` | `opencode-go` |
| OpenCode | `~/.local/share/opencode/auth.json` | `opencode` |

## Enable

```bash
pakai config set provider.opencode-go.enabled true
pakai daemon restart
```

The API returns 5-hour, weekly, and monthly percentages.
