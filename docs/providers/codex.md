# OpenAI Codex

Track your OpenAI Codex (chatgpt.com) subscription usage.

## Credential source

Choose the source in the PakAI widget Settings, or set it directly:

```bash
# Pi's openai-codex OAuth
pakai config set provider.openai.source pi

# OpenAI Codex CLI OAuth
pakai config set provider.openai.source codex
```

| Source | Credential file |
|---|---|
| Pi | `~/.pi/agent/auth.json` (`openai-codex`) |
| Codex CLI | `~/.codex/auth.json` |

## Enable

```bash
pakai config set provider.openai.enabled true
pakai daemon restart
```

Codex shows every rate-limit window returned by chatgpt.com.
