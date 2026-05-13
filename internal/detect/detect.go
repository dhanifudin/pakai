package detect

import (
	"os"
	"path/filepath"
)

type Detection struct {
	ProviderID string
	Path       string
	Found      bool
}

type homeFunc func() string

func Detect(home homeFunc) []Detection {
	if home == nil {
		home = func() string {
			h, _ := os.UserHomeDir()
			return h
		}
	}

	h := home()
	return []Detection{
		{
			ProviderID: "claude",
			Path:       filepath.Join(h, ".claude", "stats-cache.json"),
			Found:      fileExists(filepath.Join(h, ".claude", "stats-cache.json")),
		},
		{
			ProviderID: "opencode",
			Path:       filepath.Join(h, ".local", "share", "opencode", "opencode.db"),
			Found:      fileExists(filepath.Join(h, ".local", "share", "opencode", "opencode.db")),
		},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
