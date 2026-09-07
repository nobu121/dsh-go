package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOrderBySpeedPicksFasterMirror(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fast.Close()

	got := orderBySpeed(context.Background(), []string{slow.URL, fast.URL}, "SHA256SUMS")
	if len(got) != 2 || got[0] != fast.URL {
		t.Fatalf("got %#v, want %s first", got, fast.URL)
	}
}

func TestOrderBySpeedKeepsLocalOrder(t *testing.T) {
	bases := []string{"file:///tmp/a", "file:///tmp/b"}
	got := orderBySpeed(context.Background(), bases, "SHA256SUMS")
	if got[0] != bases[0] || got[1] != bases[1] {
		t.Fatalf("got %#v", got)
	}
}
