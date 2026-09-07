package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func isolateLookPath(t *testing.T) {
	t.Helper()
	prev := lookPath
	prevDirs := extraBinDirsFn
	lookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	extraBinDirsFn = func() []string { return nil }
	t.Cleanup(func() {
		lookPath = prev
		extraBinDirsFn = prevDirs
	})
}

func writeRuntimeTree(t *testing.T, root, version string) string {
	t.Helper()
	binDir := filepath.Join(root, "node_modules", "@deepseek-ai", "dsh", "lib")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	node := filepath.Join(root, bundledNodeName())
	if err := os.WriteFile(node, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "bin.js"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return node
}

func writeBundledRuntime(t *testing.T, root string) string {
	t.Helper()
	return writeRuntimeTree(t, root, "")
}

func TestBundledNodeCommand(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	node := writeBundledRuntime(t, filepath.Join(wd, "vendor", "dsh"))
	argv, err := bundledNodeCommand()
	if err != nil {
		t.Fatal(err)
	}
	if argv[0] != node {
		t.Fatalf("node = %s, want %s", argv[0], node)
	}
	if runtime.GOOS == "windows" && filepath.Base(argv[0]) != "node.exe" {
		t.Fatalf("windows node name = %s", argv[0])
	}
	if got := argv[len(argv)-2:]; got[0] != "--port" || got[1] != "0" {
		t.Fatalf("flags = %v", argv)
	}
}

func TestResolveCommandPrefersDSHExe(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	writeBundledRuntime(t, filepath.Join(wd, "vendor", "dsh"))
	exe := filepath.Join(wd, "fake-dsh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_EXE", exe)
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	argv, err := DSHConfig{}.resolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	if argv[0] != exe {
		t.Fatalf("argv[0] = %s, want %s", argv[0], exe)
	}
	if got := argv[len(argv)-2:]; got[0] != "--port" || got[1] != "0" {
		t.Fatalf("flags = %v", argv)
	}
}

func TestResolveCommandPrefersPathDSH(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	writeRuntimeTree(t, filepath.Join(wd, "vendor", "dsh"), bundledDSHVersion())
	exe := filepath.Join(wd, "dsh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := lookPath
	lookPath = func(name string) (string, error) {
		if name == "dsh" {
			return exe, nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { lookPath = prev })

	r, err := DSHConfig{}.resolve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != sourcePath || r.Argv[0] != exe {
		t.Fatalf("got kind=%s argv=%v", r.Kind, r.Argv)
	}
}

func TestResolveCommandUsesMatchingCache(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	cache := filepath.Join(wd, "cache")
	t.Setenv("DSH_RUNTIME_DIR", cache)
	node := writeRuntimeTree(t, cache, bundledDSHVersion())
	r, err := DSHConfig{}.resolve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != sourceCache || r.Argv[0] != node {
		t.Fatalf("got kind=%s argv=%v", r.Kind, r.Argv)
	}
}

// An older cache still launches: the version gap is surfaced as an upgrade
// offer, not as a reason to refuse a working runtime (which used to strand
// offline users after every shell update).
func TestResolveCommandUsesOlderCache(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	cache := filepath.Join(wd, "cache")
	t.Setenv("DSH_RUNTIME_DIR", cache)
	node := writeRuntimeTree(t, cache, "0.0.1-rc.0")
	writeRuntimeTree(t, filepath.Join(wd, "vendor", "dsh"), "")
	r, err := DSHConfig{}.resolve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != sourceCache || r.Argv[0] != node {
		t.Fatalf("got kind=%s argv=%v, want cache", r.Kind, r.Argv)
	}
	if got := installedDSHVersion(r); got != "0.0.1-rc.0" {
		t.Fatalf("installed version = %q", got)
	}
	if !shouldOfferDSHUpdate(installedDSHVersion(r), "0.1.2-rc.1") {
		t.Fatal("older cache should be offered an upgrade")
	}
}

func TestResolveCommandUsesBundledRuntime(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	node := writeBundledRuntime(t, filepath.Join(wd, "vendor", "dsh"))
	argv, err := DSHConfig{}.resolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	if argv[0] != node {
		t.Fatalf("argv[0] = %s, want bundled node %s", argv[0], node)
	}
}

func TestResolveCommandMissing(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	t.Setenv("DSH_RUNTIME_BASE_URL", "")
	_, err := DSHConfig{}.resolveCommand()
	if err == nil {
		t.Fatal("expected error when nothing is available")
	}
}

func TestParseDSHWebURL(t *testing.T) {
	url, ok := parseDSHWebURL("dsh web: http://127.0.0.1:43123/?token=abc")
	if !ok || url != "http://127.0.0.1:43123/?token=abc" {
		t.Fatalf("got %q ok=%v", url, ok)
	}
	if _, ok := parseDSHWebURL("listening on http://127.0.0.1:43123"); ok {
		t.Fatal("expected non-token line to be ignored")
	}
	lan := "dsh web: http://127.0.0.1:43123/?token=abc (LAN: http://192.168.1.8:43123/?token=abc)"
	got, ok := parseDSHWebURL(lan)
	if !ok || got != "http://127.0.0.1:43123/?token=abc" {
		t.Fatalf("lan line: got %q ok=%v", got, ok)
	}
	if _, ok := parseDSHWebURL("dsh web: http://127.0.0.1:43123/"); ok {
		t.Fatal("URL without token must be ignored")
	}
}

func TestDefaultHomeDirMatchesNpmDSH(t *testing.T) {
	t.Setenv("DSH_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".dsh")
	if got := defaultHomeDir(); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestDefaultHomeDirHonorsEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSH_HOME", dir)
	if got := defaultHomeDir(); got != dir {
		t.Fatalf("got %s, want %s", got, dir)
	}
	t.Setenv("DSH_HOME", "   ")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := defaultHomeDir(); got != filepath.Join(home, ".dsh") {
		t.Fatalf("blank DSH_HOME = %s", got)
	}
}

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := expandHomePath("~"); got != home {
		t.Fatalf("~ = %s", got)
	}
	if got := expandHomePath(filepath.Join("~", ".dsh")); got != filepath.Join(home, ".dsh") {
		t.Fatalf("~/ = %s", got)
	}
	if got := expandHomePath(`/abs/dsh`); got != `/abs/dsh` {
		t.Fatalf("abs = %s", got)
	}
}
