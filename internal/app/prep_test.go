package app

import "testing"

func TestPrepStartProgress(t *testing.T) {
	p := prepStartProgress()
	if p.Stage != "start" || p.Message != prepStartMsg {
		t.Fatalf("got stage=%s message=%q", p.Stage, p.Message)
	}
}
