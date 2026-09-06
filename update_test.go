package main

import "testing"

func TestIsManualInstallArtifact(t *testing.T) {
	if !isManualInstallArtifact("dsh-go-darwin-arm64.dmg") {
		t.Fatal("dmg should be manual")
	}
	if isManualInstallArtifact("dsh-go-windows-amd64.exe") {
		t.Fatal("exe should auto-install")
	}
}

func TestReleaseAssetURL(t *testing.T) {
	got := releaseAssetURL("v0.1.2-rc.1", "dsh-go-darwin-arm64.dmg")
	want := "https://cnb.cool/nobu121/dsh-go/-/releases/download/v0.1.2-rc.1/dsh-go-darwin-arm64.dmg"
	if got != want {
		t.Fatalf("got %s", got)
	}
}
