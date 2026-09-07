package main

import (
	"strings"
	"testing"
)

func TestDesktopReleaseMirrors(t *testing.T) {
	got := desktopReleaseMirrors("v0.2.0")
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if !strings.Contains(got[0], "cnb.cool") || !strings.HasSuffix(got[0], "/v0.2.0") {
		t.Fatalf("cnb = %s", got[0])
	}
	if !strings.Contains(got[1], "github.com") || !strings.HasSuffix(got[1], "/v0.2.0") {
		t.Fatalf("github = %s", got[1])
	}
}

func TestDesktopAssetName(t *testing.T) {
	name := desktopAssetName()
	if name == "" || !strings.HasPrefix(name, "dsh-go-") {
		t.Fatalf("asset = %q", name)
	}
}

func TestMirrorName(t *testing.T) {
	if got := mirrorName("https://cnb.cool/nobu121/dsh-go/-/releases/download/v1"); got != "CNB" {
		t.Fatalf("cnb = %s", got)
	}
	if got := mirrorName("https://github.com/nobu121/dsh-go/releases/download/v1"); got != "GitHub" {
		t.Fatalf("github = %s", got)
	}
}
