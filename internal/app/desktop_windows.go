//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installDesktopPackage(path string) error {
	if strings.HasSuffix(strings.ToLower(path), ".dmg") {
		return fmt.Errorf("this Windows build cannot install a macOS image")
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.Abs(current)
	if err != nil {
		return err
	}
	src := path
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		extracted, err := unpackDesktopExe(path)
		if err != nil {
			return err
		}
		src = extracted
	}
	src, err = filepath.Abs(src)
	if err != nil {
		return err
	}
	script := filepath.Join(os.TempDir(), "dsh-go-update.cmd")
	if err := os.WriteFile(script, []byte(desktopUpdateScript(current, src)), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", script)
	cmd.SysProcAttr = hiddenProcAttr(0)
	return cmd.Start()
}

func desktopInstallRestarts() bool { return true }
