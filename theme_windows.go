//go:build windows

package main

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func systemDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false
	}
	return v == 0
}

func applyNativeChrome(win *application.WebviewWindow, dark bool) {
	hwnd := windows.HWND(uintptr(win.NativeWindow()))
	if hwnd == 0 {
		return
	}
	var immersive uint32
	if dark {
		immersive = 1
	}
	_ = windows.DwmSetWindowAttribute(hwnd, windows.DWMWA_USE_IMMERSIVE_DARK_MODE, unsafe.Pointer(&immersive), 4)

	caption := uint32(0x00FFFFFF)
	text := uint32(0x00111111)
	if dark {
		caption = 0x000F0706
		text = 0x00EEEEEE
	}
	_ = windows.DwmSetWindowAttribute(hwnd, windows.DWMWA_CAPTION_COLOR, unsafe.Pointer(&caption), 4)
	_ = windows.DwmSetWindowAttribute(hwnd, windows.DWMWA_TEXT_COLOR, unsafe.Pointer(&text), 4)
}
