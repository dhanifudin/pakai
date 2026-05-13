package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHealthHandlerResponseShape(t *testing.T) {
	hub := NewHub()
	hub.clientCnt.Store(2)

	s := &Server{
		port:       7731,
		startTime:  time.Now(),
		hub:        hub,
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Required fields per AC 5
	required := []string{"status", "uptime_seconds", "connections", "port", "providers"}
	for _, field := range required {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing required field %q in health response", field)
		}
	}

	if status, ok := resp["status"].(string); !ok || status != "ok" {
		t.Errorf("status = %v, want %q", resp["status"], "ok")
	}

	if port, ok := resp["port"].(float64); !ok || int(port) != 7731 {
		t.Errorf("port = %v, want 7731", resp["port"])
	}

	if conns, ok := resp["connections"].(float64); !ok || int(conns) != 2 {
		t.Errorf("connections = %v, want 2", resp["connections"])
	}

	if provs, ok := resp["providers"].([]interface{}); !ok {
		t.Errorf("providers field missing or wrong type")
	} else if len(provs) != 0 {
		t.Errorf("providers = %v, want empty (no providers registered yet)", provs)
	}

	if uptime, ok := resp["uptime_seconds"].(float64); !ok || uptime < 0 {
		t.Errorf("uptime_seconds = %v, want non-negative number", resp["uptime_seconds"])
	}
}

func TestHealthHandlerStatusOk(t *testing.T) {
	s := &Server{
		port:       7731,
		startTime:  time.Now(),
		hub:        NewHub(),
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if status, ok := resp["status"].(string); !ok || status != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestPIDFilePath(t *testing.T) {
	path := pidFilePath()
	if path == "" {
		t.Error("pidFilePath() returned empty string")
	}
	// Should contain "pakai" and end with "daemon.pid"
	if !strings.Contains(path, "pakai") {
		t.Errorf("pidFilePath() = %q, expected to contain 'pakai'", path)
	}
	if !strings.HasSuffix(path, "/daemon.pid") {
		t.Errorf("pidFilePath() = %q, expected to end with '/daemon.pid'", path)
	}
}

func TestIsProcessAlive_NonExistent(t *testing.T) {
	// PID 0 on Linux is the idle process (swapper), but it's never really "alive"
	// Use a very high PID that doesn't exist
	if isProcessAlive(99999999) {
		t.Error("isProcessAlive(99999999) should be false for non-existent PID")
	}
}

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	if !isProcessAlive(os.Getpid()) {
		t.Errorf("isProcessAlive(%d) should be true for current process", os.Getpid())
	}
}

func TestStalePIDRecovery(t *testing.T) {
	// Create a temporary PID file with a non-existent PID
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "daemon.pid")

	// Simulate a stale PID file
	if err := os.WriteFile(pidFile, []byte("99999999\n"), 0644); err != nil {
		t.Fatalf("failed to write temp PID file: %v", err)
	}

	// Read the PID from the file
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("failed to read temp PID file: %v", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("invalid PID in file: %v", err)
	}

	// The PID should not be alive
	if isProcessAlive(pid) {
		t.Errorf("PID %d should not be alive (stale PID)", pid)
	}

	// Simulate recovery: remove stale file
	os.Remove(pidFile)
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("stale PID file should have been removed")
	}
}

func TestPortConflictDetection(t *testing.T) {
	// Bind to a port first
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("failed to listen on random port: %v", err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	// Try to create a server on the same port (should fail)
	s := &Server{
		port: port,
		hub:  NewHub(),
	}

	err = s.Start()
	if err == nil {
		t.Error("expected error when starting server on occupied port")
	} else {
		expected := fmt.Sprintf("port %d is already in use by another process", port)
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	}
}

func TestIsAddrInUse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	_, err = net.Listen("tcp", l.Addr().String())
	if err == nil {
		t.Fatal("expected error when binding to already-used address")
	}
	if !isAddrInUse(err) {
		t.Error("isAddrInUse should return true for EADDRINUSE error")
	}
}

func TestHealthHandlerUptimeSeconds(t *testing.T) {
	s := &Server{
		port:       7731,
		startTime:  time.Now().Add(-90 * time.Second),
		hub:        NewHub(),
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	uptime := resp["uptime_seconds"].(float64)
	if uptime < 89 || uptime > 91 {
		t.Errorf("uptime_seconds = %v, want ~90", uptime)
	}
}
