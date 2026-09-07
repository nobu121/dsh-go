//go:build windows

package main

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
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
	if win == nil {
		return
	}
	hwnd := w32.HWND(uintptr(win.NativeWindow()))
	if hwnd == 0 {
		return
	}
	raw := uintptr(hwnd)

	if w32.RefreshImmersiveColorPolicyState != nil {
		w32.RefreshImmersiveColorPolicyState()
	}
	if w32.AllowDarkModeForWindow != nil {
		w32.AllowDarkModeForWindow(hwnd, true)
	}

	// Both attribute numbers: pre-20H1 hosts only honor 19.
	var immersive uint32
	if dark {
		immersive = 1
	}
	_ = windows.DwmSetWindowAttribute(windows.HWND(raw), 19, unsafe.Pointer(&immersive), 4)
	_ = windows.DwmSetWindowAttribute(windows.HWND(raw), windows.DWMWA_USE_IMMERSIVE_DARK_MODE, unsafe.Pointer(&immersive), 4)

	w32.SetTheme(raw, dark)
	var useDark int32
	if dark {
		useDark = 1
	}
	data := w32.WINDOWCOMPOSITIONATTRIBDATA{
		Attrib: w32.WCA_USEDARKMODECOLORS,
		PvData: unsafe.Pointer(&useDark),
		CbData: unsafe.Sizeof(useDark),
	}
	w32.SetWindowCompositionAttribute(hwnd, &data)

	caption, text := windowsCaptionColors(dark)
	w32.SetTitleBarColour(raw, caption)
	w32.SetTitleTextColour(raw, text)
	w32.SetBorderColour(raw, caption)

	if w32.FlushMenuThemes != nil {
		w32.FlushMenuThemes()
	}
	active := w32.GetForegroundWindow() == hwnd
	w32.SendMessage(hwnd, w32.WM_NCACTIVATE, 0, 0)
	if active {
		w32.SendMessage(hwnd, w32.WM_NCACTIVATE, 1, 0)
	}
	w32.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE|w32.SWP_FRAMECHANGED)
	w32.RedrawWindow(hwnd, nil, 0, w32.RDW_INVALIDATE|w32.RDW_FRAME|w32.RDW_UPDATENOW)
	w32.InvalidateRect(hwnd, nil, true)
}
