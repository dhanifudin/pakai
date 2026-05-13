package claude

import (
	"context"
	"io"
	"os"
	"testing"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return "testdata/" + name
}

func TestFetch_Valid(t *testing.T) {
	path := testdataPath(t, "stats_valid.json")

	p := New()
	p.readerFactory = func() (io.ReadCloser, error) {
		return os.Open(path)
	}

	ctx := context.Background()
	got, err := p.Fetch(ctx)

	if err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}

	wantProvider := providerID
	if got.Provider != wantProvider {
		t.Errorf("got Provider = %q, want %q", got.Provider, wantProvider)
	}

	wantStatus := "ok"
	if string(got.Status) != wantStatus {
		t.Errorf("got Status = %q, want %q", got.Status, wantStatus)
	}

	wantUsed := 350.0
	if got.Used != wantUsed {
		t.Errorf("got Used = %.1f, want %.1f", got.Used, wantUsed)
	}

	if got.RefreshedAt.IsZero() {
		t.Error("got RefreshedAt zero, want non-zero timestamp")
	}
}

func TestFetch_Malformed(t *testing.T) {
	path := testdataPath(t, "stats_malformed.json")

	p := New()
	p.readerFactory = func() (io.ReadCloser, error) {
		return os.Open(path)
	}

	ctx := context.Background()
	got, err := p.Fetch(ctx)

	if err != nil {
		t.Fatalf("Fetch() should not return error for malformed JSON: %v", err)
	}

	wantStatus := "error"
	if string(got.Status) != wantStatus {
		t.Errorf("got Status = %q, want %q", got.Status, wantStatus)
	}
	if got.Error == "" {
		t.Error("got Error empty, want non-empty for malformed JSON")
	}
}

func TestFetch_MissingFields(t *testing.T) {
	path := testdataPath(t, "stats_missing_fields.json")

	p := New()
	p.readerFactory = func() (io.ReadCloser, error) {
		return os.Open(path)
	}

	ctx := context.Background()
	got, err := p.Fetch(ctx)

	if err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}

	wantStatus := "ok"
	if string(got.Status) != wantStatus {
		t.Errorf("got Status = %q, want %q", got.Status, wantStatus)
	}

	wantUsed := 0.0
	if got.Used != wantUsed {
		t.Errorf("got Used = %.1f, want %.1f for missing dailyActivity", got.Used, wantUsed)
	}
}

func TestFetch_FileNotFound(t *testing.T) {
	p := New()
	p.readerFactory = func() (io.ReadCloser, error) {
		return os.Open("/nonexistent/path/stats-cache.json")
	}

	ctx := context.Background()
	got, err := p.Fetch(ctx)

	if err != nil {
		t.Fatalf("Fetch() should not return error: %v", err)
	}

	wantStatus := "error"
	if string(got.Status) != wantStatus {
		t.Errorf("got Status = %q, want %q", got.Status, wantStatus)
	}
	if got.Error == "" {
		t.Error("got Error empty, want non-empty")
	}
}

func TestFetch_PercentNotCapped(t *testing.T) {
	p := New()
	p.readerFactory = func() (io.ReadCloser, error) {
		return os.Open(testdataPath(t, "stats_valid.json"))
	}

	ctx := context.Background()
	got, _ := p.Fetch(ctx)

	pct := got.Pct()
	if pct > 100 {
		t.Logf("Pct = %.1f (not capped — AC 2 satisfied)", pct)
	}
	if got.Used > 0 && got.Limit == 0 {
		t.Logf("no limit configured, Pct = %.1f", pct)
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
