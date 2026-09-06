package main

import (
	"context"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	updateInterval   = 6 * time.Hour
	manualUpdateEvt  = "dsh-go:manual-update"
	cnbReleasePrefix = "https://cnb.cool/nobu121/dsh-go/-/releases/download/v"
)

var (
	manualUpdateMu  sync.Mutex
	manualUpdateURL string
)

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
		if isManualInstallArtifact(rel.Artifact.Filename) {
			url := releaseAssetURL(rel.Version, rel.Artifact.Filename)
			setManualUpdateURL(url)
			log.Printf("update available: %s; open %s", rel.Version, url)
			app.Event.Emit(manualUpdateEvt, rel.Version)
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

func isManualInstallArtifact(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".dmg")
}

func releaseAssetURL(version, filename string) string {
	ver := strings.TrimPrefix(strings.TrimSpace(version), "v")
	return cnbReleasePrefix + ver + "/" + filename
}

func setManualUpdateURL(url string) {
	manualUpdateMu.Lock()
	manualUpdateURL = url
	manualUpdateMu.Unlock()
}

func takeManualUpdateURL() string {
	manualUpdateMu.Lock()
	defer manualUpdateMu.Unlock()
	return manualUpdateURL
}

func openURL(raw string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", raw)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw)
	default:
		cmd = exec.Command("xdg-open", raw)
	}
	return cmd.Start()
}
