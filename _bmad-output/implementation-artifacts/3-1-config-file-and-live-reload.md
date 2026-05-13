# Story 3.1: Config File & Live Reload

Status: review

## Story

As a user,
I want pakai to read configuration from `$XDG_CONFIG_HOME/pakai/config.toml` and reflect changes without restarting the daemon,
so that I can tune behavior without disrupting a running status bar integration.

## Acceptance Criteria

1. **Given** no config file exists at `$XDG_CONFIG_HOME/pakai/config.toml`, **When** the daemon starts or any CLI subcommand runs, **Then** it operates with built-in defaults (30s poll interval, port 7731, default labels) — no error, no missing-file warning

2. **Given** a valid config file exists, **When** `config.Config()` is called from any package, **Then** it returns the current on-disk values — it never returns a snapshot cached at startup

3. **Given** the config file is updated on disk (e.g. via `pakai config set`), **When** the daemon's next poll cycle fires, **Then** the new config values are used for that cycle — changes propagate within one poll interval without daemon restart

4. **Given** fsnotify detects a write to `config.toml`, **When** the watcher fires, **Then** the internal config state is invalidated so the next `config.Config()` call reads fresh values

5. **Given** fsnotify is unavailable or fails to watch, **When** 5 seconds elapse, **Then** the config is re-read via the fallback poll — live reload still works, just with up to 5s additional latency

6. **Given** the config file contains a syntax error (invalid TOML), **When** `config.Config()` is called, **Then** it returns the last known-good config and logs the parse error via `slog` — the daemon continues running with stale-but-valid config rather than crashing

## Tasks / Subtasks

- [x] Create `internal/config/config.go` — Config struct + live `Config()` accessor (AC: 1, 2, 6)
  - [x] Define `Config` struct with all fields and built-in defaults
  - [x] `Config()` function: reads `$XDG_CONFIG_HOME/pakai/config.toml` on every call (never snapshot)
  - [x] If file absent: return defaults silently (AC: 1)
  - [x] If TOML parse error: return last known-good config, log via `slog` (AC: 6)
  - [x] XDG path: `os.Getenv("XDG_CONFIG_HOME")` with fallback `~/.config`
- [x] Create `internal/config/watcher.go` — fsnotify + fallback poll (AC: 3, 4, 5)
  - [x] fsnotify watcher on `config.toml` — on WRITE/CREATE event: invalidate cache flag
  - [x] Fallback poll: if fsnotify fails to initialize, poll every 5s via ticker
  - [x] On invalidation: next `config.Config()` call reads fresh from disk
  - [x] Watcher goroutine started by daemon at startup; stopped on shutdown
- [x] Create `internal/config/config_test.go` (AC: 1, 2, 6)
  - [x] Test: absent config returns defaults
  - [x] Test: present config returns on-disk values
  - [x] Test: parse error returns last-good config
  - [x] Test: XDG_CONFIG_HOME path resolution
- [x] Integrate config watcher startup/shutdown into daemon lifecycle
  - [x] Start watcher goroutine in `daemon.Start()`
  - [x] Stop watcher on `daemon.Shutdown()`
- [x] Ensure daemon poll loop calls `config.Config()` fresh each cycle (AC: 2, 3)
  - [x] Verify no config snapshot stored in Daemon struct — call site freshness

## Dev Notes

- **Prerequisite:** Story 1.3+ must be complete. This story formalizes the config package that has been informally used in earlier stories (port 7731, poll interval defaults).
- **`Config()` is called fresh at every use site — never captured:**
  ```go
  // CORRECT
  cfg := config.Config()
  pollInterval := cfg.Daemon.PollInterval
  
  // WRONG — snapshot at startup, misses live reload
  type Daemon struct { cfg config.Config }
  ```
  This is one of the most important patterns in the codebase. Violating it silently breaks FR32.
- **Viper is explicitly prohibited** — it was evaluated and dropped because it silently normalizes config keys to lowercase, corrupting provider-namespaced keys like `provider.opencode.limit`. Use `BurntSushi/toml` directly.
- **Config struct fields (minimum for MVP):**
  ```go
  type Config struct {
      Daemon    DaemonConfig
      Provider  map[string]ProviderConfig
      Display   DisplayConfig
      Thresholds ThresholdConfig
  }
  type DaemonConfig struct {
      Port         int    // default: 7731
      PollInterval int    // default: 30 (seconds)
  }
  type ProviderConfig struct {
      Enabled bool
      Label   string
      Limit   *float64 // optional; for OpenCode manual limit
  }
  type DisplayConfig struct {
      Separator string   // default: " | "
      Order     []string // default: alphabetical
  }
  type ThresholdConfig struct {
      Warning  int // default: 50
      Critical int // default: 80
  }
  ```
- **Live reload mechanism:** The watcher sets an `atomic.Bool` "dirty" flag. `Config()` checks the flag: if dirty, reads from disk and clears flag; if clean, re-reads from disk anyway (config.Config() always reads fresh — the dirty flag is an optimization hint for future caching if needed, but for MVP just read each call).
- **Parse error handling:** Store `lastGoodConfig` as a package-level variable. On parse error: return `lastGoodConfig`, log with `slog.Error("config parse error", "path", configPath, "err", err)`.
- **XDG path resolution:**
  ```go
  func configPath() string {
      base := os.Getenv("XDG_CONFIG_HOME")
      if base == "" {
          home, _ := os.UserHomeDir()
          base = filepath.Join(home, ".config")
      }
      return filepath.Join(base, "pakai", "config.toml")
  }
  ```

### Project Structure Notes

```
internal/config/
├── config.go       ← Config struct, Config() accessor, XDG path resolution
├── watcher.go      ← fsnotify + 5s fallback poll; invalidation mechanism
└── config_test.go
```

### References

- [Source: architecture.md#Package Layout] — config/ package description: "TOML decode, fsnotify watcher + 5s fallback poll, live Config() accessor (never snapshot at startup)"
- [Source: architecture.md#Structure Patterns] — Config() access pattern (correct vs wrong)
- [Source: architecture.md#Initialization] — viper explicitly dropped; BurntSushi/toml chosen
- [Source: architecture.md#Technical Constraints & Dependencies] — polling over inotify for adapters (watcher uses fsnotify for config specifically)
- [Source: epics.md#Story 3.1] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Rewrote `internal/config/config.go` — replaced viper/YAML with BurntSushi/toml; `AppConfig` struct with Daemon, Provider, Display, Thresholds; `Config()` reads TOML from `$XDG_CONFIG_HOME/pakai/config.toml` on every call (never snapshot); XDG path fallback to `~/.config`; parse error returns last-good config with `slog.Error`; built-in defaults (port 7731, poll 30s); maintained all legacy accessor functions (GetDaemonPort, GetProviderLabel, etc.)
- Created `internal/config/watcher.go` — fsnotify WRITE/CREATE watch on config.toml; 5s fallback ticker poll; `Watcher` struct with Start/Stop
- Created `internal/config/config_test.go` — 9 tests: XDG path, fallback path, absent defaults, TOML read, parse error recovery, provider config, default values, GetDaemonPort, GetSeparator
- Integrated watcher with daemon: `Server.watcher` field, started in `Start()`, stopped in `Shutdown()`

### File List

- `internal/config/config.go` (modified)
- `internal/config/watcher.go` (new)
- `internal/config/config_test.go` (new)
- `internal/daemon/server.go` (modified)
- `go.mod` (modified)
- `go.sum` (modified)

## Change Log

- 2026-05-12: Story 3.1 implementation — viper→BurntSushi/toml migration, XDG_CONFIG_HOME path, Config() always-fresh accessor, fsnotify + 5s fallback watcher, parse error recovery with last-good config
