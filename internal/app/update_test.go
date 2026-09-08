package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
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
	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(name, ".zip") {
			t.Fatalf("windows asset should be zip, got %q", name)
		}
	case "darwin":
		if !strings.HasSuffix(name, ".dmg") {
			t.Fatalf("darwin asset should be dmg, got %q", name)
		}
	}
}

func TestUnpackDesktopExePrefersDSHGo(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "dsh-go-windows-amd64.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	other, err := w.Create("helper.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Write([]byte("helper")); err != nil {
		t.Fatal(err)
	}
	want, err := w.Create("dsh-go.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := want.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	exe, err := unpackDesktopExe(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(exe) })
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Fatalf("extracted %q", got)
	}
}

func TestUnpackDesktopExeMissing(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	readme, err := w.Create("README.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readme.Write([]byte("no exe")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := unpackDesktopExe(zipPath); err == nil {
		t.Fatal("expected error for zip without exe")
	}
}

func TestDesktopUpdateScriptDeletesItselfQuietly(t *testing.T) {
	s := desktopUpdateScript(`C:\AppData\dsh-go\dsh-go.exe`, `C:\Temp\new.exe`)
	if !strings.Contains(s, "@echo off") {
		t.Fatal("update script must hide its own echo")
	}
	if !strings.Contains(s, `(goto) 2>nul & del "%~f0"`) {
		t.Fatal("self-delete must not leave a visible cmd after the batch is gone")
	}
	if strings.Contains(s, "\ndel \"%") || strings.Contains(s, "\r\ndel \"%") {
		t.Fatal("bare del of the running script opens a console with 找不到批处理文件")
	}
}

func TestAppBundlePath(t *testing.T) {
	exe := filepath.Join(string(filepath.Separator)+"Applications", "Foo.app", "Contents", "MacOS", "dsh-go")
	want := filepath.Join(string(filepath.Separator)+"Applications", "Foo.app")
	if got := appBundlePath(exe); got != want {
		t.Fatalf("bundle = %q, want %q", got, want)
	}
	if appBundlePath(filepath.Join(string(filepath.Separator)+"tmp", "dsh-go")) != "" {
		t.Fatal("bare binary should not look like a bundle")
	}
	dev := filepath.Join(string(filepath.Separator)+"tmp", "bin", "dsh-go.dev.app", "Contents", "MacOS", "dsh-go")
	if appBundlePath(dev) != "" {
		t.Fatal("dev bundle must not be replaced")
	}
	if appBundlePath("") != "" {
		t.Fatal("empty exe")
	}
}

func TestFindAppInDir(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "Deepseek Harness GO.app")
	if err := os.Mkdir(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/Applications", filepath.Join(dir, "Applications")); err != nil {
		t.Logf("skip Applications symlink: %v", err)
	}
	got, err := findAppInDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != app {
		t.Fatalf("got %q, want %q", got, app)
	}
	empty := t.TempDir()
	if _, err := findAppInDir(empty); err == nil {
		t.Fatal("expected error for dir without .app")
	}
	onlyLink := t.TempDir()
	if err := os.Symlink(app, filepath.Join(onlyLink, "Fake.app")); err != nil {
		t.Logf("skip Fake.app symlink: %v", err)
	} else if _, err := findAppInDir(onlyLink); err == nil {
		t.Fatal("symlink .app should be skipped")
	}
}

func TestDarwinUpdateScript(t *testing.T) {
	s := darwinUpdateScript()
	for _, want := range []string{
		"kill -0",
		"ditto \"$SRC\" \"$DEST\"",
		`mv "$BAK" "$DEST"`,
		`open "$DEST"`,
		`rm -f -- "$0"`,
		"com.apple.quarantine",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("script missing %q", want)
		}
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
