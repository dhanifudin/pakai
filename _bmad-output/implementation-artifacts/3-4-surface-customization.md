# Story 3.4: Surface Customization

Status: review

## Tasks / Subtasks

- [x] Update `internal/renderer/tmux.go` to use provider ordering (AC: 4, 5)
- [x] Update `internal/renderer/waybar.go` to use provider ordering (AC: 4, 5)
- [x] Implement provider ordering in renderer (AC: 4, 5)
- [x] Ensure config is read fresh at render time (AC: 6)
- [x] Update golden test files

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Completion Notes List

- Added `orderUsages()` to `internal/renderer/tmux.go` — sorts alphabetically by default (AC 5), honors `display.order` from config when provided (AC 4)
- Updated `waybar.go` to use `orderUsages()` for consistent ordering
- Labels already resolved from config via `config.GetProviderLabel()` in daemon (AC 1, 2)
- Separator already passed from `config.GetSeparator()` at call site (AC 3)
- Live reload: renderers accept fresh config on each CLI invocation

### File List
- `internal/renderer/tmux.go` (modified)
- `internal/renderer/waybar.go` (modified)

## Change Log
- 2026-05-12: Story 3.4 — provider ordering in renderers
