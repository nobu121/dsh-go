package app

import (
	"context"
	"sync"
)

const (
	capsuleKindApp = "app"
	capsuleKindDSH = "dsh"
)

// updateCapsule holds a pending update offer shown on the shell prep page.
type updateCapsule struct {
	mu     sync.Mutex
	kind   string
	appVer string
	dshVer string
	ready  bool
	done   chan struct{}
}

func newUpdateCapsule() *updateCapsule {
	return &updateCapsule{done: make(chan struct{})}
}

func (c *updateCapsule) show(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kind = capsuleKindApp
	c.appVer = version
	c.ready = true
}

func (c *updateCapsule) showDSH(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kind = capsuleKindDSH
	c.dshVer = version
	c.ready = true
}

func (c *updateCapsule) stashApp(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.appVer = version
	if !c.ready {
		c.kind = capsuleKindApp
		c.ready = true
	}
}

func (c *updateCapsule) hide() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = false
	c.kind = ""
}

func (c *updateCapsule) restoreAppIfPending() {
	c.mu.Lock()
	ver := c.appVer
	c.mu.Unlock()
	if ver == "" {
		c.hide()
		return
	}
	c.show(ver)
}

func (c *updateCapsule) pending() (kind, version string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready {
		return "", "", false
	}
	if c.kind == capsuleKindDSH {
		return c.kind, c.dshVer, true
	}
	return c.kind, c.appVer, true
}

func (c *updateCapsule) isDSH() bool {
	kind, _, ok := c.pending()
	return ok && kind == capsuleKindDSH
}

func (c *updateCapsule) dshVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dshVer
}

func (c *updateCapsule) appVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.appVer
}

func (c *updateCapsule) clearDSH() {
	c.mu.Lock()
	c.dshVer = ""
	if c.kind == capsuleKindDSH {
		c.kind = ""
		c.ready = false
	}
	c.mu.Unlock()
}

func (c *updateCapsule) clearApp() {
	c.mu.Lock()
	c.appVer = ""
	if c.kind == capsuleKindApp {
		c.kind = ""
		c.ready = false
	}
	c.mu.Unlock()
}

func (c *updateCapsule) progress() PrepProgress {
	kind, version, ok := c.pending()
	if !ok {
		return PrepProgress{}
	}
	c.mu.Lock()
	appVer, dshVer := c.appVer, c.dshVer
	c.mu.Unlock()

	ui := currentUI()
	var items []PrepOfferItem
	if appVer != "" {
		items = append(items, PrepOfferItem{Kind: capsuleKindApp, Name: ui.Client, Version: appVer})
	}
	if kind == capsuleKindDSH && dshVer != "" {
		items = append(items, PrepOfferItem{Kind: capsuleKindDSH, Name: ui.Runtime, Version: dshVer})
	}

	msg := ui.OfferApp
	if kind == capsuleKindDSH && appVer != "" {
		msg = ui.OfferBoth
	} else if kind == capsuleKindDSH {
		msg = ui.OfferDSH
	}
	return PrepProgress{
		Stage:   prepOfferStage,
		Message: msg,
		Action:  ui.UpdateNow,
		Version: version,
		Items:   items,
	}
}

func (c *updateCapsule) proceed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = false
	c.kind = ""
	if c.done == nil {
		return
	}
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *updateCapsule) wait(ctx context.Context) {
	c.mu.Lock()
	ch := c.done
	c.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case <-ch:
	case <-ctx.Done():
	}
}
