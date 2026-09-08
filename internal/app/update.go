package app

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	updateInterval      = 6 * time.Hour
	dshCheckTimeout     = 20 * time.Second
	startupOfferTimeout = 2 * time.Second
	manualUpdateEvt     = "dsh-go:manual-update"
)

// checkShellUpdate reports the newest client version the client-latest channel
// advertises when it is newer than the running shell, or "" when the shell is
// current or the channel is unreachable. The CNB and GitHub mirrors are raced
// (see latestClientVersion): whichever answers first wins, so an unreachable
// GitHub never stalls a launch where CNB responds, and vice versa.
//
// UpdateRepo gates the whole check: dev builds (empty UpdateRepo) never touch
// the channel and keep exercising offers through DSH_SIMULATE_UPDATE only.
func checkShellUpdate(ctx context.Context) string {
	if UpdateRepo == "" {
		return ""
	}
	v := latestClientVersion(ctx)
	if v == "" {
		return ""
	}
	if !shouldOfferDSHUpdate(shellVersion(), v) {
		return ""
	}
	log.Printf("update available: %s", v)
	return v
}

func runUpdateLoop(ctx context.Context, app *application.App) {
	// First check happens on the prep page before launch (see waitStartupOffer).
	t := time.NewTicker(updateInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if ver := checkShellUpdate(ctx); ver != "" {
				app.Event.Emit(manualUpdateEvt, ver)
			}
		}
	}
}

// runDSHUpdateLoop re-checks the runtime channel on the same cadence as the
// shell updater, but through a separate path: a new dsh needs no new shell.
func runDSHUpdateLoop(ctx context.Context, dsh *DSH, show func(string)) {
	t := time.NewTicker(updateInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			offerDSHUpdate(ctx, dsh, show)
		}
	}
}
