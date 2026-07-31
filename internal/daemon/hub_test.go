package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func waitForHubClients(t *testing.T, hub *Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		hub.mu.Lock()
		got := len(hub.clients)
		hub.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("hub clients did not reach %d", want)
}

func waitForSubscription(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscription did not stop")
	}
}

func subscribe(t *testing.T, hub *Hub, rec *httptest.ResponseRecorder, ctx context.Context) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.Subscribe(rec, req)
	}()
	waitForHubClients(t, hub, 1)
	return cancel, done
}

func TestHubInitialSnapshot(t *testing.T) {
	hub := NewHub()
	hub.SetSnapshot([]*schema.Usage{{Provider: "claude", Label: "Claude", Status: schema.StatusOK}})
	rec := httptest.NewRecorder()
	cancel, done := subscribe(t, hub, rec, context.Background())
	cancel()
	waitForSubscription(t, done)

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}

	scanner := bufio.NewScanner(resp.Body)
	var gotData string
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "data: ") {
			gotData = line
			break
		}
	}
	if !strings.Contains(gotData, "claude") {
		t.Errorf("initial snapshot missing 'claude', got %q", gotData)
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	hub.SetSnapshot([]*schema.Usage{{Provider: "claude", Label: "Claude", Status: schema.StatusOK}})
	rec := httptest.NewRecorder()
	cancel, done := subscribe(t, hub, rec, context.Background())

	hub.Broadcast([]*schema.Usage{{Provider: "opencode", Label: "OpenCode", Status: schema.StatusOK}})
	cancel()
	waitForSubscription(t, done)

	resp := rec.Result()
	var events []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "data: ") {
			events = append(events, line)
		}
	}
	if len(events) < 2 {
		t.Errorf("got %d events, want at least 2 (initial + broadcast)", len(events))
	}
}

func TestHubMaxClients(t *testing.T) {
	hub := NewHub()
	for i := 0; i < maxClients; i++ {
		if !hub.TryIncrement() {
			t.Fatalf("TryIncrement returned false at client %d (max=%d)", i+1, maxClients)
		}
	}
	if hub.TryIncrement() {
		t.Error("TryIncrement should return false when at max clients")
	}
	hub.Decrement()
	if !hub.TryIncrement() {
		t.Error("TryIncrement should return true after Decrement")
	}
}

func TestHubWaitGroup(t *testing.T) {
	hub := NewHub()
	rec := httptest.NewRecorder()
	cancel, done := subscribe(t, hub, rec, context.Background())
	cancel()
	waitForSubscription(t, done)
	hub.wg.Wait()
}

func TestHubBroadcastNoPanicOnClosedChannel(t *testing.T) {
	hub := NewHub()
	rec := httptest.NewRecorder()
	cancel, done := subscribe(t, hub, rec, context.Background())
	hub.Broadcast([]*schema.Usage{{Provider: "test", Status: schema.StatusOK}})
	cancel()
	waitForSubscription(t, done)
}

func TestHubEmptySnapshot(t *testing.T) {
	hub := NewHub()
	hub.SetSnapshot([]*schema.Usage{})
	rec := httptest.NewRecorder()
	cancel, done := subscribe(t, hub, rec, context.Background())
	cancel()
	waitForSubscription(t, done)

	resp := rec.Result()
	var gotJSON bool
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var parsed []interface{}
			if json.Unmarshal([]byte(data), &parsed) == nil {
				gotJSON = true
			}
			break
		}
	}
	if !gotJSON {
		t.Error("empty snapshot should still send valid JSON array")
	}
}
