//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestHiddenProcAttrCreatesNoWindow(t *testing.T) {
	attr := hiddenProcAttr(0)
	if attr == nil || !attr.HideWindow {
		t.Fatal("HideWindow")
	}
	if attr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("CREATE_NO_WINDOW")
	}
	attr = hiddenProcAttr(windows.CREATE_NEW_PROCESS_GROUP)
	if attr.CreationFlags&windows.CREATE_NO_WINDOW == 0 || attr.CreationFlags&windows.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatalf("flags = %#x", attr.CreationFlags)
	}
}
