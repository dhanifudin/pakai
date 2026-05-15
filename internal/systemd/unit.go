package systemd

import (
	"fmt"
	"os"
	"path/filepath"
)

type homeFunc func() string

func WriteUnit(execPath string, home homeFunc) error {
	if home == nil {
		home = func() string {
			h, _ := os.UserHomeDir()
			return h
		}
	}

	unit := fmt.Sprintf(`[Unit]
Description=PakAI usage tracker daemon
After=default.target

[Service]
ExecStart=%s daemon start --foreground
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, execPath)

	unitDir := filepath.Join(home(), ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd unit dir: %w", err)
	}

	unitFile := filepath.Join(unitDir, "pakai.service")
	if err := os.WriteFile(unitFile, []byte(unit), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}

	return nil
}

func UnitPath(home homeFunc) string {
	if home == nil {
		home = func() string {
			h, _ := os.UserHomeDir()
			return h
		}
	}
	return filepath.Join(home(), ".config", "systemd", "user", "pakai.service")
}
