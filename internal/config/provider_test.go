package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProviderEnabled(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "pakai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[provider.openai]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))

	if IsProviderEnabled("openai") {
		t.Fatal("configured disabled provider should be disabled")
	}
	if !IsProviderEnabled("claude") {
		t.Fatal("unconfigured provider should remain enabled")
	}
}
