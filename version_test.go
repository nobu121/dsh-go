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
