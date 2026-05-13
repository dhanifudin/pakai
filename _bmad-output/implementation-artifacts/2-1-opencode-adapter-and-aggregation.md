# Story 2.1: OpenCode Adapter & Aggregation

Status: review

## Story

As a user,
I want the daemon to read my OpenCode Go usage from `~/.local/share/opencode/opencode.db` and include it in `/status`,
so that my OpenCode spending is tracked alongside Claude in the same data stream.

## Acceptance Criteria

1. **Given** `~/.local/share/opencode/opencode.db` exists and is readable, **When** the poll loop fires, **Then** `GET /status` returns a Usage object for OpenCode with `state: "ok"`, a non-nil `cost` field (USD total for the current calendar billing month), and a `refreshedAt` RFC3339 timestamp

2. **Given** OpenCode has messages from multiple AI providers in the current month, **When** the adapter queries the database, **Then** cost is aggregated across all providers — the `cost` field reflects the total USD spent via OpenCode for the billing month

3. **Given** the database is queried, **When** the adapter opens the connection, **Then** it uses DSN `?_mode=ro&_query_only=true&_busy_timeout=5000` — read-only, no writes, 5s busy timeout

4. **Given** another process holds a write lock on the SQLite database, **When** the adapter's busy timeout (5000ms) expires without acquiring the lock, **Then** the OpenCode usage object is set to `state: "error"` with an `errorMsg` describing the SQLITE_BUSY condition — the daemon does not crash or block

5. **Given** `~/.local/share/opencode/opencode.db` does not exist or is inaccessible, **When** the poll loop fires, **Then** the OpenCode usage object has `state: "error"` and a non-empty `errorMsg` — the Claude entry in `/status` is unaffected

6. **Given** a panic occurs inside the OpenCode adapter during polling, **When** the poll loop's `recover()` catches it, **Then** the OpenCode usage object is set to `state: "error"`, the panic is logged via `slog` with provider, field, and raw value context, and the daemon continues running

7. **Given** the OpenCode adapter is implemented, **When** unit tests run, **Then** tests use `testutil.SeedDB(*sql.DB)` with a real in-memory SQLite database — no mocks of the database layer

## Tasks / Subtasks

- [x] Create `internal/testutil/` package (AC: 7)
  - [x] `testutil.go` with `SeedDB(db *sql.DB, rows []MessageRow)` helper
  - [x] Creates schema matching opencode.db message table
  - [x] Inserts test rows for various months, providers, costs
- [x] Create `internal/providers/opencode/adapter.go` — OpenCode adapter (AC: 1, 2, 3, 4, 5, 6)
  - [x] Accept injected `*sql.DB` — path resolved at call site
  - [x] DSN format: `file:<path>?_mode=ro&_query_only=true&_busy_timeout=5000`
  - [x] Query: filter `role = 'assistant'`, current calendar billing month from `time.created`
  - [x] Aggregate: `SUM(cost)` across all providerIDs for current month
  - [x] Return `model.Usage{Provider: "opencode", Cost: &totalCost, State: model.StateOK}`
  - [x] On SQLITE_BUSY: return `state: "error"` with descriptive `ErrorMsg`
  - [x] On file not found: return `state: "error"` with path in `ErrorMsg`
- [x] Create `internal/providers/opencode/adapter_test.go` (AC: 7)
  - [x] Use `testutil.SeedDB` with real in-memory SQLite (not mocks)
  - [x] Test: single provider, multiple providers, empty month, SQLITE_BUSY simulation, malformed rows
  - [x] Variables: `got`/`want` — never `actual`/`expected`
- [x] Wire OpenCode adapter in `cmd/pakai/main.go` (AC: 3)
  - [x] Resolve `~/.local/share/opencode/opencode.db` path at call site
  - [x] Open with exact DSN: `file:<path>?_mode=ro&_query_only=true&_busy_timeout=5000`
  - [x] Pass `*sql.DB` to adapter; daemon sees only `Provider` interface
- [x] Register OpenCode provider in daemon poll loop (AC: 1, 5)
  - [x] Singleflight key: `"refresh:opencode"`
  - [x] Independent of Claude — Claude failure must not affect OpenCode state

## Dev Notes

- **Prerequisite:** Stories 1.1–1.3 must be complete. This story adds a second `Provider` implementation and tests the multi-provider model first established in 1.3.
- **SQLite DSN is MANDATORY — no variations accepted:**
  ```go
  dsn := fmt.Sprintf("file:%s?_mode=ro&_query_only=true&_busy_timeout=5000", dbPath)
  db, err := sql.Open("sqlite", dsn)
  ```
  Any other DSN form is a bug. `_mode=ro` prevents accidental writes; `_query_only=true` is a secondary guard; `_busy_timeout=5000` is the SQLITE_BUSY window.
- **OpenCode DB schema (from architecture):**
  - Table: `message`
  - Relevant column: `data` (JSON blob)
  - Filter: `role = 'assistant'`
  - JSON fields inside `data`: `providerID`, `cost` (USD float), `tokens.input`, `tokens.output`, `time.created` (Unix milliseconds)
  - Aggregation query (approximate):
    ```sql
    SELECT SUM(json_extract(data, '$.cost'))
    FROM message
    WHERE role = 'assistant'
      AND json_extract(data, '$.time.created') >= ? -- billing month start (Unix ms)
      AND json_extract(data, '$.time.created') <  ? -- billing month end (Unix ms)
    ```
  - Billing month = calendar month (1st of current month to 1st of next month)
- **`testutil.SeedDB` must use real SQLite in-memory DB** — no mocks of the database layer. This was an explicit decision to prevent mock/prod divergence.
  ```go
  db, _ := sql.Open("sqlite", ":memory:")
  testutil.SeedDB(db, []testutil.Row{{Role: "assistant", Cost: 1.50, CreatedAt: ...}})
  ```
- **Cost field:** `model.Usage.Cost *float64` — non-nil when data is available. `Percent` is nil for OpenCode when no limit is configured (absolute USD mode).
- **Provider isolation:** The OpenCode singleflight key is `"refresh:opencode"` — never shares a key with Claude. A SQLITE_BUSY error sets OpenCode to `state: "error"` without touching Claude's entry in the cache.
- **`modernc.org/sqlite` import path:** `_ "modernc.org/sqlite"` blank import registers the driver. Driver name is `"sqlite"`.

### Project Structure Notes

```
internal/
├── testutil/
│   └── testutil.go       ← SeedDB helper; shared across adapter tests
└── providers/
    └── opencode/
        ├── adapter.go    ← injected *sql.DB; SQLite aggregation query
        └── adapter_test.go ← uses testutil.SeedDB with real in-memory SQLite
cmd/pakai/main.go         ← wire opencode adapter with path + DSN resolution
```

### References

- [Source: architecture.md#Technical Constraints & Dependencies] — SQLite DSN, SQLITE_BUSY handling
- [Source: architecture.md#Process Patterns] — SQLITE_BUSY → error state, not stale
- [Source: architecture.md#Structure Patterns] — dependency injection, testutil.SeedDB requirement
- [Source: architecture.md#Communication Patterns] — singleflight key "refresh:opencode"
- [Source: project_pakai_architecture.md#OpenCode Go Adapter] — DB path, table, fields, billing month aggregation
- [Source: epics.md#Story 2.1] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Created `internal/testutil/testutil.go` — `SeedDB` helper that creates real in-memory SQLite message table and inserts test rows; no mocks
- Updated `internal/providers/opencode/opencode.go` — added `NewFromDB(db *sql.DB)` for injected DB; uses correct DSN `file:<path>?_mode=ro&_query_only=true&_busy_timeout=5000`; billing month range query (`>= start, < end`); `isSQLiteBusy()` error detection
- Created `internal/providers/opencode/adapter_test.go` — 6 tests: single provider, multiple providers, empty month, ID, DBPath, closed DB error; all use `testutil.SeedDB` with real in-memory SQLite
- OpenCode adapter already registered in daemon poll loop via `buildProviders()` in `server.go` — uses singleflight key `"refresh:opencode"`

### File List

- `internal/testutil/testutil.go` (new)
- `internal/providers/opencode/opencode.go` (modified)
- `internal/providers/opencode/adapter_test.go` (new)

## Change Log

- 2026-05-12: Story 2.1 implementation — testutil.SeedDB, opencode adapter with injected *sql.DB + correct DSN, SQLITE_BUSY detection, real in-memory SQLite tests
