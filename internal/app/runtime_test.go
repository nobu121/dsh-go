package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
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
	// Deliberately unrelated to any version baked into the shell: the channel
	// publishes whatever dsh is current and the zip describes itself.
	const channelVer = "9.9.9"
	zipPath := filepath.Join(stage, runtimeAssetName())
	writeRuntimeZip(t, zipPath, channelVer)
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	t.Setenv("DSH_RUNTIME_DIR", cache)
	if err := fetchCachedRuntime(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !runtimeLooksValid(cache) {
		t.Fatal("cache missing node or bin.js")
	}
	if got := readRuntimeVersion(cache); got != channelVer {
		t.Fatalf("VERSION = %q, want %q", got, channelVer)
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
	asset := runtimeAssetName()
	zipPath := filepath.Join(stage, asset)
	writeRuntimeZip(t, zipPath, "9.9.9")
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

func TestThrottlePrepDownload(t *testing.T) {
	var n atomic.Int32
	fn := throttlePrep(func(PrepProgress) { n.Add(1) }, 40*time.Millisecond)
	fn(PrepProgress{Stage: "download", Bytes: 1, Total: 100})
	fn(PrepProgress{Stage: "download", Bytes: 2, Total: 100})
	if n.Load() != 1 {
		t.Fatalf("dropped intermediate = %d", n.Load())
	}
	time.Sleep(50 * time.Millisecond)
	fn(PrepProgress{Stage: "download", Bytes: 3, Total: 100})
	if n.Load() != 2 {
		t.Fatalf("after interval = %d", n.Load())
	}
	fn(PrepProgress{Stage: "download", Bytes: 100, Total: 100})
	if n.Load() != 3 {
		t.Fatalf("final flush = %d", n.Load())
	}
	fn(PrepProgress{Stage: "unpack"})
	if n.Load() != 4 {
		t.Fatalf("non-download = %d", n.Load())
	}
}

func TestUnzipRuntimeReportsProgress(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "rt.zip")
	writeRuntimeZip(t, zipPath, "1.0.0")
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	var lastN, lastTot int64
	var reports int
	if err := unzipRuntime(zipPath, dest, func(n, tot int64) {
		reports++
		if tot <= 0 {
			t.Fatalf("total = %d", tot)
		}
		if n < lastN {
			t.Fatalf("progress went backwards: %d -> %d", lastN, n)
		}
		lastN, lastTot = n, tot
	}); err != nil {
		t.Fatal(err)
	}
	if reports == 0 || lastN != lastTot {
		t.Fatalf("reports=%d n=%d total=%d", reports, lastN, lastTot)
	}
	if !runtimeLooksValid(dest) {
		t.Fatal("unzipped tree missing node or bin.js")
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
	RuntimeBaseURL = "https://cnb.cool/nobu121/dsh-go/-/releases/download/" + runtimeChannelTag
	UpdateRepo = "nobu121/dsh-go"
	got := runtimeBaseURLs()
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got[0] != RuntimeBaseURL {
		t.Fatalf("primary = %s", got[0])
	}
	wantGH := "https://github.com/nobu121/dsh-go/releases/download/" + runtimeChannelTag
	if got[1] != wantGH {
		t.Fatalf("fallback = %s, want %s", got[1], wantGH)
	}
}

// The runtime channel must not be version-scoped; that coupling is what forced
// a shell release for every upstream dsh bump.
func TestRuntimeBaseURLsAreVersionIndependent(t *testing.T) {
	t.Setenv("DSH_RUNTIME_BASE_URL", "")
	oldBase, oldRepo, oldVer := RuntimeBaseURL, UpdateRepo, Version
	t.Cleanup(func() {
		RuntimeBaseURL, UpdateRepo, Version = oldBase, oldRepo, oldVer
	})
	RuntimeBaseURL = ""
	UpdateRepo = "nobu121/dsh-go"
	Version = "1.2.3"
	before := runtimeBaseURLs()
	Version = "4.5.6"
	if after := runtimeBaseURLs(); len(after) != 2 || len(before) != 2 || after[0] != before[0] || after[1] != before[1] {
		t.Fatalf("channel moved with client version: %#v vs %#v", before, after)
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
