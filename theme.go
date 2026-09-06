package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	themeDarkEvent  = "dsh-go:theme-dark"
	themeLightEvent = "dsh-go:theme-light"
	themePrefEvent  = "dsh-go:theme-pref"
)

// themeWatchJS observes harness ui-theme's body[data-ds-dark-theme] and pushes
// bare-name events to the shell. No host-side polling.
//
// Emit goes through chrome.webview / webkit messageHandlers directly so it
// works before window._wails.invoke is wired on remote pages.
var themeWatchJS = `(function(){
  if (window.__dshGoThemeWatch) return;
  window.__dshGoThemeWatch = true;
  function isDark(){
    var body = document.body;
    if (!body) return null;
    return body.hasAttribute("data-ds-dark-theme");
  }
  function emit(dark){
    var name = dark ? "dsh-go:theme-dark" : "dsh-go:theme-light";
    var msg = "wails:event:emit:" + name;
    try {
      if (window._wails && typeof window._wails.invoke === "function") {
        window._wails.invoke(msg);
        return;
      }
    } catch (e) {}
    try {
      if (window.chrome && window.chrome.webview && typeof window.chrome.webview.postMessage === "function") {
        window.chrome.webview.postMessage(msg);
        return;
      }
    } catch (e2) {}
    try { window.webkit.messageHandlers.external.postMessage(msg); } catch (e3) {}
  }
  var last;
  function report(){
    var dark = isDark();
    if (dark === null || dark === last) return;
    last = dark;
    emit(dark);
  }
  function bind(el){
    if (!el || el.__dshGoThemeObs) return;
    el.__dshGoThemeObs = true;
    new MutationObserver(report).observe(el, {attributes:true, attributeFilter:["data-ds-dark-theme"]});
  }
  bind(document.body);
  new MutationObserver(function(){ bind(document.body); report(); }).observe(document.documentElement, {childList:true, subtree:true});
  try { matchMedia("(prefers-color-scheme: dark)").addEventListener("change", report); } catch (e) {}
  report();
})();`

var (
	themeListenerMu sync.Mutex
	themeListener   func(bool)
	themeEventsOnce sync.Once
)

func registerThemeEvents(app *application.App) {
	themeEventsOnce.Do(func() {
		app.Event.On(themeDarkEvent, func(*application.CustomEvent) { onThemeDetected(true) })
		app.Event.On(themeLightEvent, func(*application.CustomEvent) { onThemeDetected(false) })
	})
}

func setThemeListener(fn func(bool)) {
	themeListenerMu.Lock()
	themeListener = fn
	themeListenerMu.Unlock()
}

func onThemeDetected(dark bool) {
	themeListenerMu.Lock()
	fn := themeListener
	themeListenerMu.Unlock()
	if fn != nil {
		fn(dark)
	}
}

var (
	themeMu    sync.Mutex
	themeDark  bool
	themeKnown bool
)

func settingsYAMLPath() string {
	return filepath.Join(defaultHomeDir(), "settings.yaml")
}

func parseUIThemePreference(raw string) string {
	in := false
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "ui-theme:" {
			in = true
			continue
		}
		if !in {
			continue
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		if !indented && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			in = false
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok || strings.TrimSpace(key) != "preference" {
			continue
		}
		pref := strings.Trim(strings.TrimSpace(val), `"'`)
		switch pref {
		case "light", "dark", "system":
			return pref
		}
	}
	return ""
}

// lockedThemePreference is the user's explicit ui-theme lock.
// Missing, empty, or "system" means follow the OS — not the last-seen chrome.
func lockedThemePreference() string {
	b, err := os.ReadFile(settingsYAMLPath())
	if err != nil {
		return "system"
	}
	switch parseUIThemePreference(string(b)) {
	case "dark":
		return "dark"
	case "light":
		return "light"
	default:
		return "system"
	}
}

func knownThemeDark() bool {
	themeMu.Lock()
	defer themeMu.Unlock()
	if themeKnown {
		return themeDark
	}
	switch lockedThemePreference() {
	case "dark":
		themeDark = true
	case "light":
		themeDark = false
	default:
		themeDark = systemDark()
	}
	themeKnown = true
	return themeDark
}

func rememberTheme(dark bool) {
	themeMu.Lock()
	themeDark = dark
	themeKnown = true
	themeMu.Unlock()
}

func themeChanged(dark bool) bool {
	themeMu.Lock()
	same := themeKnown && themeDark == dark
	themeMu.Unlock()
	rememberTheme(dark)
	return !same
}

func themeBackground(dark bool) application.RGBA {
	if dark {
		return application.NewRGB(6, 7, 15)
	}
	return application.NewRGB(255, 255, 255)
}

func macChrome(dark bool) application.MacWindow {
	appearance := application.NSAppearanceNameAqua
	if dark {
		appearance = application.NSAppearanceNameDarkAqua
	}
	return application.MacWindow{
		TitleBar:   application.MacTitleBarDefault,
		Appearance: appearance,
	}
}

func windowsTheme() application.Theme {
	switch lockedThemePreference() {
	case "dark":
		return application.Dark
	case "light":
		return application.Light
	default:
		return application.SystemDefault
	}
}

func applyChrome(win *application.WebviewWindow, dark bool) {
	if win == nil {
		return
	}
	win.SetBackgroundColour(themeBackground(dark))
	// DWM title-bar attributes must be set on the UI thread; a direct call
	// from an event goroutine leaves the caption stuck on the OS theme.
	application.InvokeAsync(func() {
		applyNativeChrome(win, dark)
	})
}

// harnessInitHTML exists so Windows registers Options.JS as a WebView2 Init
// script. The shell then SetURL's the harness address (host-initiated), which
// keeps dsh's SameSite=Strict launch cookie working. Do not location.replace
// from this page — that is a cross-site hop and gets a 401.
const harnessInitHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body></body></html>`
