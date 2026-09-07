//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func installDesktopPackage(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if home, err := os.UserHomeDir(); err == nil {
		dest := filepath.Join(home, "Downloads", filepath.Base(abs))
		if dest != abs {
			if data, err := os.ReadFile(abs); err == nil {
				if err := os.WriteFile(dest, data, 0o644); err == nil {
					abs = dest
				}
			}
		}
	}
	return exec.Command("open", abs).Start()
}

func desktopInstallRestarts() bool { return false }
