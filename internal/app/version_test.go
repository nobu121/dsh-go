package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestShellVersionFallsBackToAppVersion(t *testing.T) {
	prev := Version
	t.Cleanup(func() { Version = prev })
	Version = ""
	SetEmbeddedVersions(
		string(mustReadFile(t, filepath.Join("..", "..", "app.version"))),
		string(mustReadFile(t, filepath.Join("..", "..", "dsh.version"))),
	)
	want := strings.TrimSpace(string(mustReadFile(t, filepath.Join("..", "..", "app.version"))))
	if got := shellVersion(); got != want {
		t.Fatalf("shellVersion() = %q, want app.version %q", got, want)
	}
	Version = " 9.9.9 "
	if got := shellVersion(); got != "9.9.9" {
		t.Fatalf("shellVersion() = %q, want ldflags override", got)
	}
}

// The shell version and the vendored dsh version are separate numbers; letting
// them drift is the whole point of the runtime channel.
func TestShellVersionIndependentOfBundledDSH(t *testing.T) {
	prev := Version
	t.Cleanup(func() { Version = prev })
	Version = "9.9.9"
	if shellVersion() == bundledDSHVersion() {
		t.Fatal("shell version must not track the bundled dsh pin")
	}
	want := strings.TrimSpace(string(mustReadFile(t, filepath.Join("..", "..", "dsh.version"))))
	if got := bundledDSHVersion(); got != want {
		t.Fatalf("bundledDSHVersion() = %q, want dsh.version %q", got, want)
	}
}

func TestParseDSHVersion(t *testing.T) {
	if got := parseDSHVersion("dsh 0.1.2-rc.1\n"); got != "0.1.2-rc.1" {
		t.Fatalf("got %q", got)
	}
}

func TestCompareDSHVersion(t *testing.T) {
	if compareDSHVersion("0.1.2-rc.1", "0.1.2-rc.2") >= 0 {
		t.Fatal("rc.1 should be older than rc.2")
	}
	if compareDSHVersion("0.1.2-rc.2", "0.1.2") >= 0 {
		t.Fatal("rc should be older than release")
	}
	if compareDSHVersion("0.1.2", "0.1.2") != 0 {
		t.Fatal("same version")
	}
}

func TestShouldOfferDSHUpdate(t *testing.T) {
	if !shouldOfferDSHUpdate("0.1.1-rc.1", "0.1.2-rc.1") {
		t.Fatal("older global dsh should offer update")
	}
	if shouldOfferDSHUpdate("0.1.2-rc.1", "0.1.2-rc.1") {
		t.Fatal("same version should not offer")
	}
	if shouldOfferDSHUpdate("0.1.3-rc.1", "0.1.2-rc.1") {
		t.Fatal("newer installed should not offer downgrade")
	}
}

func TestGlobalUpgradeArgs(t *testing.T) {
	got := globalUpgradeArgs("0.1.2-rc.1")
	if got[0] != "install" || got[len(got)-1] != "@deepseek-ai/dsh@0.1.2-rc.1" {
		t.Fatalf("args = %v", got)
	}
}

func TestCanUpdateDSH(t *testing.T) {
	if !canUpdateDSH(sourcePath) || !canUpdateDSH(sourceCache) || !canUpdateDSH(sourceBundled) {
		t.Fatal("path, cache and bundled should be updatable")
	}
	if canUpdateDSH(sourceNpx) || canUpdateDSH(sourceRepo) {
		t.Fatal("npx/repo must not get a fake upgrade")
	}
}
