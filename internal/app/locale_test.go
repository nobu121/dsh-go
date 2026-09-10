package app

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func resetLocaleState(t *testing.T) {
	t.Helper()
	localeOnce = sync.Once{}
	localePref = ""
	t.Cleanup(func() {
		localeOnce = sync.Once{}
		localePref = ""
	})
}

func writeLocalePreference(t *testing.T, pref string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("DSH_HOME", home)
	if pref == "" {
		return
	}
	if err := os.WriteFile(settingsYAMLPath(), []byte("locale:\n  preference: "+pref+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseLocalePreference(t *testing.T) {
	raw := "ui-theme:\n  preference: dark\nlocale:\n  preference: en\n"
	if got := parseLocalePreference(raw); got != localeEN {
		t.Fatalf("got %q", got)
	}
	if got := parseLocalePreference("locale:\n  preference: zh\n"); got != localeZH {
		t.Fatalf("got %q", got)
	}
	if got := parseLocalePreference("locale:\n  preference: en-US\n"); got != localeEN {
		t.Fatalf("got %q", got)
	}
	if got := parseLocalePreference("other: 1\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadStartupLocaleReadsOnce(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "en")
	if got := loadStartupLocale(); got != localeEN {
		t.Fatalf("got %q", got)
	}
	if err := os.WriteFile(settingsYAMLPath(), []byte("locale:\n  preference: zh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadStartupLocale(); got != localeEN {
		t.Fatalf("startup locale must stay %q, got %q", localeEN, got)
	}
}

func TestLoadStartupLocaleDefaultsToChinese(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "")
	if got := loadStartupLocale(); got != localeZH {
		t.Fatalf("got %q", got)
	}
}

func TestCurrentUIFollowsStartupLocale(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "en")
	if got := currentUI().UpdateNow; got != uiEN.UpdateNow {
		t.Fatalf("got %q", got)
	}

	resetLocaleState(t)
	writeLocalePreference(t, "zh")
	if got := currentUI().UpdateNow; got != uiZH.UpdateNow {
		t.Fatalf("got %q", got)
	}
}

func TestPrepPageURLFollowsStartupLocale(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "en")
	if got := prepPageURL(); got != "/?lang=en" {
		t.Fatalf("got %q", got)
	}

	resetLocaleState(t)
	writeLocalePreference(t, "zh")
	if got := prepPageURL(); got != "/" {
		t.Fatalf("got %q", got)
	}
}

func TestPrepPageHasBothLanguages(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{"正在初始化", "发现新版本", "立即更新", "稍后", "无法启动", "Harness 插件加载失败", "禁用并重启", "仅重启", "查看报错", "收起报错", "Starting", "Update available", "Update now", "Later", "Could not start", "Harness failed to load plugins", "Disable and restart", "Restart only", "View error", "Hide error"} {
		if !strings.Contains(page, want) {
			t.Fatalf("prep page missing %q", want)
		}
	}
}
