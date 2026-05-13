package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Found(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, ".claude", "stats-cache.json")
	os.MkdirAll(filepath.Dir(claudePath), 0755)
	os.WriteFile(claudePath, []byte("{}"), 0644)

	opencodePath := filepath.Join(dir, ".local", "share", "opencode", "opencode.db")
	os.MkdirAll(filepath.Dir(opencodePath), 0755)
	os.WriteFile(opencodePath, []byte(""), 0644)

	results := Detect(func() string { return dir })
	if len(results) != 2 {
		t.Fatalf("got %d detections, want 2", len(results))
	}
	if !results[0].Found {
		t.Errorf("claude should be found")
	}
	if !results[1].Found {
		t.Errorf("opencode should be found")
	}
}

func TestDetect_NoneFound(t *testing.T) {
	dir := t.TempDir()
	results := Detect(func() string { return dir })

	for _, r := range results {
		if r.Found {
			t.Errorf("provider %s should not be found", r.ProviderID)
		}
	}
}

func TestDetect_ClaudeOnly(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, ".claude", "stats-cache.json")
	os.MkdirAll(filepath.Dir(claudePath), 0755)
	os.WriteFile(claudePath, []byte("{}"), 0644)

	results := Detect(func() string { return dir })
	if len(results) == 0 {
		t.Fatal("expected at least claude detection")
	}
	if !results[0].Found {
		t.Errorf("claude should be found")
	}
}

func TestDetection_Fields(t *testing.T) {
	d := Detection{
		ProviderID: "claude",
		Path:       "/home/user/.claude/stats-cache.json",
		Found:      true,
	}
	if d.ProviderID != "claude" {
		t.Errorf("got ProviderID=%q, want claude", d.ProviderID)
	}
	if !d.Found {
		t.Error("Found should be true")
	}
}
