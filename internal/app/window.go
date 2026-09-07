package app

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	openHarnessEvent = "dsh-go:open-harness"
	showPrepEvent    = "dsh-go:show-prep"
)

type shellWindows struct {
	app     *application.App
	mu      sync.Mutex
	win     *application.WebviewWindow
	prep    *application.WebviewWindow
	harness *application.WebviewWindow
}

func (s *shellWindows) showPrep() *application.WebviewWindow {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.harness != nil {
		s.harness.Hide()
	}
	if s.prep != nil {
		s.prep.Show()
		s.win = s.prep
		setThemeTarget(func() *application.WebviewWindow { return s.current() })
		return s.prep
	}
	s.prep = newPrepWindow(s.app)
	s.win = s.prep
	setThemeTarget(func() *application.WebviewWindow { return s.current() })
	s.prep.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		s.mu.Lock()
		closingPrep := s.win == s.prep
		s.mu.Unlock()
		if closingPrep {
			s.app.Quit()
		}
	})
	return s.prep
}

func (s *shellWindows) current() *application.WebviewWindow {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.win
}

func (s *shellWindows) showHarness(dshURL string) *application.WebviewWindow {
	s.mu.Lock()
	if s.harness != nil {
		h := s.harness
		prep := s.prep
		s.win = h
		s.mu.Unlock()
		h.SetURL(dshURL)
		h.Show()
		if prep != nil {
			prep.Hide()
		}
		setThemeTarget(func() *application.WebviewWindow { return s.current() })
		return h
	}
	from := s.win
	prep := s.prep
	s.mu.Unlock()

	log.Printf("opening harness window")
	next := newHarnessWindow(s.app, dshURL, from)
	setThemeTarget(func() *application.WebviewWindow { return s.current() })
	setThemeListener(func(dark bool) {
		rememberTheme(dark)
		applyChrome(next, dark)
	})
	next.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		s.app.Quit()
	})

	s.mu.Lock()
	s.win = next
	s.harness = next
	s.mu.Unlock()

	var shown atomic.Bool
	var finishes atomic.Int32
	reveal := func() {
		if !shown.CompareAndSwap(false, true) {
			return
		}
		next.Show()
		applyChrome(next, knownThemeDark())
		if prep != nil {
			prep.Hide()
		}
	}
	// Init HTML + host SetURL(/?token=) + 303 to / .
	onNav := func(*application.WindowEvent) {
		if finishes.Add(1) >= 2 {
			reveal()
		}
	}
	next.OnWindowEvent(events.Mac.WebViewDidFinishNavigation, onNav)
	next.OnWindowEvent(events.Windows.WebViewNavigationCompleted, onNav)
	time.AfterFunc(2*time.Second, reveal)
	return next
}

func newPrepWindow(app *application.App) *application.WebviewWindow {
	dark := knownThemeDark()
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                "DeepSeek Harness",
		Width:                1280,
		Height:               800,
		URL:                  "/",
		AllowSimpleEventEmit: true,
		Mac:                  macChrome(dark),
		Windows: application.WindowsWindow{
			Theme:       windowsTheme(),
			CustomTheme: windowsCustomTheme(),
		},
		BackgroundColour: themeBackground(dark),
	})
	applyNativeChrome(win, dark)
	return win
}

func newHarnessWindow(app *application.App, dshURL string, from *application.WebviewWindow) *application.WebviewWindow {
	dark := knownThemeDark()
	opts := application.WebviewWindowOptions{
		Title:                "DeepSeek Harness",
		Width:                1280,
		Height:               800,
		HTML:                 harnessInitHTML,
		Hidden:               true,
		JS:                   themeWatchJS,
		AllowSimpleEventEmit: true,
		Mac:                  macChrome(dark),
		Windows: application.WindowsWindow{
			Theme:       windowsTheme(),
			CustomTheme: windowsCustomTheme(),
		},
		BackgroundColour: themeBackground(dark),
	}
	if from != nil {
		opts.Width, opts.Height = from.Size()
	}
	win := app.Window.NewWithOptions(opts)
	if from != nil {
		x, y := from.Position()
		win.SetPosition(x, y)
	}
	win.SetURL(dshURL)
	applyNativeChrome(win, dark)
	return win
}
