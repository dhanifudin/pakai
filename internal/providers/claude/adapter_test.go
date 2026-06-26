package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func TestFetch_APIWindows(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	windows := []schema.UsageWindow{
		{Key: "5h", Label: "5h", Used: 30, Limit: 100, Unit: "percent"},
		{Key: "weekly", Label: "weekly", Used: 70, Limit: 100, Unit: "percent"},
	}
	cached := cachedUsageWindows{FetchedAt: now, Windows: windows}
	data, _ := json.Marshal(cached)

	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "claude-usage.json")
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("failed to seed API cache: %v", err)
	}

	p := New()
	p.apiCachePath = cachePath
	p.now = func() time.Time { return now }

	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}
	if string(got.Status) != "ok" {
		t.Errorf("got Status = %q, want %q", got.Status, "ok")
	}
	if len(got.Windows) != 2 {
		t.Fatalf("got %d windows, want 2", len(got.Windows))
	}
	for _, w := range got.Windows {
		if w.Key == "monthly" || w.Unit == "messages" {
			t.Errorf("unexpected monthly/messages window: %+v", w)
		}
	}
	if got.Windows[0].Key != "5h" {
		t.Errorf("got windows[0].Key = %q, want %q", got.Windows[0].Key, "5h")
	}
	if got.Windows[1].Key != "weekly" {
		t.Errorf("got windows[1].Key = %q, want %q", got.Windows[1].Key, "weekly")
	}
}

func TestFetch_Error(t *testing.T) {
	tmp := t.TempDir()
	p := New()
	p.credsPath = filepath.Join(tmp, "no-creds.json")
	p.apiCachePath = filepath.Join(tmp, "no-cache.json")

	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() should not return error, got %v", err)
	}
	if string(got.Status) != "error" {
		t.Errorf("got Status = %q, want %q", got.Status, "error")
	}
	if got.Error == "" {
		t.Error("got Error empty, want non-empty")
	}
}

func TestID(t *testing.T) {
	p := New()
	got := p.ID()
	want := providerID
	if got != want {
		t.Errorf("got ID = %q, want %q", got, want)
	}
}

func TestCachePath(t *testing.T) {
	p := New()
	got := p.CachePath()
	if got == "" {
		t.Error("CachePath returned empty string")
	}
	if got[len(got)-len("stats-cache.json"):] != "stats-cache.json" {
		t.Errorf("CachePath should end with stats-cache.json, got %q", got)
	}
}
