package app

import (
	"strings"
	"testing"
)

func TestParseOpenExternalMessage(t *testing.T) {
	got, ok := parseOpenExternalMessage("dsh-go:open:https://example.com/a")
	if !ok || got != "https://example.com/a" {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := parseOpenExternalMessage("wails:event:emit:other"); ok {
		t.Fatal("must ignore unrelated messages")
	}
	if _, ok := parseOpenExternalMessage("dsh-go:open:javascript:alert(1)"); ok {
		t.Fatal("must reject javascript URLs")
	}
	if _, ok := parseOpenExternalMessage("dsh-go:open:file:///tmp/x"); ok {
		t.Fatal("must reject file URLs")
	}
	if _, ok := parseOpenExternalMessage("dsh-go:open:mailto:a@b.c"); !ok {
		t.Fatal("mailto should open in the system handler")
	}
}

func TestShouldOpenExternally(t *testing.T) {
	if !shouldOpenExternally("https://github.com/x") {
		t.Fatal("https")
	}
	if shouldOpenExternally("about:blank") {
		t.Fatal("about:blank")
	}
	if shouldOpenExternally("") {
		t.Fatal("empty")
	}
}

func TestExternalLinkJSInterceptsClicksAndWindowOpen(t *testing.T) {
	if !strings.Contains(externalLinkJS, `closest("a[href]")`) {
		t.Fatal("must intercept anchor clicks")
	}
	if !strings.Contains(externalLinkJS, `window.open`) {
		t.Fatal("must override window.open")
	}
	if !strings.Contains(externalLinkJS, `dsh-go:open:`) {
		t.Fatal("must post a raw open message")
	}
	if !strings.Contains(externalLinkJS, `chrome.webview.postMessage`) {
		t.Fatal("must post via WebView2 on Windows")
	}
	if strings.Contains(externalLinkJS, "wails:event:emit:") {
		t.Fatal("simple emit cannot carry a URL")
	}
}

func TestHarnessInitJSIncludesThemeAndLinks(t *testing.T) {
	js := harnessInitJS()
	if !strings.Contains(js, "__dshGoThemeWatch") || !strings.Contains(js, "__dshGoExtLinks") {
		t.Fatal("harness init JS must include theme watch and external links")
	}
	if !strings.Contains(js, "__dshGoBootFail") {
		t.Fatal("harness init JS must include boot-fail watch")
	}
}
