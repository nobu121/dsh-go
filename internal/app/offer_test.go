package app

import (
	"context"
	"testing"
	"time"
)

func TestWaitStartupOfferSimulatedBlocksUntilLater(t *testing.T) {
	t.Setenv("DSH_SIMULATE_DSH_UPDATE", "9.9.9")
	t.Setenv("DSH_SIMULATE_UPDATE", "")
	offer := newUpdateCapsule()
	presented := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		waitStartupOffer(
			context.Background(),
			NewDSH(DSHConfig{}, nil),
			offer,
			func() { presented <- struct{}{} },
			func(PrepProgress) {},
		)
		close(done)
	}()
	select {
	case <-presented:
	case <-time.After(time.Second):
		t.Fatal("offer was not presented")
	}
	if !offer.isDSH() {
		t.Fatal("expected dsh offer")
	}
	offer.proceed()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("wait did not return after proceed")
	}
}

func TestWaitStartupOfferSkipsWhenCurrent(t *testing.T) {
	t.Setenv("DSH_SIMULATE_UPDATE", "")
	t.Setenv("DSH_SIMULATE_DSH_UPDATE", "")
	offer := newUpdateCapsule()
	waitStartupOffer(
		context.Background(),
		NewDSH(DSHConfig{}, nil),
		offer,
		func() { t.Fatal("should not present") },
		func(PrepProgress) {},
	)
	if _, _, ok := offer.pending(); ok {
		t.Fatal("no offer expected")
	}
}
