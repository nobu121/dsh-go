package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWebviewUserDataPath(t *testing.T) {
	got := webviewUserDataPath()
	if filepath.Base(got) != "webview" {
		t.Fatalf("got %q", got)
	}
	if filepath.Dir(got) != dshGoDir() {
		t.Fatalf("dir %q", got)
	}
}

func TestPurgeWebViewCookies(t *testing.T) {
	root := t.TempDir()
	network := filepath.Join(root, "EBWebView", "Default", "Network")
	if err := os.MkdirAll(network, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(network, "Local Storage")
	if err := os.WriteFile(filepath.Join(network, "Cookies"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(network, "Cookies-journal"), []byte("j"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	purgeWebViewCookies(root)
	if _, err := os.Stat(filepath.Join(network, "Cookies")); !os.IsNotExist(err) {
		t.Fatal("cookies")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("must keep other files")
	}
}
