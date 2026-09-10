//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installDesktopPackage(path string) (bool, error) {
	if strings.HasSuffix(strings.ToLower(path), ".dmg") {
		return false, fmt.Errorf("this Windows build cannot install a macOS image")
	}
	current, err := os.Executable()
	if err != nil {
		return false, err
	}
	current, err = filepath.Abs(current)
	if err != nil {
		return false, err
	}
	src := path
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		extracted, err := unpackDesktopExe(path)
		if err != nil {
			return false, err
		}
		src = extracted
	}
	src, err = filepath.Abs(src)
	if err != nil {
		return false, err
	}
	script := filepath.Join(os.TempDir(), "dsh-go-update.cmd")
	if err := os.WriteFile(script, []byte(desktopUpdateScript(current, src)), 0o644); err != nil {
		return false, err
	}
	// Detach so taskkill /IM dsh-go.exe in the script cannot kill the updater.
	cmd := exec.Command("cmd", "/C", "start", "", "/B", script)
	cmd.SysProcAttr = hiddenProcAttr(0)
	return true, cmd.Start()
}
