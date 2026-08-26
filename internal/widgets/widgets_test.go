package widgets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractToFollowsDestinationSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "pakai")
	intermediate := filepath.Join(root, "intermediate")
	if err := os.Symlink(target, intermediate); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(intermediate, link); err != nil {
		t.Fatal(err)
	}

	if err := ExtractTo("dms", link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "PakAIWidget.qml")); err != nil {
		t.Fatalf("widget was not extracted through symlink: %v", err)
	}
}
