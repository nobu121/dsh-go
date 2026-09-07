//go:build !windows && !darwin

package app

import "fmt"

func installDesktopPackage(path string) error {
	return fmt.Errorf("this platform cannot install %s", path)
}

func desktopInstallRestarts() bool { return false }
