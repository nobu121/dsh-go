package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetClientChannelCache(t *testing.T) {
	t.Helper()
	set := func(v string, at time.Time) {
		clientMu.Lock()
		clientVer, clientAt = v, at
		clientMu.Unlock()
	}
	clientMu.Lock()
	prevVer, prevAt := clientVer, clientAt
	clientMu.Unlock()
	t.Cleanup(func() { set(prevVer, prevAt) })
	set("", time.Time{})
}

func writeClientManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, clientChannelFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The client channel must default to the CNB mirror first (reachable without
// a proxy in mainland China) with GitHub as the fallback, and client.json must
// ride a fixed tag, never a version tag.
func TestClientReleaseBasesDefaults(t *testing.T) {
	oldBase, oldRepo := RuntimeBaseURL, UpdateRepo
	t.Cleanup(func() {
		RuntimeBaseURL, UpdateRepo = oldBase, oldRepo
	})
	RuntimeBaseURL, UpdateRepo = "", "nobu121/dsh-go"
	t.Setenv("DSH_CLIENT_BASE_URL", "")

	got := clientReleaseBases()
	if len(got) != 2 {
		t.Fatalf("bases = %#v", got)
	}
	if !strings.Contains(got[0], "cnb.cool") || !strings.HasSuffix(got[0], "/"+clientChannelTag) {
		t.Fatalf("cnb = %s", got[0])
	}
	if !strings.Contains(got[1], "github.com") || !strings.HasSuffix(got[1], "/"+clientChannelTag) {
		t.Fatalf("github = %s", got[1])
	}
}

func TestClientReleaseBasesEnvOverride(t *testing.T) {
	stage := t.TempDir()
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))
	got := clientReleaseBases()
	if len(got) != 1 || got[0] != fileURL(stage) {
		t.Fatalf("bases = %#v", got)
	}
}

func TestLatestClientVersionFromChannel(t *testing.T) {
	resetClientChannelCache(t)
	stage := t.TempDir()
	writeClientManifest(t, stage, `{"version":"0.9.1"}`)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))
	if got := latestClientVersion(context.Background()); got != "0.9.1" {
		t.Fatalf("latest = %q", got)
	}
}

func TestLatestClientVersionUnreachableIsEmpty(t *testing.T) {
	resetClientChannelCache(t)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(filepath.Join(t.TempDir(), "absent")))
	if got := latestClientVersion(context.Background()); got != "" {
		t.Fatalf("latest = %q, want empty when channel is unreachable", got)
	}
}

// A reachable answer is memoised, so the background re-check inside one
// session does not hit the network again until the TTL expires, and a lost
// channel after a warm answer serves the last known version.
func TestLatestClientVersionServesStaleAfterChannelLoss(t *testing.T) {
	resetClientChannelCache(t)
	stage := t.TempDir()
	writeClientManifest(t, stage, `{"version":"0.9.2"}`)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))
	if got := latestClientVersion(context.Background()); got != "0.9.2" {
		t.Fatalf("warm latest = %q", got)
	}
	if err := os.Remove(filepath.Join(stage, clientChannelFile)); err != nil {
		t.Fatal(err)
	}
	clientMu.Lock()
	clientAt = time.Now().Add(-2 * clientChannelTTL)
	clientMu.Unlock()
	if got := latestClientVersion(context.Background()); got != "0.9.2" {
		t.Fatalf("stale latest = %q, want last known answer", got)
	}
}

func TestLatestClientVersionRejectsJunkManifest(t *testing.T) {
	resetClientChannelCache(t)
	stage := t.TempDir()
	writeClientManifest(t, stage, `{"version":"not-a-version"}`)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))
	if got := latestClientVersion(context.Background()); got != "" {
		t.Fatalf("latest = %q, want empty for junk manifest", got)
	}
}

// saveShellState snapshots the package vars checkShellUpdate consults and
// restores them on cleanup, so tests never leak versions or repos.
func saveShellState(t *testing.T) {
	t.Helper()
	oldRepo, oldVer := UpdateRepo, Version
	t.Cleanup(func() {
		UpdateRepo, Version = oldRepo, oldVer
	})
}

// checkShellUpdate must stay gated on UpdateRepo: dev builds (empty UpdateRepo)
// never consult the client channel and only exercise offers through the
// simulation env hooks.
func TestCheckShellUpdateDisabledWithoutRepo(t *testing.T) {
	saveShellState(t)
	UpdateRepo = ""
	Version = "0.1.0"
	resetClientChannelCache(t)
	stage := t.TempDir()
	writeClientManifest(t, stage, `{"version":"9.9.9"}`)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))
	if got := checkShellUpdate(context.Background()); got != "" {
		t.Fatalf("check = %q, want empty when UpdateRepo is empty", got)
	}
}

func TestCheckShellUpdateNewerOnly(t *testing.T) {
	saveShellState(t)
	UpdateRepo = "nobu121/dsh-go"
	Version = "0.1.0"
	resetClientChannelCache(t)

	stage := t.TempDir()
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(stage))

	writeClientManifest(t, stage, `{"version":"0.2.0"}`)
	if got := checkShellUpdate(context.Background()); got != "0.2.0" {
		t.Fatalf("newer check = %q, want 0.2.0", got)
	}

	// Same and older channel versions must not offer an update.
	for _, v := range []string{"0.1.0", "0.0.9"} {
		resetClientChannelCache(t)
		if err := os.Remove(filepath.Join(stage, clientChannelFile)); err != nil {
			t.Fatal(err)
		}
		writeClientManifest(t, stage, `{"version":"`+v+`"}`)
		if got := checkShellUpdate(context.Background()); got != "" {
			t.Fatalf("check for %s = %q, want empty", v, got)
		}
	}
}

func TestCheckShellUpdateUnreachableIsEmpty(t *testing.T) {
	saveShellState(t)
	UpdateRepo = "nobu121/dsh-go"
	Version = "0.1.0"
	resetClientChannelCache(t)
	t.Setenv("DSH_CLIENT_BASE_URL", fileURL(filepath.Join(t.TempDir(), "absent")))
	if got := checkShellUpdate(context.Background()); got != "" {
		t.Fatalf("check = %q, want empty when channel is unreachable", got)
	}
}
