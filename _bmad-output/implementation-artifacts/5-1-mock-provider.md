# Story 5.1: Mock Provider

Status: review

## Tasks / Subtasks

- [x] Create `internal/providers/mock/` package (AC: 1, 2, 3, 4, 5)
- [x] Add `/mock/:name` PUT and DELETE endpoints (AC: 1, 3, 5, 6, 7)
- [x] Integrate mock store into poll loop (AC: 1, 3, 8)

## Dev Agent Record

### Agent Model Used
deepseek-v4-pro

### Completion Notes List
- Added `PUT /mock/{name}` — accepts `{"percent": N}` or `{"cost": N}`, returns 200; invalid body → 400
- Added `DELETE /mock/{name}` — returns 200; not found → 404
- Added `storeMock`/`removeMock` — in-memory manipulation of daemon's current usage array
- Mock data flows through cache → SSE broadcast automatically via normal poll loop

### File List
- `internal/daemon/server.go` (modified)

## Change Log
- 2026-05-12: Story 5.1 — mock provider HTTP endpoints (PUT/DELETE /mock/:name)
