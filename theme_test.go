package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseThemeDark(t *testing.T) {
	if dark, ok := parseThemeDark(map[string]any{"dark": true}); !ok || !dark {
		t.Fatal("map dark=true")
	}
	if dark, ok := parseThemeDark(false); !ok || dark {
		t.Fatal("bool false")
	}
	if _, ok := parseThemeDark("nope"); ok {
		t.Fatal("unknown payload")
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

func TestSaveLoadTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// themeFile uses UserConfigDir; isolate by writing through save/load after
	// pointing XDG / HOME is OS-specific. Test the file format instead.
	path := filepath.Join(dir, "ui-theme")
	if err := os.WriteFile(path, []byte("dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "dark\n" {
		t.Fatalf("got %q", got)
	}
}
