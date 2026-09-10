package app

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestShouldRevealHarness(t *testing.T) {
	var harness, prep application.WebviewWindow
	if !shouldRevealHarness(&harness, &harness) {
		t.Fatal("first reveal should proceed while harness is current")
	}
	if shouldRevealHarness(&prep, &harness) {
		t.Fatal("must not reveal harness after prep has taken the foreground")
	}
}
