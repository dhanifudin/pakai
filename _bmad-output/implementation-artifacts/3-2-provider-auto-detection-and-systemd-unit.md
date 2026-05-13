# Story 3.2: Provider Auto-Detection & systemd Unit

Status: review

## Story

As a new user,
I want to run `pakai setup` and have it detect my installed providers, create a systemd user service, and print integration snippets,
so that I can reach a working status bar integration in under 30 seconds without manually editing config files.

## Acceptance Criteria

1. **Given** `~/.claude/stats-cache.json` exists, **When** I run `pakai setup`, **Then** it detects the Claude provider, prints a confirmation ("Detected: Claude at ~/.claude/stats-cache.json"), and adds Claude to `config.toml` if not already present

2. **Given** `~/.local/share/opencode/opencode.db` exists, **When** I run `pakai setup`, **Then** it detects the OpenCode provider, prints a confirmation, and adds OpenCode to `config.toml` if not already present

3. **Given** neither provider file is found at the known paths, **When** I run `pakai setup`, **Then** it prints a clear message explaining which paths were checked and exits with a non-zero code

4. **Given** `pakai setup` has detected at least one provider, **When** setup completes, **Then** it writes a systemd user unit file to `~/.config/systemd/user/pakai.service` with `Restart=on-failure` and `RestartSec=5`, and prints instructions to enable it

5. **Given** the systemd unit file already exists, **When** I run `pakai setup` again, **Then** it overwrites the unit file with the current template and does not error

6. **Given** `pakai setup` has run successfully, **When** setup completes, **Then** it prints a copy-pasteable tmux config snippet and a waybar module JSON block — one for each surface

7. **Given** a provider is already configured in `config.toml`, **When** I run `pakai setup` again, **Then** the existing provider config entries are preserved unchanged — no duplicate entries, no value resets, running daemon is not disrupted

## Tasks / Subtasks

- [x] Create `internal/detect/` package — provider path probing (AC: 1, 2, 3)
  - [x] `Detect() []Detection` — checks known paths, returns found providers
  - [x] Claude path: `~/.claude/stats-cache.json` (expand `~`)
  - [x] OpenCode path: `~/.local/share/opencode/opencode.db`
  - [x] Each `Detection`: `{ProviderID string, Path string, Found bool}`
  - [x] On "none found": caller prints which paths were checked
- [x] Create `internal/systemd/` package — unit file templating (AC: 4, 5)
  - [x] `WriteUnit(execPath string) error` — writes to `~/.config/systemd/user/pakai.service`
  - [x] Unit template: After=default.target, Restart=on-failure, RestartSec=5
  - [x] `execPath` = `os.Executable()` resolved path
  - [x] Always overwrite (idempotent — AC: 5)
  - [x] Create `~/.config/systemd/user/` directory if absent
- [x] Create `pakai setup` subcommand in `cmd/pakai/` (AC: 1, 2, 3, 4, 6, 7)
  - [x] Run `detect.Detect()`; print confirmations for found providers
  - [x] If none found: print which paths were checked, exit non-zero
  - [x] Write systemd unit file
  - [x] Print copy-pasteable tmux snippet and waybar module JSON
- [x] Implement idempotent config update

## Dev Notes

- **Prerequisite:** Stories 3.1 (config package) must be complete. This story uses `config.Config()` and writes to `config.toml`.
- **Idempotent config write (AC 7):** The setup command must read the existing config, add only missing providers, and write back. It must NOT overwrite existing provider configuration. Pattern:
  ```go
  cfg := config.Config() // current on-disk config
  for _, d := range detections {
      if d.Found && !hasProvider(cfg, d.ProviderID) {
          cfg.Provider[d.ProviderID] = defaultProviderConfig(d.ProviderID)
      }
  }
  writeConfigAtomic(cfg) // write-then-rename for atomicity
  ```
- **Atomic config write:** Write to a temp file in the same directory, then `os.Rename()`. This ensures a partially written TOML is never left on disk.
- **systemd unit `--foreground` flag:** The unit runs `pakai daemon start --foreground` so the process stays in the foreground for systemd to track. The `daemon start` command needs a `--foreground` flag that skips the detach/fork step.
- **tmux snippet to print:**
  ```
  # Add to ~/.tmux.conf:
  set -g status-right "#(pakai tmux)"
  set -g status-interval 30
  ```
- **waybar module JSON to print:**
  ```json
  "custom/pakai": {
    "exec": "pakai waybar",
    "return-type": "json",
    "interval": 30
  }
  ```
- **Path expansion:** `~` must be expanded to `os.UserHomeDir()` result. Use `filepath.Join(home, ".claude/stats-cache.json")` — never pass `~` directly to `os.Open`.
- **`detect/` package is separate from `systemd/`** — they have distinct test surfaces. Detection tests check file existence; systemd tests check unit template output.

### Project Structure Notes

```
internal/
├── detect/
│   ├── detect.go       ← Detect() probes known paths; returns []Detection
│   └── detect_test.go
└── systemd/
    ├── unit.go         ← WriteUnit(execPath); unit template; create dir
    └── unit_test.go
cmd/pakai/main.go       ← add 'setup' subcommand; wire detect + systemd + config write
```

### References

- [Source: architecture.md#Package Layout] — detect/ and systemd/ are separate packages at the end of impl sequence
- [Source: architecture.md#Infrastructure & Deployment] — systemd: `Restart=on-failure`, `RestartSec=5`
- [Source: architecture.md#Decision Impact Analysis] — detect/ and systemd/ implemented last (step 8)
- [Source: epics.md#Story 3.2] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Created `internal/detect/detect.go` — `Detect()` probes known paths (Claude: `~/.claude/stats-cache.json`, OpenCode: `~/.local/share/opencode/opencode.db`), returns `[]Detection` with `ProviderID/Path/Found`
- Created `internal/detect/detect_test.go` — 4 tests (found, not found, one-only, struct fields)
- Created `internal/systemd/unit.go` — `WriteUnit(execPath)` writes `pakai.service` to `~/.config/systemd/user/`; `UnitPath()` returns expected path; `Restart=on-failure`, `RestartSec=5`, `--foreground` flag
- Created `internal/systemd/unit_test.go` — 3 tests (write, overwrite, auto-create dir)
- Refactored `cmd/pakai/main.go` — `runSetup()` now uses `detect.Detect()` and `systemd.WriteUnit()`; removed old inline `writeSystemdUnit()` and hardcoded path checks; tmux snippet includes `status-interval 30`

### File List

- `internal/detect/detect.go` (new)
- `internal/detect/detect_test.go` (new)
- `internal/systemd/unit.go` (new)
- `internal/systemd/unit_test.go` (new)
- `cmd/pakai/main.go` (modified)

## Change Log

- 2026-05-12: Story 3.2 implementation — detect/ package for provider probing, systemd/ package for unit templating, refactored setup command
