package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed index.html
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "dsh-go",
		Description: "A Wails v3 desktop shell for DeepSeek Harness",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	readyURL := make(chan string, 1)
	dsh := NewDSH(DSHConfig{Home: defaultHomeDir()}, func(url string) {
		select {
		case readyURL <- url:
		default:
		}
	})

	go func() {
		if err := dsh.Start(ctx); err != nil {
			log.Printf("dsh supervisor stopped: %v", err)
		}
	}()

	dshURL := <-readyURL
	mainWin := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "DeepSeek Harness",
		Width:  1280,
		Height: 800,
		URL:    dshURL,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
	})

	capsule := newUpdateCapsule(app, mainWin)
	if setupUpdater(app) {
		app.Event.On(updater.EventUpdateReady, func(e *application.CustomEvent) {
			rel, ok := e.Data.(*updater.Release)
			if !ok || rel == nil {
				return
			}
			capsule.show(rel.Version)
		})
		app.Event.On(capsuleEvent, func(*application.CustomEvent) {
			if err := app.Updater.Restart(ctx); err != nil {
				log.Printf("update restart: %v", err)
			}
		})
		go runUpdateLoop(ctx, app)
	}

	app.OnShutdown(func() {
		cancel()
		dsh.Close()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
