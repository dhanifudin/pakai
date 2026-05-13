# Story 3.3: Config CLI & Provider Management

Status: review

## Tasks / Subtasks

- [x] Add `pakai config get <key>` subcommand (AC: 1, 2)
  - [ ] Read `config.Config()`, navigate to key via dot-notation path
  - [ ] Print value on match, error + non-zero exit on miss
- [x] Add `pakai config set <key> <value>` subcommand (AC: 3, 4, 5)
- [x] Add `pakai config list` subcommand (AC: 6)
- [x] Implement config key schema + validation (AC: 4)
- [x] Implement atomic config write utility (AC: 5)
- [x] Integrate enabled/disabled flag into daemon poll loop (AC: 7, 8)

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Completion Notes List

- Created `internal/config/write.go` — `WriteConfigAtomic` with temp file + os.Rename
- Added `SetKey`/`GetKey` with dot-notation key schema and type/range validation
- Rewrote `cmd/pakai/main.go` config commands (set/get/list) for TOML config
- Removed viper from main.go imports
- Added enabled check in daemon `refresh.go` — skips disabled providers each cycle
- Config list shows all current key-value pairs in flat format

### File List
- `internal/config/write.go` (new)
- `internal/config/config.go` (modified)
- `internal/daemon/refresh.go` (modified)
- `cmd/pakai/main.go` (modified)

## Change Log
- 2026-05-12: Story 3.3 — config CLI (get/set/list), key schema validation, atomic write, provider enable/disable in daemon
