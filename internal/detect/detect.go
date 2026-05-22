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
		func() Detection {
			dir := filepath.Join(h, ".local", "share", "opencode")
			path, found := resolveOpenCodeDB(dir)
			return Detection{ProviderID: "opencode", Path: path, Found: found}
		}(),
		{
			ProviderID: "openai",
			Path:       filepath.Join(h, ".codex", "auth.json"),
			Found:      fileExists(filepath.Join(h, ".codex", "auth.json")),
		},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// opencodeDBNames lists candidate filenames in preference order, matching the
// opencode provider package. Kept in sync manually — both must agree.
var opencodeDBNames = []string{"opencode-stable.db", "opencode.db"}

// resolveOpenCodeDB returns the path and found status for the first existing
// opencode db file under dir, falling back to the first candidate if none exist.
func resolveOpenCodeDB(dir string) (string, bool) {
	for _, name := range opencodeDBNames {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p, true
		}
	}
	return filepath.Join(dir, opencodeDBNames[0]), false
}
