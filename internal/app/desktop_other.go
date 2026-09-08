//go:build !windows && !darwin

package app

import "fmt"

func installDesktopPackage(path string) (bool, error) {
	return false, fmt.Errorf("this platform cannot install %s", path)
}
