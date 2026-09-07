package main

import (
	"path/filepath"
	"testing"
)

func TestInitialPrepUsesStartWhenRuntimeExists(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	cache := filepath.Join(wd, "cache")
	t.Setenv("DSH_RUNTIME_DIR", cache)
	writeRuntimeTree(t, cache, bundledDSHVersion())
	p := initialPrepProgress(DSHConfig{})
	if p.Stage != "start" || p.Message != prepStartMsg {
		t.Fatalf("got stage=%s message=%q", p.Stage, p.Message)
	}
}

func TestInitialPrepDetectsWhenMissing(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)
	isolateLookPath(t)
	t.Setenv("DSH_EXE", "")
	t.Setenv("DSH_REPO", filepath.Join(wd, "missing-repo"))
	t.Setenv("DSH_RUNTIME_DIR", filepath.Join(wd, "cache-empty"))
	t.Setenv("DSH_RUNTIME_BASE_URL", "")
	p := initialPrepProgress(DSHConfig{})
	if p.Stage != "detect" || p.Message != prepDetectMsg {
		t.Fatalf("got stage=%s message=%q", p.Stage, p.Message)
	}
}
