//go:build unix

package app

import (
	"os"
	"path/filepath"
)

func persistShimPATHImpl(dir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	src := filepath.Join(dir, "dsh")
	if _, err := os.Stat(src); err != nil {
		return err
	}
	destDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(destDir, "dsh")
	if sameResolved(src, dest) {
		return nil
	}
	if fi, err := os.Lstat(dest); err == nil {
		if !fi.Mode().IsRegular() && fi.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		if !hasShimMarker(dest) {
			return nil
		}
	}
	_ = os.Remove(dest)
	if err := os.Symlink(src, dest); err == nil {
		return nil
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, body, 0o755)
}

func sameResolved(a, b string) bool {
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		return false
	}
	return ra == rb
}
