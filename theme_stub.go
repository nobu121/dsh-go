//go:build !darwin && !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func systemDark() bool { return false }

func applyNativeChrome(*application.WebviewWindow, bool) {}

func pollWindowTheme(*application.WebviewWindow) {}
