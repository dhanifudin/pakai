package config

import "testing"

func TestProviderSource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SetKey("provider.openai.source", "codex"); err != nil {
		t.Fatal(err)
	}
	if got := GetProviderSource("openai"); got != "codex" {
		t.Fatalf("source = %q, want codex", got)
	}
	if got, err := GetKey("provider.openai.source"); err != nil || got != "codex" {
		t.Fatalf("GetKey = %q, %v", got, err)
	}
}
