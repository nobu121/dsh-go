package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeBundledRuntime(t *testing.T, root string) string {
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
	return node
}

func TestBundledNodeCommand(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
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
	writeBundledRuntime(t, filepath.Join(wd, "vendor", "dsh"))
	exe := filepath.Join(wd, "fake-dsh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_EXE", exe)
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

func TestResolveCommandUsesBundledRuntime(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	node := writeBundledRuntime(t, filepath.Join(wd, "vendor", "dsh"))
	argv, err := DSHConfig{}.resolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	if argv[0] != node {
		t.Fatalf("argv[0] = %s, want bundled node %s", argv[0], node)
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
}
