package main

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const updateInterval = 6 * time.Hour

func setupUpdater(app *application.App) bool {
	repo := UpdateRepo
	if repo == "" {
		log.Printf("updater disabled: UpdateRepo is empty")
		return false
	}
	gh, err := github.New(github.Config{
		Repository:    repo,
		Prerelease:    true,
		ChecksumAsset: "SHA256SUMS",
	})
	if err != nil {
		log.Printf("updater: %v", err)
		return false
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: currentVersion(),
		Providers:      []updater.Provider{gh},
		Window:         updater.WindowNone,
	}); err != nil {
		log.Printf("updater init: %v", err)
		return false
	}
	return true
}

func runUpdateLoop(ctx context.Context, app *application.App) {
	check := func() {
		rel, err := app.Updater.Check(ctx)
		if err != nil {
			log.Printf("update check: %v", err)
			return
		}
		if rel == nil {
			return
		}
		log.Printf("update available: %s; downloading", rel.Version)
		if err := app.Updater.DownloadAndInstall(ctx); err != nil {
			log.Printf("update download: %v", err)
		}
	}
	check()
	t := time.NewTicker(updateInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			check()
		}
	}
}
