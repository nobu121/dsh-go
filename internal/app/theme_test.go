package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func resetThemeState(t *testing.T) {
	t.Helper()
	themeMu.Lock()
	themeKnown = false
	themeDark = false
	themeMu.Unlock()
	t.Cleanup(func() {
		themeMu.Lock()
		themeKnown = false
		themeDark = false
		themeMu.Unlock()
	})
}

func writeThemePreference(t *testing.T, pref string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("DSH_HOME", home)
	if pref == "" {
		return
	}
	if err := os.WriteFile(settingsYAMLPath(), []byte("ui-theme:\n  preference: "+pref+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseUIThemePreference(t *testing.T) {
	raw := "ui-onboarding:\n  welcomeNoticeVersion: x\nui-theme:\n  preference: dark\n"
	if got := parseUIThemePreference(raw); got != "dark" {
		t.Fatalf("got %q", got)
	}
	if got := parseUIThemePreference("ui-theme:\n  preference: system\n"); got != "system" {
		t.Fatalf("got %q", got)
	}
	if got := parseUIThemePreference("other: 1\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestLockedThemePreferenceDefaultsToSystem(t *testing.T) {
	writeThemePreference(t, "")
	if got := lockedThemePreference(); got != "system" {
		t.Fatalf("got %q", got)
	}
}

func TestLockedThemePreferenceReadsSettings(t *testing.T) {
	writeThemePreference(t, "dark")
	if got := lockedThemePreference(); got != "dark" {
		t.Fatalf("got %q", got)
	}
	writeThemePreference(t, "system")
	if got := lockedThemePreference(); got != "system" {
		t.Fatalf("got %q", got)
	}
}

func TestWindowsThemeFollowsLock(t *testing.T) {
	writeThemePreference(t, "light")
	if got := windowsTheme(); got != application.Light {
		t.Fatalf("locked light = %v", got)
	}
	writeThemePreference(t, "dark")
	if got := windowsTheme(); got != application.Dark {
		t.Fatalf("locked dark = %v", got)
	}
	writeThemePreference(t, "")
	if got := windowsTheme(); got != application.SystemDefault {
		t.Fatalf("unlocked = %v", got)
	}
}

func TestCurrentThemeDarkFollowsLock(t *testing.T) {
	writeThemePreference(t, "light")
	if currentThemeDark() {
		t.Fatal("locked light")
	}
	writeThemePreference(t, "dark")
	if !currentThemeDark() {
		t.Fatal("locked dark")
	}
	writeThemePreference(t, "system")
	if currentThemeDark() != systemDark() {
		t.Fatal("system should follow the OS")
	}
}

func TestKnownThemeFollowsLockOrSystem(t *testing.T) {
	resetThemeState(t)
	writeThemePreference(t, "light")
	if knownThemeDark() {
		t.Fatal("locked light should not be dark")
	}

	resetThemeState(t)
	writeThemePreference(t, "")
	if knownThemeDark() != systemDark() {
		t.Fatal("unlocked theme should follow the OS")
	}
}

func TestPrepPageShowsInitAndFollowsSystem(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if !strings.Contains(page, "Deepseek Harness GO") {
		t.Fatal("prep page must show start copy when a runtime is already available")
	}
	if !strings.Contains(page, "\u6b63\u5728\u4e0b\u8f7d") || !strings.Contains(page, "\u6b63\u5728\u5b89\u88c5") {
		t.Fatal("prep page must show download and install percent under the progress bar")
	}
	if !strings.Contains(page, "\u6b63\u5728\u521d\u59cb\u5316") {
		t.Fatal("prep page must fall back to init copy when progress is unknown")
	}
	if !strings.Contains(page, `data-theme="system"`) {
		t.Fatal("prep page must default to system theme")
	}
	if !strings.Contains(page, "prefers-color-scheme") {
		t.Fatal("prep page must follow OS colors until locked")
	}
}

func TestThemeWatchJSSkipsNullBody(t *testing.T) {
	if !strings.Contains(themeWatchJS, `if (dark === null || dark === last) return`) {
		t.Fatal("must not emit before body exists")
	}
	if !strings.Contains(themeWatchJS, `chrome.webview.postMessage`) {
		t.Fatal("must emit via WebView2 postMessage on Windows")
	}
	if !strings.Contains(themeWatchJS, `webkit.messageHandlers.external.postMessage`) {
		t.Fatal("must emit via WKWebView messageHandlers on macOS")
	}
	if strings.Contains(themeWatchJS, `setInterval`) || strings.Contains(themeWatchJS, `setTimeout`) {
		t.Fatal("theme watch must be event-driven, not polled")
	}
	host, err := os.ReadFile("theme.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(host), "NewTicker") || strings.Contains(string(host), "time.Tick") {
		t.Fatal("host must not poll theme settings")
	}
}

// Windows runs Options.JS as a document-created script, before the HTML is
// parsed. observe(null) there throws and silently kills the watcher, so the
// title bar never hears about a theme change.
func TestThemeWatchJSSurvivesEmptyDocument(t *testing.T) {
	if strings.Contains(themeWatchJS, `.observe(document.documentElement`) {
		t.Fatal("must not observe documentElement unguarded; it is null in a document-created script")
	}
	if !strings.Contains(themeWatchJS, `if (!el || el.__dshGoThemeObs) return false`) {
		t.Fatal("observe targets must be nil-guarded")
	}
	for _, retry := range []string{"readystatechange", "DOMContentLoaded"} {
		if !strings.Contains(themeWatchJS, retry) {
			t.Fatalf("setup must retry on %s once the document exists", retry)
		}
	}
}

func TestHarnessInitHTMLHasNoRedirect(t *testing.T) {
	if strings.Contains(harnessInitHTML, "location.") {
		t.Fatal("init HTML must not navigate; the shell SetURL's the token URL")
	}
}
