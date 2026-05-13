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

func TestHubInitialSnapshot(t *testing.T) {
	hub := NewHub()

	usages := []*schema.Usage{{
		Provider: "claude",
		Label:    "Claude",
		Status:   schema.StatusOK,
	}}
	hub.SetSnapshot(usages)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go hub.Subscribe(rec, req)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(30 * time.Millisecond)

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	// Read first SSE event — should be the initial snapshot
	scanner := bufio.NewScanner(resp.Body)
	var gotData string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
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

	// Set initial snapshot so first event is sent
	hub.SetSnapshot([]*schema.Usage{{
		Provider: "claude",
		Label:    "Claude",
		Status:   schema.StatusOK,
	}})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go hub.Subscribe(rec, req)
	time.Sleep(150 * time.Millisecond)

	hub.Broadcast([]*schema.Usage{{
		Provider: "opencode",
		Label:    "OpenCode",
		Status:   schema.StatusOK,
	}})

	time.Sleep(200 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	resp := rec.Result()
	var events []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events = append(events, line)
		}
	}

	if len(events) < 2 {
		t.Errorf("got %d events, want at least 2 (initial + broadcast)", len(events))
	}
}

func TestHubMaxClients(t *testing.T) {
	hub := NewHub()

	// Fill up to maxClients using TryIncrement directly (simulating pre-subscription)
	for i := 0; i < maxClients; i++ {
		if !hub.TryIncrement() {
			t.Fatalf("TryIncrement returned false at client %d (max=%d)", i+1, maxClients)
		}
	}

	// 11th should be rejected
	if hub.TryIncrement() {
		t.Error("TryIncrement should return false when at max clients")
	}

	// Decrement one, then should be able to add
	hub.Decrement()
	if !hub.TryIncrement() {
		t.Error("TryIncrement should return true after Decrement")
	}
}

func TestHubWaitGroup(t *testing.T) {
	hub := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.Subscribe(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe did not exit after context cancel")
	}

	hub.wg.Wait()
}

func TestHubBroadcastNoPanicOnClosedChannel(t *testing.T) {
	hub := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go hub.Subscribe(rec, req)
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast([]*schema.Usage{{Provider: "test", Status: schema.StatusOK}})
	cancel()
}

func TestHubEmptySnapshot(t *testing.T) {
	hub := NewHub()
	hub.SetSnapshot([]*schema.Usage{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go hub.Subscribe(rec, req)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(30 * time.Millisecond)

	resp := rec.Result()
	var gotJSON bool
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var parsed []interface{}
			if err := json.Unmarshal([]byte(data), &parsed); err == nil {
				gotJSON = true
			}
			break
		}
	}

	if !gotJSON {
		t.Error("empty snapshot should still send valid JSON array")
	}
}
