# Story 4.3: Warning & Critical CSS Classes

Status: review

## Tasks / Subtasks

- [x] Update `internal/renderer/waybar.go` to compute threshold-based class (AC: 1, 2, 3, 4, 5, 6, 7)
- [x] Implement worst-state precedence (AC: 5)
- [x] Verify `pakai tmux` still shows numeric percent in warning/critical (AC: 9)
- [x] Update waybar golden test files (AC: 1, 2, 3, 4, 5)

## Dev Agent Record

### Agent Model Used
deepseek-v4-pro

### Completion Notes List
- Added `classForUsage()` — per-provider class based on percentage vs hardcoded defaults (50/80)
- Updated `classPrecedence`: ok(0) < stale(1) < error(2) < warning(3) < critical(4) < over-limit(5)
- Error/stale providers bypass threshold evaluation
- Updated golden files to reflect 72% → warning class
- tmux already shows numeric percent unchanged (AC 9)

### File List
- `internal/renderer/waybar.go` (modified)
- `internal/renderer/testdata/waybar_ok.golden` (updated)
- `internal/renderer/testdata/waybar_two_ok.golden` (updated)

## Change Log
- 2026-05-12: Story 4.3 — warning/critical CSS classes, full classPrecedence, classForUsage
