package app

import (
	"context"
	"testing"
	"time"
)

func TestUpdateCapsuleProgress(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "zh")
	c := newUpdateCapsule()
	c.showDSH("0.4.0")
	p := c.progress()
	if p.Stage != prepOfferStage {
		t.Fatalf("progress stage = %q", p.Stage)
	}
	if p.Action != uiZH.UpdateNow || p.Version != "0.4.0" {
		t.Fatalf("progress action=%q version=%q", p.Action, p.Version)
	}
}

func TestUpdateCapsuleProgressListsBoth(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "zh")
	c := newUpdateCapsule()
	c.stashApp("1.2.3")
	c.showDSH("0.4.0")
	p := c.progress()
	if p.Message != uiZH.OfferBoth {
		t.Fatalf("message = %q", p.Message)
	}
	if len(p.Items) != 2 || p.Items[0].Kind != capsuleKindApp || p.Items[1].Kind != capsuleKindDSH {
		t.Fatalf("items = %+v", p.Items)
	}
	if p.Items[0].Name != uiZH.Client || p.Items[0].Version != "1.2.3" {
		t.Fatalf("shell item = %+v", p.Items[0])
	}
	if p.Items[1].Name != uiZH.Runtime || p.Items[1].Version != "0.4.0" {
		t.Fatalf("runtime item = %+v", p.Items[1])
	}
}

func TestUpdateCapsuleProgressUsesEnglish(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "en")
	c := newUpdateCapsule()
	c.show("1.2.3")
	p := c.progress()
	if p.Action != uiEN.UpdateNow || p.Message != uiEN.OfferApp {
		t.Fatalf("action=%q message=%q", p.Action, p.Message)
	}
	if len(p.Items) != 1 || p.Items[0].Name != uiEN.Client {
		t.Fatalf("items = %+v", p.Items)
	}
}

func TestUpdateCapsuleRestoreApp(t *testing.T) {
	c := newUpdateCapsule()
	c.show("1.0.0")
	c.showDSH("2.0.0")
	c.restoreAppIfPending()
	if kind, ver, ok := c.pending(); !ok || kind != capsuleKindApp || ver != "1.0.0" {
		t.Fatalf("pending = %s %s %v", kind, ver, ok)
	}
}

func TestUpdateCapsuleStashAppUnderDSH(t *testing.T) {
	c := newUpdateCapsule()
	c.showDSH("2.0.0")
	c.stashApp("1.0.0")
	if kind, ver, ok := c.pending(); !ok || kind != capsuleKindDSH || ver != "2.0.0" {
		t.Fatalf("dsh should stay visible, got %s %s %v", kind, ver, ok)
	}
	c.restoreAppIfPending()
	if kind, ver, ok := c.pending(); !ok || kind != capsuleKindApp || ver != "1.0.0" {
		t.Fatalf("stashed app = %s %s %v", kind, ver, ok)
	}
}

func TestUpdateCapsuleProceed(t *testing.T) {
	c := newUpdateCapsule()
	c.show("1.0.0")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		c.wait(ctx)
		close(done)
	}()
	c.proceed()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("wait did not return")
	}
	if _, _, ok := c.pending(); ok {
		t.Fatal("offer still pending after proceed")
	}
}
