package main

import (
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed capsule.html
var capsuleHTML string

const (
	capsuleWidth  = 188
	capsuleHeight = 26
	capsuleEvent  = "dsh-go:apply-update"
	capsuleReady  = "dsh-go:capsule-ready"
	capsuleLabel  = "dsh-go:capsule-label"
)

type updateCapsule struct {
	app    *application.App
	parent *application.WebviewWindow
	win    *application.WebviewWindow
	label  string
	ready  bool
}

func newUpdateCapsule(app *application.App, parent *application.WebviewWindow) *updateCapsule {
	c := &updateCapsule{app: app, parent: parent}
	c.win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                 "update-capsule",
		Title:                "",
		Width:                capsuleWidth,
		Height:               capsuleHeight,
		MinWidth:             capsuleWidth,
		MinHeight:            capsuleHeight,
		MaxWidth:             capsuleWidth,
		MaxHeight:            capsuleHeight,
		Frameless:            true,
		DisableResize:        true,
		Hidden:               true,
		BackgroundType:       application.BackgroundTypeTransparent,
		BackgroundColour:     application.NewRGBA(0, 0, 0, 0),
		HTML:                 capsuleHTML,
		AllowSimpleEventEmit: true,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				Hide:               true,
				HideTitle:          true,
			},
		},
	})
	parent.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) { c.reposition() })
	parent.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) { c.reposition() })
	parent.OnWindowEvent(events.Common.WindowFocus, func(*application.WindowEvent) { c.syncVisibility() })
	parent.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) { c.win.Hide() })
	parent.OnWindowEvent(events.Common.WindowMinimise, func(*application.WindowEvent) { c.win.Hide() })
	app.Event.On(capsuleReady, func(*application.CustomEvent) {
		if c.label != "" {
			app.Event.Emit(capsuleLabel, c.label)
		}
	})
	return c
}

func capsuleLabelText(version string) string {
	return "更新到 " + version
}

func (c *updateCapsule) show(version string) {
	c.label = capsuleLabelText(version)
	c.ready = true
	c.app.Event.Emit(capsuleLabel, c.label)
	c.reposition()
	c.syncVisibility()
	log.Printf("update ready: %s", version)
}

func (c *updateCapsule) syncVisibility() {
	if !c.ready || c.parent == nil {
		return
	}
	if c.parent.IsMinimised() || !c.parent.IsVisible() {
		c.win.Hide()
		return
	}
	c.win.Show()
}

func (c *updateCapsule) reposition() {
	if c.parent == nil || c.win == nil {
		return
	}
	x, y := c.parent.Position()
	w, _ := c.parent.Size()
	c.win.SetPosition(x+(w-capsuleWidth)/2, y+6)
}
