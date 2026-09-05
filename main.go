package main

import (
	"context"
	"embed"
	"log"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed index.html
//go:embed build/appicon.png
var assets embed.FS

func main() {
	initUserPATH()

	app := application.New(application.Options{
		Name:        "dsh-go",
		Description: "A Wails v3 desktop shell for DeepSeek Harness",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Closing the prep window during the handoff must not quit the app.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ui := &shellWindows{app: app}

	var (
		dsh  *DSH
		prep prepState
	)
	dsh = NewDSH(DSHConfig{
		Home: defaultHomeDir(),
		OnPrep: func(p PrepProgress) {
			prep.store(p)
			switch p.Stage {
			case "download", "unpack", "error", "update":
				app.Event.Emit(showPrepEvent)
			}
			app.Event.Emit(prepEvent, p)
		},
	}, func(dshURL string) {
		prep.setURL(dshURL)
		app.Event.Emit(openHarnessEvent, dshURL)
	})

	app.Event.On(openHarnessEvent, func(e *application.CustomEvent) {
		dshURL, _ := e.Data.(string)
		if dshURL == "" {
			return
		}
		ui.showHarness(dshURL, dsh.Source())
		go offerDSHUpdate(dsh, func(ver string) {
			if c := ui.capsule; c != nil {
				c.showDSH(ver)
			}
		})
	})

	app.Event.On(showPrepEvent, func(*application.CustomEvent) {
		if prep.readyURL() != "" {
			return
		}
		ui.showPrep()
	})

	app.Event.On(prepReadyEvent, func(*application.CustomEvent) {
		if url := prep.readyURL(); url != "" {
			return
		}
		if p, ok := prep.last(); ok {
			app.Event.Emit(prepEvent, p)
		}
	})

	var supervising atomic.Bool
	runSupervisor := func() {
		if !supervising.CompareAndSwap(false, true) {
			return
		}
		defer supervising.Store(false)
		if err := dsh.Start(ctx); err != nil {
			log.Printf("dsh supervisor stopped: %v", err)
		}
	}

	app.Event.On(prepRetryEvent, func(*application.CustomEvent) {
		if url := dsh.LastURL(); url != "" && supervising.Load() {
			app.Event.Emit(openHarnessEvent, url)
			return
		}
		go runSupervisor()
	})

	if setupUpdater(app) {
		app.Event.On(updater.EventUpdateReady, func(e *application.CustomEvent) {
			rel, ok := e.Data.(*updater.Release)
			if !ok || rel == nil {
				return
			}
			if c := ui.capsule; c != nil {
				c.show(rel.Version)
			}
		})
		go runUpdateLoop(ctx, app)
	}

	app.Event.On(capsuleEvent, func(*application.CustomEvent) {
		c := ui.capsule
		if c != nil && c.kind == capsuleKindDSH {
			go applyDSHUpdate(ctx, app, ui, dsh, c)
			return
		}
		if err := app.Updater.Restart(ctx); err != nil {
			log.Printf("update restart: %v", err)
		}
	})

	ui.showPrep()
	go runSupervisor()

	app.OnShutdown(func() {
		cancel()
		dsh.Close()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func offerDSHUpdate(dsh *DSH, show func(string)) {
	src := dsh.Source()
	if !canUpdateDSH(src.Kind) {
		return
	}
	pin := currentVersion()
	installed := installedDSHVersion(src)
	if !shouldOfferDSHUpdate(installed, pin) {
		return
	}
	show(pin)
}

func applyDSHUpdate(ctx context.Context, app *application.App, ui *shellWindows, dsh *DSH, capsule *updateCapsule) {
	src := dsh.Source()
	pin := currentVersion()
	capsule.hide()
	win := ui.showPrep()
	if win != nil {
		win.SetURL("/")
	}
	reportPrep(func(p PrepProgress) { app.Event.Emit(prepEvent, p) }, PrepProgress{
		Stage:   "update",
		Message: "正在更新 dsh 到 " + pin + "…",
	})

	var err error
	switch src.Kind {
	case sourcePath:
		err = upgradeGlobalDSH(ctx, pin)
	case sourceCache:
		err = fetchCachedRuntime(ctx, func(p PrepProgress) { app.Event.Emit(prepEvent, p) })
	default:
		err = errNoDSH
	}
	if err != nil {
		log.Printf("dsh update: %v", err)
		app.Event.Emit(prepEvent, PrepProgress{Stage: "error", Message: err.Error()})
		capsule.restoreAppIfPending()
		return
	}
	capsule.dshVer = ""
	capsule.restoreAppIfPending()
	dsh.killCurrent()
}
