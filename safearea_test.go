package main

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestHarnessSafeAreaCSS(t *testing.T) {
	if !strings.Contains(harnessSafeAreaCSS, `_logoRow`) {
		t.Fatal("must target the sidebar brand row")
	}
	if !strings.Contains(harnessSafeAreaCSS, strconv.Itoa(harnessSafeTopPX)+"px") {
		t.Fatalf("must use %dpx safe top", harnessSafeTopPX)
	}
	css, js := harnessWindowSafeArea()
	if runtime.GOOS == "darwin" {
		if css == "" || js == "" {
			t.Fatal("macOS must inject the safe area")
		}
		return
	}
	if css != "" || js != "" {
		t.Fatal("non-macOS must not inject the safe area")
	}
}
