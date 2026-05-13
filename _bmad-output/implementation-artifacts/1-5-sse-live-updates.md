# Story 1.5: SSE Live Updates

Status: review

## Story

As a user with a live waybar or external tool integration,
I want the daemon to push usage updates via server-sent events,
so that my status bar reflects changes immediately without polling on a fixed interval.

## Acceptance Criteria

1. **Given** a client connects to `GET /events`, **When** the connection is established, **Then** the daemon immediately sends an `event: update` SSE event with the current `[]Usage` JSON as `data` — no waiting for the next poll cycle

2. **Given** a client is subscribed to `GET /events` and the poll loop produces new data, **When** the cache is updated, **Then** the daemon sends an `event: update` SSE event with the updated `[]Usage` JSON within one poll interval

3. **Given** a client is subscribed and no data changes for 30 seconds, **When** the heartbeat timer fires, **Then** the daemon sends `event: heartbeat\ndata: {}\n\n` to keep the connection alive

4. **Given** a subscribed client disconnects, **When** the HTTP request context is cancelled, **Then** the SSE goroutine exits cleanly via `ctx.Done()` selection; the hub's `sync.WaitGroup` count decrements; no goroutine leak

5. **Given** 10 clients are already subscribed, **When** an 11th client connects to `GET /events`, **Then** the daemon returns HTTP 503 and does not create a goroutine for the rejected client

6. **Given** a third-party tool subscribes to `GET /events`, **When** usage data changes, **Then** it receives the same SSE stream as pakai's own clients — no special headers or auth required

7. **Given** the daemon receives SIGTERM while SSE clients are connected, **When** shutdown begins, **Then** the daemon waits up to 5 seconds for all SSE goroutines to exit (via `WaitGroup.Wait()`) before completing shutdown

## Tasks / Subtasks

- [x] Create `internal/daemon/hub.go` — SSE fan-out hub (AC: 1, 2, 3, 4, 5, 7)
  - [x] `Hub` struct with subscriber channels, `sync.WaitGroup`, atomic client count
  - [x] `Subscribe(w http.ResponseWriter, r *http.Request)` — enforces 10-client max (AC: 5)
  - [x] On subscribe: immediately send current snapshot (AC: 1)
  - [x] Fan-out: when poll loop updates cache, broadcast to all subscriber channels
  - [x] Heartbeat: goroutine sends `event: heartbeat\ndata: {}\n\n` every 30s per subscriber
  - [x] Goroutine lifecycle: every handler goroutine selects on `r.Context().Done()` (AC: 4)
  - [x] `WaitGroup.Add(1)` on subscribe, `WaitGroup.Done()` on exit via defer
- [x] Add `/events` endpoint to `internal/daemon/handlers.go` (AC: 1, 2, 3, 4, 5, 6)
  - [x] Set SSE headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
  - [x] Delegate to `hub.Subscribe(w, r)`
  - [x] Return HTTP 503 when client limit reached — no goroutine created (AC: 5)
- [x] Integrate hub with poll loop in `internal/daemon/daemon.go` (AC: 2)
  - [x] After cache update, call `hub.Broadcast(currentUsages)`
- [x] Integrate hub `WaitGroup` with graceful shutdown (AC: 7)
  - [x] After `server.Shutdown()`, call `hub.Wait()` with 5s timeout
- [x] Add tests for hub (AC: 4, 5, 7)
  - [x] Test goroutine leak: subscribe, disconnect, verify WaitGroup reaches 0
  - [x] Test max clients: 10 connected + 11th gets 503

## Dev Notes

- **Prerequisite:** Story 1.4 (surface outputs + daemon poll loop) must be complete. The hub integrates with the existing poll loop and shutdown path.
- **SSE goroutine contract (non-negotiable):** Every SSE handler goroutine MUST:
  ```go
  func (h *Hub) Subscribe(w http.ResponseWriter, r *http.Request) {
      if h.clientCount.Add(1) > maxClients {
          h.clientCount.Add(-1)
          http.Error(w, `{"error":"too many clients"}`, http.StatusServiceUnavailable)
          return
      }
      h.wg.Add(1)
      defer h.wg.Done()
      defer h.clientCount.Add(-1)
      ch := make(chan []model.Usage, 1)
      h.register(ch)
      defer h.deregister(ch)
      // send initial snapshot immediately
      writeSSEEvent(w, "update", currentSnapshot)
      for {
          select {
          case <-r.Context().Done():
              return
          case data := <-ch:
              writeSSEEvent(w, "update", data)
          case <-heartbeat.C:
              writeSSEEvent(w, "heartbeat", "{}")
          }
      }
  }
  ```
- **SSE wire format (exact):**
  ```
  event: update\ndata: [<json>]\n\n
  event: heartbeat\ndata: {}\n\n
  ```
  Two `\n` characters terminate each event. `fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ...)`. Flush after each write: `w.(http.Flusher).Flush()`.
- **Initial snapshot (AC 1):** On subscribe, send the current cache contents immediately before entering the select loop. This is called before the next poll fires.
- **Client limit enforcement:** Check before `wg.Add(1)` using an atomic counter. If over limit: write HTTP 503 and return — do NOT call `wg.Add(1)`.
- **Shutdown sequence:**
  ```go
  server.Shutdown(shutdownCtx)
  // after server shutdown closes all connections:
  waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  hub.Wait(waitCtx) // blocks until wg reaches 0 or timeout
  ```
- **No auth or special headers for third-party clients (FR43)** — the `/events` endpoint is open to any local process.
- **Flushing:** waybar reads SSE via `curl --no-buffer` or similar. Always flush after each write. Use `http.Flusher` typecast — fail gracefully if ResponseWriter doesn't support flushing.

### Project Structure Notes

```
internal/daemon/
├── daemon.go     ← integrate hub with poll loop and shutdown
├── hub.go        ← NEW: SSE Hub struct, Subscribe, Broadcast, Wait
└── handlers.go   ← add /events route → hub.Subscribe
```

### References

- [Source: architecture.md#Communication Patterns] — SSE goroutine contract (exact code pattern)
- [Source: architecture.md#Format Patterns] — SSE wire format, heartbeat timing
- [Source: architecture.md#Technical Constraints & Dependencies] — SSE goroutines, WaitGroup, max 10 clients
- [Source: epics.md#Story 1.5] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Created `internal/daemon/hub.go` — `Hub` struct with channel-based fan-out, `sync.WaitGroup` goroutine tracking, `atomic.Int32` client count, 10-client max enforcement with HTTP 503, 30s heartbeats
- Refactored `internal/daemon/server.go` — replaced embedded SSE logic (`sseClients`, `sseMu`, inline handleEvents/broadcast) with `Hub` delegation; health handler uses `hub.clientCnt.Load()`; Shutdown waits for hub goroutines via `hub.Wait()`
- SSE wire format: `event: update\ndata: [json]\n\n`, `event: heartbeat\ndata: {}\n\n` with flush after each write
- Created `internal/daemon/hub_test.go` — 6 tests: initial snapshot, broadcast, max clients (10+1=503), WaitGroup cleanup, broadcast no-panic on closed channel, empty snapshot
- Updated `internal/daemon/daemon_test.go` — health tests use `hub` instead of `sseClients`
- Verified: `curl /events` gets immediate snapshot, 11th concurrent client gets 503

### File List

- `internal/daemon/hub.go` (new)
- `internal/daemon/hub_test.go` (new)
- `internal/daemon/server.go` (modified)
- `internal/daemon/daemon_test.go` (modified)

## Change Log

- 2026-05-12: Story 1.5 implementation — SSE Hub with fan-out, 10-client max (HTTP 503), 30s heartbeat, WaitGroup lifecycle tracking, graceful shutdown wait
