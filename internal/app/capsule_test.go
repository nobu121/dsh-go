package app

import (
	"context"
	"testing"
	"time"
)

func TestUpdateCapsuleProgress(t *testing.T) {
	c := newUpdateCapsule()
	c.showDSH("0.4.0")
	p := c.progress()
	if p.Stage != prepOfferStage {
		t.Fatalf("progress stage = %q", p.Stage)
	}
	if p.Action != "立即更新" || p.Version != "0.4.0" {
		t.Fatalf("progress action=%q version=%q", p.Action, p.Version)
	}
}

func TestUpdateCapsuleProgressListsBoth(t *testing.T) {
	c := newUpdateCapsule()
	c.stashApp("1.2.3")
	c.showDSH("0.4.0")
	p := c.progress()
	if p.Message != "将一并更新客户端和运行时。" {
		t.Fatalf("message = %q", p.Message)
	}
	if len(p.Items) != 2 || p.Items[0].Kind != capsuleKindApp || p.Items[1].Kind != capsuleKindDSH {
		t.Fatalf("items = %+v", p.Items)
	}
	if p.Items[0].Name != "客户端" || p.Items[0].Version != "1.2.3" {
		t.Fatalf("shell item = %+v", p.Items[0])
	}
	if p.Items[1].Name != "运行时" || p.Items[1].Version != "0.4.0" {
		t.Fatalf("runtime item = %+v", p.Items[1])
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
