package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

const maxClients = 10

type Hub struct {
	mu        sync.Mutex
	clients   map[chan []*schema.Usage]struct{}
	snapshot  []*schema.Usage
	clientCnt atomic.Int32
	wg        sync.WaitGroup
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[chan []*schema.Usage]struct{}),
	}
}

func (h *Hub) SetSnapshot(usages []*schema.Usage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshot = usages
}

func (h *Hub) TryIncrement() bool {
	if h.clientCnt.Load() >= maxClients {
		return false
	}
	h.clientCnt.Add(1)
	return true
}

func (h *Hub) Decrement() {
	h.clientCnt.Add(-1)
}

func (h *Hub) AddClient() {
	h.wg.Add(1)
}

func (h *Hub) RemoveClient() {
	h.wg.Done()
}

func (h *Hub) Subscribe(w http.ResponseWriter, r *http.Request) {
	if !h.TryIncrement() {
		http.Error(w, `{"error":"too many clients"}`, http.StatusServiceUnavailable)
		return
	}
	defer h.Decrement()

	h.AddClient()
	defer h.RemoveClient()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []*schema.Usage, 1)
	h.register(ch)
	defer h.deregister(ch)

	// Send initial snapshot immediately (AC 1)
	h.mu.Lock()
	snapshot := h.snapshot
	h.mu.Unlock()
	if snapshot != nil {
		h.writeSSEEvent(w, flusher, "update", snapshot)
	}

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			h.writeSSEEvent(w, flusher, "update", data)
		case <-heartbeat.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

func (h *Hub) Broadcast(usages []*schema.Usage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.snapshot = usages

	for ch := range h.clients {
		select {
		case ch <- usages:
		default:
		}
	}
}

func (h *Hub) Wait() {
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (h *Hub) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data []*schema.Usage) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	flusher.Flush()
}

func (h *Hub) register(ch chan []*schema.Usage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[ch] = struct{}{}
}

func (h *Hub) deregister(ch chan []*schema.Usage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}
