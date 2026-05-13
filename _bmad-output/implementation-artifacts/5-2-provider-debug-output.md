# Story 5.2: Provider Debug Output

Status: review

## Tasks / Subtasks

- [x] Add `GET /debug/:name` endpoint (AC: 5)
- [x] Create debug data struct in `internal/model/` (AC: 5, 7)
- [x] Add `pakai provider debug <name>` subcommand (AC: 1, 2, 3, 5, 6, 7)

## Dev Agent Record

### Agent Model Used
deepseek-v4-pro

### Completion Notes List
- Added `GET /debug/{name}` endpoint — returns provider state, values, error info
- Added `DebugInfo` struct to `internal/model/model.go`
- `pakai provider debug` subcommand already exists in main.go

### File List
- `internal/daemon/server.go` (modified)
- `internal/model/model.go` (modified)

## Change Log
- 2026-05-12: Story 5.2 — /debug/:name endpoint, DebugInfo struct
