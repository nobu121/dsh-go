package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUnixShimBody(t *testing.T) {
	body := unixShimBody(`/opt/dsh/node`, `/opt/dsh/node_modules/@deepseek-ai/dsh/lib/bin.js`)
	if !strings.Contains(body, shimMarker) {
		t.Fatal("missing marker")
	}
	if !strings.Contains(body, `'/opt/dsh/node'`) {
		t.Fatalf("node: %s", body)
	}
	if !strings.Contains(body, `"$@"`) {
		t.Fatal("missing args")
	}
}

func TestWindowsShimBody(t *testing.T) {
	body := windowsShimBody(`C:\App Data\node.exe`, `C:\App Data\bin.js`)
	if !strings.Contains(body, shimMarker) {
		t.Fatal("missing marker")
	}
	if !strings.Contains(body, `"C:\App Data\node.exe"`) {
		t.Fatalf("node: %s", body)
	}
	if !strings.Contains(body, "%*") {
		t.Fatal("missing args")
	}
}

func TestEnsureDSHShimWrites(t *testing.T) {
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	wd := t.TempDir()
	bin := filepath.Join(wd, "bin")
	root := filepath.Join(wd, "runtime")
	t.Setenv("DSH_BIN_DIR", bin)
	writeRuntimeTree(t, root, bundledDSHVersion())
	if err := ensureDSHShim(root); err != nil {
		t.Fatal(err)
	}
	name := "dsh"
	if runtime.GOOS == "windows" {
		name = "dsh.cmd"
	}
	path := filepath.Join(bin, name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), shimMarker) {
		t.Fatalf("shim: %s", b)
	}
	if !isDSHGoShim(path) {
		t.Fatal("isDSHGoShim")
	}
}

func TestResolveSkipsOwnShimUsesCache(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	cache := filepath.Join(wd, "cache")
	bin := filepath.Join(wd, "bin")
	t.Setenv("DSH_RUNTIME_DIR", cache)
	t.Setenv("DSH_BIN_DIR", bin)
	node := writeRuntimeTree(t, cache, bundledDSHVersion())
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(bin, "dsh")
	if runtime.GOOS == "windows" {
		shim = filepath.Join(bin, "dsh.cmd")
	}
	if err := os.WriteFile(shim, []byte("REM "+shimMarker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := lookPath
	prevDirs := extraBinDirsFn
	lookPath = func(name string) (string, error) {
		if name == "dsh" {
			return shim, nil
		}
		return "", exec.ErrNotFound
	}
	extraBinDirsFn = func() []string { return []string{bin} }
	t.Cleanup(func() {
		lookPath = prev
		extraBinDirsFn = prevDirs
	})
	r, err := DSHConfig{}.resolve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != sourceCache || r.Argv[0] != node {
		t.Fatalf("got kind=%s argv=%v", r.Kind, r.Argv)
	}
}

func TestLaunchEnvPrependsShimAndNode(t *testing.T) {
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	wd := t.TempDir()
	bin := filepath.Join(wd, "bin")
	cache := filepath.Join(wd, "cache")
	t.Setenv("DSH_BIN_DIR", bin)
	t.Setenv("PATH", "/usr/bin")
	writeRuntimeTree(t, cache, bundledDSHVersion())
	env := launchEnv("/tmp/dsh-home", resolvedDSH{Kind: sourceCache, Path: cache})
	var path, home string
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(k, "PATH") {
			path = v
		}
		if k == "DSH_HOME" {
			home = v
		}
	}
	if home != "/tmp/dsh-home" {
		t.Fatalf("DSH_HOME=%s", home)
	}
	dirs := splitPATH(path)
	if len(dirs) < 3 || !sameFilePath(dirs[0], bin) || !sameFilePath(dirs[1], cache) {
		t.Fatalf("PATH=%q", path)
	}
}

func TestLaunchEnvPathSourceSkipsShim(t *testing.T) {
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	wd := t.TempDir()
	t.Setenv("DSH_BIN_DIR", filepath.Join(wd, "bin"))
	env := launchEnv("/tmp/dsh-home", resolvedDSH{Kind: sourcePath, Path: filepath.Join(wd, "dsh")})
	if _, err := os.Stat(filepath.Join(wd, "bin")); err == nil {
		t.Fatal("should not create bin for PATH source")
	}
	foundHome := false
	for _, e := range env {
		if e == "DSH_HOME=/tmp/dsh-home" {
			foundHome = true
		}
	}
	if !foundHome {
		t.Fatal("DSH_HOME")
	}
}

func TestPrependPATHReplacesExisting(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := prependPATH([]string{"FOO=1", "PATH=/usr/bin", "BAR=2"}, "/shim", "/node")
	var path string
	nPATH := 0
	for _, e := range got {
		k, v, ok := strings.Cut(e, "=")
		if ok && strings.EqualFold(k, "PATH") {
			nPATH++
			path = v
		}
	}
	if nPATH != 1 {
		t.Fatalf("PATH entries = %d", nPATH)
	}
	want := "/shim" + sep + "/node" + sep + "/usr/bin"
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestAppendUserPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	got, changed := appendUserPATH("/a"+sep+"/b", "/c")
	if !changed || got != "/a"+sep+"/b"+sep+"/c" {
		t.Fatalf("got %q changed=%v", got, changed)
	}
	got, changed = appendUserPATH("/a"+sep+"/c", "/c")
	if changed || got != "/a"+sep+"/c" {
		t.Fatalf("idempotent: %q changed=%v", got, changed)
	}
}

func TestRemoveUserPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	got, changed := removeUserPATH("/a"+sep+"/c"+sep+"/b", "/c")
	if !changed || got != "/a"+sep+"/b" {
		t.Fatalf("got %q changed=%v", got, changed)
	}
	got, changed = removeUserPATH("/a"+sep+"/b", "/c")
	if changed || got != "/a"+sep+"/b" {
		t.Fatalf("missing: %q changed=%v", got, changed)
	}
	next, _ := appendUserPATH("/a", "/c")
	next, changed = removeUserPATH(next, "/c")
	if !changed || next != "/a" {
		t.Fatalf("round-trip %q changed=%v", next, changed)
	}
}

func TestIsDSHGoShimMarker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSH_BIN_DIR", filepath.Join(dir, "other"))
	p := filepath.Join(dir, "dsh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n# "+shimMarker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isDSHGoShim(p) {
		t.Fatal("marker")
	}
	plain := filepath.Join(dir, "real")
	if err := os.WriteFile(plain, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if isDSHGoShim(plain) {
		t.Fatal("plain file")
	}
}
