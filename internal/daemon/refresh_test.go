package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/cache"
	"github.com/dhanifudin/pakai/internal/providers"
	"github.com/dhanifudin/pakai/internal/schema"
)

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
	done    chan struct{}
}

func (p *blockingProvider) ID() string { return "blocking" }

func (p *blockingProvider) Fetch(ctx context.Context) (*schema.Usage, error) {
	close(p.started)
	select {
	case <-p.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	close(p.done)
	return &schema.Usage{
		Provider:    p.ID(),
		Label:       p.ID(),
		Status:      schema.StatusOK,
		RefreshedAt: time.Now(),
	}, nil
}

func TestPollIntervalConfigControlsRefreshSchedule(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "pakai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[daemon]\npoll_interval = 300\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))

	loop := NewRefreshLoop(nil, cache.New(filepath.Join(t.TempDir(), "cache.json")), nil)
	if got, want := loop.adaptiveInterval(), 300*time.Second; got != want {
		t.Fatalf("adaptiveInterval() = %s, want configured poll interval %s", got, want)
	}
}

func TestAdaptiveIntervalDoesNotExceedConfiguredInterval(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "pakai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[daemon]\npoll_interval = 15\n[thresholds]\nwarning = 50\ncritical = 80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))

	loop := NewRefreshLoop(nil, cache.New(filepath.Join(t.TempDir(), "cache.json")), nil)
	loop.current = []*schema.Usage{{
		Provider: "test",
		Status:   schema.StatusOK,
		Used:     60,
		Limit:    100,
	}}
	if got, want := loop.adaptiveInterval(), 15*time.Second; got != want {
		t.Fatalf("adaptiveInterval() = %s, want configured maximum %s", got, want)
	}
}

func TestRefreshEndpointWaitsForFreshData(t *testing.T) {
	provider := &blockingProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	loop := NewRefreshLoop([]providers.Provider{provider}, cache.New(filepath.Join(t.TempDir(), "cache.json")), nil)
	s := &Server{loop: loop}

	req := httptest.NewRequest("POST", "/api/refresh", nil)
	rec := httptest.NewRecorder()
	handlerDone := make(chan struct{})
	go func() {
		s.handleRefresh(rec, req)
		close(handlerDone)
	}()

	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider fetch did not start")
	}

	returnedEarly := false
	select {
	case <-handlerDone:
		returnedEarly = true
	case <-time.After(50 * time.Millisecond):
	}

	close(provider.release)
	select {
	case <-provider.done:
	case <-time.After(time.Second):
		t.Fatal("provider fetch did not finish")
	}

	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("refresh handler did not return after fetch completed")
	}

	if returnedEarly {
		t.Fatal("refresh endpoint returned before provider fetch completed")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("response status = %q, want ok", response["status"])
	}
}
