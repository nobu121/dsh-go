package app

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	updateInterval  = 6 * time.Hour
	dshCheckTimeout     = 20 * time.Second
	startupOfferTimeout = 2 * time.Second
	manualUpdateEvt = "dsh-go:manual-update"
)

func setupUpdater(app *application.App) bool {
	repo := UpdateRepo
	if repo == "" {
		log.Printf("updater disabled: UpdateRepo is empty")
		return false
	}
	// Shell releases are full releases, so /releases/latest picks them up and
	// skips the runtime channel, which is published as a prerelease at a fixed
	// non-version tag. Enabling Prerelease here would walk the raw releases
	// list, whose newest entry could be that channel tag.
	gh, err := github.New(github.Config{
		Repository:    repo,
		ChecksumAsset: "SHA256SUMS",
	})
	if err != nil {
		log.Printf("updater: %v", err)
		return false
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: shellVersion(),
		Providers:      []updater.Provider{gh},
		Window:         updater.WindowNone,
	}); err != nil {
		log.Printf("updater init: %v", err)
		return false
	}
	return true
}

func checkShellUpdate(ctx context.Context, app *application.App) string {
	if app != nil && app.Updater != nil && UpdateRepo != "" {
		rel, err := app.Updater.Check(ctx)
		if err != nil {
			log.Printf("update check: %v", err)
		} else if rel != nil {
			log.Printf("update available: %s", rel.Version)
			return rel.Version
		} else {
			return ""
		}
	}
	latest := latestReleaseVersion(ctx)
	if latest == "" || latest == shellVersion() || !shouldOfferDSHUpdate(shellVersion(), latest) {
		return ""
	}
	log.Printf("update available: %s", latest)
	return latest
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
			if ver := checkShellUpdate(ctx, app); ver != "" {
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
