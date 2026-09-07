//go:build !darwin && !windows

package app

import "github.com/wailsapp/wails/v3/pkg/application"

func systemDark() bool { return false }

func applyNativeChrome(*application.WebviewWindow, bool) {}
