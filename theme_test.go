package main

import (
	"strings"
	"testing"
)

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
}

func TestHarnessInitHTMLHasNoRedirect(t *testing.T) {
	if strings.Contains(harnessInitHTML, "location.") {
		t.Fatal("init HTML must not navigate; the shell SetURL's the token URL")
	}
}
