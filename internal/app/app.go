package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// assetHandler serves the prep page at / and other embedded files (icon, versions)
// at their repo paths. Wails looks for index.html at the FS root, so frontend/
// is mounted there instead of exposing /frontend/index.html.
func assetHandler(assets embed.FS) http.Handler {
	frontend, err := fs.Sub(assets, "frontend")
	if err != nil {
		return application.AssetFileServerFS(assets)
	}
	page := application.AssetFileServerFS(frontend)
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/build/") {
			files.ServeHTTP(w, r)
			return
		}
		page.ServeHTTP(w, r)
	})
}

// Run starts the desktop shell.
func Run(assets embed.FS) {
	if handleAppLifecycle() {
		return
	}
	initUserPATH()
	loadStartupLocale()

	ui := &shellWindows{}
	var app *application.App
	app = application.New(application.Options{
		Name:        "dsh-go",
		Description: "A Wails v3 desktop shell for DeepSeek Harness",
		Assets: application.AssetOptions{
			Handler: assetHandler(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.dshgo.app",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if win := ui.current(); win != nil {
					win.Show()
					win.Restore()
					win.Focus()
				}
			},
		},
		RawMessageHandler: func(_ application.Window, message string, _ *application.OriginInfo) {
			handleOpenExternalMessage(app, message)
		},
	})
	ui.app = app
	registerThemeEvents(app)

	ctx, cancel := context.WithCancel(context.Background())
	offer := newUpdateCapsule()

	var (
		dsh          *DSH
		prep         prepState
		updaterReady bool
	)

	emitPrep := func(p PrepProgress) {
		prep.store(p)
		app.Event.Emit(prepEvent, p)
	}

	presentOffer := func() {
		if _, _, ok := offer.pending(); !ok {
			return
		}
		ui.showPrep()
		emitPrep(offer.progress())
	}

	showDSHUpdate := func(ver string) {
		offer.showDSH(ver)
		presentOffer()
	}

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
		AfterResolve: func(ctx context.Context, d *DSH) {
			waitStartupOffer(ctx, app, d, offer, presentOffer, emitPrep, updaterReady)
		},
	}, func(dshURL string) {
		if _, _, ok := offer.pending(); ok {
			return
		}
		app.Event.Emit(openHarnessEvent, dshURL)
	})

	app.Event.On(openHarnessEvent, func(e *application.CustomEvent) {
		dshURL, _ := e.Data.(string)
		if dshURL == "" {
			return
		}
		ui.showHarness(dshURL)
	})

	app.Event.On(showPrepEvent, func(*application.CustomEvent) {
		ui.showPrep()
	})

	app.Event.On(prepReadyEvent, func(*application.CustomEvent) {
		app.Event.Emit(themePrefEvent, lockedThemePreference())
		app.Event.Emit(localePrefEvent, loadStartupLocale())
		if _, _, ok := offer.pending(); ok {
			app.Event.Emit(prepEvent, offer.progress())
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
		if _, _, ok := offer.pending(); ok {
			go applyAllUpdates(ctx, app, emitPrep, presentOffer, dsh, offer)
			return
		}
		if url := dsh.LastURL(); url != "" && supervising.Load() {
			app.Event.Emit(openHarnessEvent, url)
			return
		}
		go runSupervisor()
	})

	updaterReady = setupUpdater(app)
	if updaterReady {
		app.Event.On(updater.EventUpdateReady, func(e *application.CustomEvent) {
			rel, ok := e.Data.(*updater.Release)
			if !ok || rel == nil {
				return
			}
			offer.stashApp(rel.Version)
			if _, _, pending := offer.pending(); pending && !offer.isDSH() {
				presentOffer()
			}
		})
		app.Event.On(manualUpdateEvt, func(e *application.CustomEvent) {
			ver, _ := e.Data.(string)
			if ver == "" {
				return
			}
			offer.stashApp(ver)
			presentOffer()
		})
		go runUpdateLoop(ctx, app)
	}
	go runDSHUpdateLoop(ctx, dsh, showDSHUpdate)

	app.Event.On(prepApplyEvent, func(*application.CustomEvent) {
		go applyAllUpdates(ctx, app, emitPrep, presentOffer, dsh, offer)
	})

	app.Event.On(prepLaterEvent, func(*application.CustomEvent) {
		offer.proceed()
		if url := dsh.LastURL(); url != "" {
			app.Event.Emit(openHarnessEvent, url)
		}
	})

	prep.store(prepStartProgress())
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

func waitStartupOffer(
	ctx context.Context,
	app *application.App,
	dsh *DSH,
	offer *updateCapsule,
	present func(),
	emit func(PrepProgress),
	updaterReady bool,
) {
	emit(prepStartProgress())
	simulateUpdates(offer)
	if !updateSimulationActive() {
		checkCtx, cancel := context.WithTimeout(ctx, startupOfferTimeout)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, _, ok := offer.pending(); !ok {
				offerDSHUpdate(checkCtx, dsh, offer.showDSH)
			}
		}()
		go func() {
			defer wg.Done()
			if !updaterReady {
				return
			}
			if ver := checkShellUpdate(checkCtx, app); ver != "" {
				offer.stashApp(ver)
			}
		}()
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-checkCtx.Done():
		}
		cancel()
	}
	if _, _, ok := offer.pending(); !ok {
		return
	}
	present()
	offer.wait(ctx)
}

func offerDSHUpdate(ctx context.Context, dsh *DSH, show func(string)) {
	src := dsh.Source()
	if !canUpdateDSH(src.Kind) {
		return
	}
	installed := installedDSHVersion(src)
	if installed == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, dshCheckTimeout)
	defer cancel()
	target := targetDSHVersion(ctx)
	if !shouldOfferDSHUpdate(installed, target) {
		return
	}
	show(target)
}

func applyAllUpdates(ctx context.Context, app *application.App, emit func(PrepProgress), present func(), dsh *DSH, offer *updateCapsule) {
	if offer.dshVersion() != "" {
		if err := applyDSHUpdateNow(ctx, emit, dsh, offer.dshVersion()); err != nil {
			log.Printf("dsh update: %v", err)
			offer.showDSH(offer.dshVersion())
			p := offer.progress()
			p.Message = err.Error()
			emit(p)
			return
		}
		if dsh.LastURL() != "" {
			dsh.killCurrent()
		}
		offer.clearDSH()
	}
	if offer.appVersion() != "" {
		emit(PrepProgress{Stage: "update", Message: currentUI().SelectingSource})
		path, err := downloadDesktopUpdate(ctx, offer.appVersion(), emit)
		if err != nil {
			log.Printf("client update: %v", err)
			offer.restoreAppIfPending()
			emit(PrepProgress{Stage: "error", Message: err.Error()})
			present()
			return
		}
		emit(PrepProgress{Stage: "update", Message: currentUI().InstallingClient})
		if err := installDesktopPackage(path); err != nil {
			log.Printf("client install: %v", err)
			offer.restoreAppIfPending()
			emit(PrepProgress{Stage: "error", Message: err.Error()})
			present()
			return
		}
		if desktopInstallRestarts() {
			if app != nil {
				app.Quit()
			}
			return
		}
		offer.clearApp()
	}
	offer.proceed()
}

func applyDSHUpdateNow(ctx context.Context, emit func(PrepProgress), dsh *DSH, target string) error {
	if target == "" {
		target = targetDSHVersion(ctx)
	}
	emit(PrepProgress{
		Stage:   "update",
		Message: fmt.Sprintf(currentUI().UpdatingRuntime, target),
	})
	switch dsh.Source().Kind {
	case sourcePath:
		return upgradeGlobalDSH(ctx, target)
	case sourceCache, sourceBundled:
		return fetchCachedRuntime(ctx, emit)
	default:
		return errNoDSH
	}
}
