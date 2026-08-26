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
		func() Detection {
			path, found := firstExisting(filepath.Join(h, ".pi", "agent", "auth.json"), filepath.Join(h, ".codex", "auth.json"))
			return Detection{ProviderID: "openai", Path: path, Found: found}
		}(),
		func() Detection {
			path, found := firstExisting(filepath.Join(h, ".pi", "agent", "auth.json"), filepath.Join(h, ".local", "share", "opencode", "auth.json"))
			return Detection{ProviderID: "opencode-go", Path: path, Found: found}
		}(),
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstExisting(paths ...string) (string, bool) {
	for _, path := range paths {
		if fileExists(path) {
			return path, true
		}
	}
	return paths[0], false
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
