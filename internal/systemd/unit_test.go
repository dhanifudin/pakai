package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteUnit(t *testing.T) {
	dir := t.TempDir()
	home := dir
	t.Setenv("HOME", home)

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	os.MkdirAll(unitDir, 0755)

	execPath := "/usr/local/bin/pakai"
	err := WriteUnit(execPath, func() string { return home })
	if err != nil {
		t.Fatalf("WriteUnit error: %v", err)
	}

	unitFile := filepath.Join(unitDir, "pakai.service")
	data, err := os.ReadFile(unitFile)
	if err != nil {
		t.Fatalf("unit file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Description=PakAI") {
		t.Error("unit missing Description")
	}
	if !strings.Contains(content, "ExecStart="+execPath) {
		t.Errorf("unit missing ExecStart with %q", execPath)
	}
	if !strings.Contains(content, "Restart=on-failure") {
		t.Error("unit missing Restart=on-failure")
	}
	if !strings.Contains(content, "RestartSec=5") {
		t.Error("unit missing RestartSec=5")
	}
	if !strings.Contains(content, "--foreground") {
		t.Error("unit missing --foreground flag")
	}
}

func TestWriteUnit_Overwrite(t *testing.T) {
	dir := t.TempDir()
	home := dir
	t.Setenv("HOME", home)

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	os.MkdirAll(unitDir, 0755)

	unitFile := filepath.Join(unitDir, "pakai.service")
	os.WriteFile(unitFile, []byte("old content"), 0644)

	err := WriteUnit("/bin/pakai", func() string { return home })
	if err != nil {
		t.Fatalf("WriteUnit error: %v", err)
	}

	data, _ := os.ReadFile(unitFile)
	if strings.Contains(string(data), "old content") {
		t.Error("unit file was not overwritten")
	}
	if !strings.Contains(string(data), "ExecStart") {
		t.Error("unit file should contain new content after overwrite")
	}
}

func TestWriteUnit_AutoCreateDir(t *testing.T) {
	dir := t.TempDir()
	home := dir
	t.Setenv("HOME", home)

	err := WriteUnit("/bin/pakai", func() string { return home })
	if err != nil {
		t.Fatalf("WriteUnit error: %v", err)
	}

	unitFile := filepath.Join(home, ".config", "systemd", "user", "pakai.service")
	if _, err := os.Stat(unitFile); os.IsNotExist(err) {
		t.Error("unit file was not created when directory didn't exist")
	}
}
