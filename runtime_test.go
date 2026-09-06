package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func writeRuntimeZip(t *testing.T, zipPath, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	add := func(name, body string, mode os.FileMode) {
		t.Helper()
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(mode)
		fw, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(fw, body); err != nil {
			t.Fatal(err)
		}
	}
	add(bundledNodeName(), "#!/bin/sh\n", 0o755)
	add("VERSION", version+"\n", 0o644)
	add("node_modules/@deepseek-ai/dsh/lib/bin.js", "", 0o644)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func fileURL(path string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	return u.String()
}

func TestFetchCachedRuntimeFileURL(t *testing.T) {
	isolateLookPath(t)
	stage := t.TempDir()
	cache := filepath.Join(t.TempDir(), "cache")
	pin := currentVersion()
	zipPath := filepath.Join(stage, runtimeAssetName())
	writeRuntimeZip(t, zipPath, pin)
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	t.Setenv("DSH_RUNTIME_DIR", cache)
	if err := fetchCachedRuntime(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !runtimeLooksValid(cache) {
		t.Fatal("cache missing node or bin.js")
	}
	if got := readRuntimeVersion(cache); got != pin {
		t.Fatalf("VERSION = %q, want %q", got, pin)
	}
	argv, ok := cacheRuntimeCommand()
	if !ok {
		t.Fatal("cacheRuntimeCommand should hit")
	}
	if argv[0] != filepath.Join(cache, bundledNodeName()) {
		t.Fatalf("node = %s", argv[0])
	}
}

func TestFetchCachedRuntimeChecksum(t *testing.T) {
	isolateLookPath(t)
	stage := t.TempDir()
	cache := filepath.Join(t.TempDir(), "cache")
	pin := currentVersion()
	asset := runtimeAssetName()
	zipPath := filepath.Join(stage, asset)
	writeRuntimeZip(t, zipPath, pin)
	sum, err := fileSHA256(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "SHA256SUMS"), []byte(sum+"  "+asset+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	t.Setenv("DSH_RUNTIME_DIR", cache)
	if err := fetchCachedRuntime(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestChecksumForAsset(t *testing.T) {
	sums := "abc  dsh-runtime-darwin-arm64.zip\ndef *other.zip\n"
	got, ok := checksumForAsset(sums, "dsh-runtime-darwin-arm64.zip")
	if !ok || got != "abc" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRuntimeAssetName(t *testing.T) {
	name := runtimeAssetName()
	if name == "" || filepath.Ext(name) != ".zip" {
		t.Fatalf("asset = %q", name)
	}
}

func TestRuntimeBaseURLsEnvOverrides(t *testing.T) {
	t.Setenv("DSH_RUNTIME_BASE_URL", "https://example.test/runtime/")
	oldBase, oldRepo := RuntimeBaseURL, UpdateRepo
	t.Cleanup(func() {
		RuntimeBaseURL = oldBase
		UpdateRepo = oldRepo
	})
	RuntimeBaseURL = "https://cnb.cool/example/-/releases/download/v1"
	UpdateRepo = "acme/dsh-go"
	got := runtimeBaseURLs()
	if len(got) != 1 || got[0] != "https://example.test/runtime" {
		t.Fatalf("got %#v", got)
	}
}

func TestRuntimeBaseURLsFallbackGitHub(t *testing.T) {
	t.Setenv("DSH_RUNTIME_BASE_URL", "")
	oldBase, oldRepo := RuntimeBaseURL, UpdateRepo
	t.Cleanup(func() {
		RuntimeBaseURL = oldBase
		UpdateRepo = oldRepo
	})
	RuntimeBaseURL = "https://cnb.cool/nobu121/dsh-go/-/releases/download/v" + currentVersion()
	UpdateRepo = "nobu121/dsh-go"
	got := runtimeBaseURLs()
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got[0] != RuntimeBaseURL {
		t.Fatalf("primary = %s", got[0])
	}
	wantGH := "https://github.com/nobu121/dsh-go/releases/download/v" + currentVersion()
	if got[1] != wantGH {
		t.Fatalf("fallback = %s, want %s", got[1], wantGH)
	}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
