package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func resetChannelCache(t *testing.T) {
	t.Helper()
	set := func(v string, at time.Time) {
		channelMu.Lock()
		channelVer, channelAt = v, at
		channelMu.Unlock()
	}
	channelMu.Lock()
	prevVer, prevAt := channelVer, channelAt
	channelMu.Unlock()
	t.Cleanup(func() { set(prevVer, prevAt) })
	set("", time.Time{})
}

func writeChannelManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runtimeChannelFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTargetDSHVersionFromChannel(t *testing.T) {
	resetChannelCache(t)
	stage := t.TempDir()
	writeChannelManifest(t, stage, `{"version":"0.9.1-rc.3","node":"24.18.0"}`)
	t.Setenv("DSH_RUNTIME_VERSION", "")
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	if got := targetDSHVersion(context.Background()); got != "0.9.1-rc.3" {
		t.Fatalf("target = %q", got)
	}
}

// The channel is the only thing that has to move for a new dsh, so its answer
// must win over the version vendored into this build.
func TestTargetDSHVersionOverridesBundledPin(t *testing.T) {
	resetChannelCache(t)
	stage := t.TempDir()
	writeChannelManifest(t, stage, `{"version":"9.9.9"}`)
	t.Setenv("DSH_RUNTIME_VERSION", "")
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	got := targetDSHVersion(context.Background())
	if got == bundledDSHVersion() {
		t.Fatalf("target %q should not be the bundled pin", got)
	}
	if got != "9.9.9" {
		t.Fatalf("target = %q", got)
	}
}

func TestTargetDSHVersionFallsBackToBundled(t *testing.T) {
	resetChannelCache(t)
	t.Setenv("DSH_RUNTIME_VERSION", "")
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(filepath.Join(t.TempDir(), "absent")))
	if got := targetDSHVersion(context.Background()); got != bundledDSHVersion() {
		t.Fatalf("target = %q, want bundled pin %q", got, bundledDSHVersion())
	}
}

func TestTargetDSHVersionServesStaleAfterChannelLoss(t *testing.T) {
	resetChannelCache(t)
	stage := t.TempDir()
	writeChannelManifest(t, stage, `{"version":"0.9.2"}`)
	t.Setenv("DSH_RUNTIME_VERSION", "")
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	if got := targetDSHVersion(context.Background()); got != "0.9.2" {
		t.Fatalf("warm target = %q", got)
	}
	if err := os.Remove(filepath.Join(stage, runtimeChannelFile)); err != nil {
		t.Fatal(err)
	}
	channelMu.Lock()
	channelAt = time.Now().Add(-2 * runtimeChannelTTL)
	channelMu.Unlock()
	if got := targetDSHVersion(context.Background()); got != "0.9.2" {
		t.Fatalf("stale target = %q, want last known answer", got)
	}
}

func TestTargetDSHVersionEnvOverride(t *testing.T) {
	resetChannelCache(t)
	stage := t.TempDir()
	writeChannelManifest(t, stage, `{"version":"0.9.2"}`)
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	t.Setenv("DSH_RUNTIME_VERSION", "1.2.3")
	if got := targetDSHVersion(context.Background()); got != "1.2.3" {
		t.Fatalf("target = %q", got)
	}
}

func TestTargetDSHVersionRejectsJunkManifest(t *testing.T) {
	resetChannelCache(t)
	stage := t.TempDir()
	writeChannelManifest(t, stage, `{"version":"not-a-version"}`)
	t.Setenv("DSH_RUNTIME_VERSION", "")
	t.Setenv("DSH_RUNTIME_BASE_URL", fileURL(stage))
	if got := targetDSHVersion(context.Background()); got != bundledDSHVersion() {
		t.Fatalf("target = %q, want bundled pin", got)
	}
}

// A runtime whose VERSION stamp is missing still reports its version, so the
// upgrade check never silently goes quiet.
func TestReadRuntimeVersionFallsBackToPackageJSON(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "node_modules", "@deepseek-ai", "dsh")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "package.json"), []byte(`{"version":"0.5.0-rc.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readRuntimeVersion(root); got != "0.5.0-rc.2" {
		t.Fatalf("version = %q", got)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("0.6.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readRuntimeVersion(root); got != "0.6.0" {
		t.Fatalf("VERSION should win, got %q", got)
	}
}
