//go:build darwin

package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func installDesktopPackage(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	if exe, err := os.Executable(); err == nil {
		if bundle := appBundlePath(exe); bundleReplaceable(bundle) {
			extracted, err := extractAppFromDMG(abs)
			if err != nil {
				log.Printf("client install: extract dmg: %v; opening image", err)
			} else if err := startDarwinReplace(bundle, extracted); err != nil {
				log.Printf("client install: replace script: %v; opening image", err)
				_ = os.RemoveAll(filepath.Dir(extracted))
			} else {
				return true, nil
			}
		}
	}
	return false, openDesktopDMG(abs)
}

func bundleReplaceable(bundle string) bool {
	if bundle == "" {
		return false
	}
	return unix.Access(filepath.Dir(bundle), unix.W_OK) == nil
}

func extractAppFromDMG(dmg string) (string, error) {
	mnt, err := os.MkdirTemp("", "dsh-go-dmg-*")
	if err != nil {
		return "", err
	}
	detach := func() {
		_ = exec.Command("hdiutil", "detach", mnt, "-quiet", "-force").Run()
		_ = os.RemoveAll(mnt)
	}

	cmd := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-noverify", "-mountpoint", mnt, dmg)
	if out, err := cmd.CombinedOutput(); err != nil {
		detach()
		return "", fmt.Errorf("attach dmg: %w: %s", err, strings.TrimSpace(string(out)))
	}
	src, err := findAppInDir(mnt)
	if err != nil {
		detach()
		return "", err
	}
	destDir, err := os.MkdirTemp("", "dsh-go-update-*")
	if err != nil {
		detach()
		return "", err
	}
	dest := filepath.Join(destDir, filepath.Base(src))
	if out, err := exec.Command("ditto", src, dest).CombinedOutput(); err != nil {
		detach()
		_ = os.RemoveAll(destDir)
		return "", fmt.Errorf("ditto: %w: %s", err, strings.TrimSpace(string(out)))
	}
	detach()
	return dest, nil
}

func startDarwinReplace(dest, src string) error {
	script := filepath.Join(os.TempDir(), "dsh-go-update.sh")
	if err := os.WriteFile(script, []byte(darwinUpdateScript()), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("/bin/bash", script, strconv.Itoa(os.Getpid()), dest, src)
	applyProcAttr(cmd)
	return cmd.Start()
}

func openDesktopDMG(abs string) error {
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
