package pi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchAllAggregatesPiSessionProviders(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old := now.AddDate(0, -1, 0)
	lines := `{"type":"message","timestamp":"` + now.Format(time.RFC3339Nano) + `","message":{"role":"assistant","provider":"opencode","usage":{"totalTokens":120}}}
` +
		`{"type":"message","timestamp":"` + now.Format(time.RFC3339Nano) + `","message":{"role":"assistant","provider":"opencode","usage":{"totalTokens":30}}}
` +
		`{"type":"message","timestamp":"` + now.Format(time.RFC3339Nano) + `","message":{"role":"assistant","provider":"kosyayuk","usage":{"totalTokens":50}}}
` +
		`{"type":"message","timestamp":"` + old.Format(time.RFC3339Nano) + `","message":{"role":"assistant","provider":"opencode","usage":{"totalTokens":999}}}
` +
		`{"type":"message","timestamp":"` + now.Format(time.RFC3339Nano) + `","message":{"role":"user","provider":"opencode"}}
`
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(lines), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := NewWithPath(dir).FetchAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d providers, want 2", len(got))
	}

	usage := map[string]float64{}
	for _, item := range got {
		usage[item.Provider] = item.Used
	}
	if usage["pi/opencode"] != 150 {
		t.Fatalf("opencode tokens = %.0f, want 150", usage["pi/opencode"])
	}
	if usage["pi/kosyayuk"] != 50 {
		t.Fatalf("kosyayuk tokens = %.0f, want 50", usage["pi/kosyayuk"])
	}
}

func TestFetchAllHidesProvidersWithoutCurrentMonthUsage(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, -1, 0)
	line := `{"type":"message","timestamp":"` + old.Format(time.RFC3339Nano) + `","message":{"role":"assistant","provider":"opencode","usage":{"totalTokens":999}}}
`
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(line), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := NewWithPath(dir).FetchAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d providers, want none", len(got))
	}
}
