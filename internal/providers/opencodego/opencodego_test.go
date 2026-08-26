package opencodego

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchUsageWithPiKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"opencode-go":{"type":"api_key","key":"test-key"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"usage":{"rolling":{"percent":10,"resetsAt":"2026-08-27T10:00:00Z"},"weekly":{"percent":20,"resetsAt":"2026-08-30T10:00:00Z"},"monthly":{"percent":30,"resetsAt":"2026-09-01T00:00:00Z"}}}`))
	}))
	defer server.Close()

	p := newWithPath(path)
	p.usageURL = server.URL
	usage, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.Windows) != 3 {
		t.Fatalf("windows = %d, want 3", len(usage.Windows))
	}
	if usage.Windows[0].Used != 10 || usage.Windows[1].Used != 20 || usage.Windows[2].Used != 30 {
		t.Fatalf("windows = %+v", usage.Windows)
	}
}

func TestReadAPIKeyFromOpenCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"opencode":{"type":"api","key":"test-key"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	p := newWithPath(path)
	p.source = "opencode"
	key, err := p.readAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if key != "test-key" {
		t.Fatalf("key = %q", key)
	}
}
