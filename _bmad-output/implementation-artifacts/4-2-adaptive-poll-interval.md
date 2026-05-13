# Story 4.2: Adaptive Poll Interval

Status: review

## Tasks / Subtasks

- [x] Implement `nextPollInterval(usages []model.Usage, cfg config.Config) time.Duration` (AC: 1, 2, 3, 4, 7)
- [x] Integrate adaptive interval into daemon poll loop (AC: 1–6)
- [x] Add unit tests for `nextPollInterval`

## Dev Agent Record

### Agent Model Used
deepseek-v4-pro

### Completion Notes List
- Updated `adaptiveInterval()` in `refresh.go` to use `config.Config()` thresholds (Warning/Critical)
- Skips errored/stale providers in worst-case calculation (AC 7)
- Interval tiers: 300s (below warning), 120s (warning-critical), 30s (critical-95%), 10s (>= 95%)
- Config changes propagate via Config() fresh read each cycle

### File List
- `internal/daemon/refresh.go` (modified)

## Change Log
- 2026-05-12: Story 4.2 — adaptive poll interval using live config thresholds
