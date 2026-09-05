package main

import "testing"

func TestCurrentVersionFallsBackToPin(t *testing.T) {
	prev := Version
	t.Cleanup(func() { Version = prev })
	Version = ""
	if got := currentVersion(); got != "0.1.2-rc.1" {
		t.Fatalf("currentVersion() = %q, want pin", got)
	}
	Version = " 0.1.3-rc.1 "
	if got := currentVersion(); got != "0.1.3-rc.1" {
		t.Fatalf("currentVersion() = %q, want ldflags override", got)
	}
}

func TestCapsuleLabelText(t *testing.T) {
	if got := capsuleLabelText("0.1.2-rc.1"); got != "更新到 0.1.2-rc.1" {
		t.Fatalf("label = %q", got)
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

func TestDSHCapsuleLabel(t *testing.T) {
	if got := dshCapsuleLabel("0.1.2-rc.2"); got != "更新 dsh 到 0.1.2-rc.2" {
		t.Fatalf("label = %q", got)
	}
}

func TestGlobalUpgradeArgs(t *testing.T) {
	got := globalUpgradeArgs("0.1.2-rc.1")
	if got[0] != "install" || got[len(got)-1] != "@deepseek-ai/dsh@0.1.2-rc.1" {
		t.Fatalf("args = %v", got)
	}
}

func TestHarnessTitle(t *testing.T) {
	if got := harnessTitle(resolvedDSH{Kind: sourcePath}); got != "DeepSeek Harness · 本机 dsh" {
		t.Fatalf("got %q", got)
	}
	if got := harnessTitle(resolvedDSH{Kind: sourceCache}); got != "DeepSeek Harness · 缓存 runtime" {
		t.Fatalf("got %q", got)
	}
}

func TestCanUpdateDSH(t *testing.T) {
	if !canUpdateDSH(sourcePath) || !canUpdateDSH(sourceCache) {
		t.Fatal("path and cache should be updatable")
	}
	if canUpdateDSH(sourceNpx) || canUpdateDSH(sourceRepo) {
		t.Fatal("npx/repo must not get a fake upgrade")
	}
}
