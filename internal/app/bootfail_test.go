package app

import (
	"strings"
	"testing"
)

func TestParseBootFailMessage(t *testing.T) {
	text := "HARNESS\nFailed to load plugins\ndsh-wsl-workspace"
	got, ok := parseBootFailMessage(bootFailPrefix + text)
	if !ok || got != text {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := parseBootFailMessage("dsh-go:open:https://example.com"); ok {
		t.Fatal("must ignore unrelated messages")
	}
	if _, ok := parseBootFailMessage(bootFailPrefix + "Loading plugins…"); ok {
		t.Fatal("must ignore the loading state")
	}
}

func TestBootFailWatchJSRecognizesOfficialPage(t *testing.T) {
	if !strings.Contains(bootFailWatchJS, `[data-dsh-boot]`) {
		t.Fatal("must look for the official boot root")
	}
	if !strings.Contains(bootFailWatchJS, `Failed to load plugins`) {
		t.Fatal("must recognize the official failure title")
	}
	if !strings.Contains(bootFailWatchJS, bootFailPrefix) {
		t.Fatal("must post a raw boot-fail message")
	}
	if strings.Contains(bootFailWatchJS, `setInterval`) || strings.Contains(bootFailWatchJS, `setTimeout`) {
		t.Fatal("boot-fail watch must be event-driven, not polled")
	}
	if !strings.Contains(bootFailWatchJS, `if (!el || el.__dshGoBootObs) return false`) {
		t.Fatal("observe targets must be nil-guarded")
	}
	for _, retry := range []string{"readystatechange", "DOMContentLoaded"} {
		if !strings.Contains(bootFailWatchJS, retry) {
			t.Fatalf("setup must retry on %s once the document exists", retry)
		}
	}
}
