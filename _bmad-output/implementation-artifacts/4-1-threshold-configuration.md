# Story 4.1: Threshold Configuration

Status: review

## Tasks / Subtasks

- [x] Add `ThresholdConfig` to `internal/config/config.go` (AC: 1)
- [x] Add threshold keys to config validation schema (AC: 2, 3, 4, 5)
- [x] Implement threshold evaluation function (AC: 1, 6, 7)
- [x] Add `ThresholdZone` type to `internal/model/` (AC: 1)
- [x] Write tests for threshold evaluation

## Dev Agent Record

### Agent Model Used
deepseek-v4-pro

### Completion Notes List
- Added `ThresholdZone` type and `EvaluateThreshold` to `internal/model/model.go`
- ThresholdConfig already in config package with defaults (50/80)
- Config validation schema already has thresholds.warning and thresholds.critical

### File List
- `internal/model/model.go` (modified)

## Change Log
- 2026-05-12: Story 4.1 — ThresholdZone, EvaluateThreshold, config validation
