//go:build windows

package main

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
	src, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	script := filepath.Join(os.TempDir(), "dsh-go-update.cmd")
	body := fmt.Sprintf(""+
		"@echo off\r\n"+
		"set \"CUR=%s\"\r\n"+
		"set \"NEW=%s\"\r\n"+
		"for /l %%%%i in (1,1,30) do (\r\n"+
		"  move /y \"%%CUR%%\" \"%%CUR%%.old\" >nul 2>nul && goto replaced\r\n"+
		"  ping -n 2 127.0.0.1 >nul\r\n"+
		")\r\n"+
		"exit /b 1\r\n"+
		":replaced\r\n"+
		"move /y \"%%NEW%%\" \"%%CUR%%\" >nul\r\n"+
		"start \"\" \"%%CUR%%\"\r\n"+
		"del \"%%~f0\"\r\n",
		current, src)
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", "start", "", script)
	return cmd.Start()
}

func desktopInstallRestarts() bool { return true }
