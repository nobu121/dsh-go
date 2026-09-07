package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLookNamedFindsExtraDir(t *testing.T) {
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	dir := t.TempDir()
	exe := filepath.Join(dir, "dsh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := lookPath
	prevDirs := extraBinDirsFn
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	extraBinDirsFn = func() []string { return []string{dir} }
	t.Cleanup(func() {
		lookPath = prev
		extraBinDirsFn = prevDirs
	})
	got, err := lookNamed("dsh")
	if err != nil {
		t.Fatal(err)
	}
	if got != exe {
		t.Fatalf("got %s, want %s", got, exe)
	}
}

func TestLookNamedFindsWindowsCmd(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows PATHEXT")
	}
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	dir := t.TempDir()
	exe := filepath.Join(dir, "dsh.cmd")
	if err := os.WriteFile(exe, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := lookPath
	prevDirs := extraBinDirsFn
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	extraBinDirsFn = func() []string { return []string{dir} }
	t.Cleanup(func() {
		lookPath = prev
		extraBinDirsFn = prevDirs
	})
	got, err := lookNamed("dsh")
	if err != nil {
		t.Fatal(err)
	}
	if got != exe {
		t.Fatalf("got %s, want %s", got, exe)
	}
}

func TestMergePATHDedupes(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := mergePATH("/a"+sep+"/b", "/b"+sep+"/c", "/a")
	if got != "/a"+sep+"/b"+sep+"/c" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveFindsDSHOutsidePATH(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	t.Setenv("DSH_SKIP_LOGIN_PATH", "1")
	dir := t.TempDir()
	exe := filepath.Join(dir, "dsh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := lookPath
	prevDirs := extraBinDirsFn
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	extraBinDirsFn = func() []string { return []string{dir} }
	t.Cleanup(func() {
		lookPath = prev
		extraBinDirsFn = prevDirs
	})
	r, err := DSHConfig{}.resolve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != sourcePath || r.Argv[0] != exe {
		t.Fatalf("got kind=%s argv=%v", r.Kind, r.Argv)
	}
}
