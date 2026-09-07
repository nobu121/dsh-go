//go:build windows

package app

import (
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func persistShimPATHImpl(dir string) error {
	return mutateUserPath(func(path string) (string, bool) {
		return appendUserPATH(path, dir)
	})
}

func removeShimPATHImpl(dir string) error {
	return mutateUserPath(func(path string) (string, bool) {
		return removeUserPATH(path, dir)
	})
}

func mutateUserPath(fn func(string) (string, bool)) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	path, typ, err := k.GetStringValue("Path")
	if err == registry.ErrNotExist {
		path, typ = "", registry.EXPAND_SZ
	} else if err != nil {
		return err
	}
	next, changed := fn(path)
	if !changed {
		return nil
	}
	if typ == registry.EXPAND_SZ {
		err = k.SetExpandStringValue("Path", next)
	} else {
		err = k.SetStringValue("Path", next)
	}
	if err != nil {
		return err
	}
	broadcastEnvironment()
	return nil
}

func broadcastEnvironment() {
	env, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	send := user32.NewProc("SendMessageTimeoutW")
	_, _, _ = send.Call(
		uintptr(0xffff), // HWND_BROADCAST
		uintptr(0x001A), // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(env)),
		uintptr(0x0002), // SMTO_ABORTIFHUNG
		5000,
		0,
	)
}
