package widgets

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:plasma all:dms
var FS embed.FS

// ExtractTo copies the subtree at src within FS into dst on the real filesystem.
// dst is created if it does not exist.
func ExtractTo(src, dst string) error {
	for range 16 {
		link, err := os.Readlink(dst)
		if err != nil {
			break
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(dst), link)
		}
		dst = link
	}

	sub, err := fs.Sub(FS, src)
	if err != nil {
		return fmt.Errorf("open embedded %s: %w", src, err)
	}
	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
		}
		f, err := sub.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer f.Close()
		out, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("create %s: %w", target, err)
		}
		defer out.Close()
		if _, err := io.Copy(out, f); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
		return nil
	})
}
