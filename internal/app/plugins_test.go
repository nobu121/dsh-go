package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWebProfile(t *testing.T, home string, manifest string) {
	t.Helper()
	dir := filepath.Join(home, "profiles", "web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleWebProfile = `{
  "name": "dsh-profile-web",
  "private": true,
  "dependencies": {
    "dsh-wsl-workspace": "^0.4.0",
    "dshmarket": "^1.45.1"
  },
  "dsh": {
    "profile": {
      "bundles": [
        "@deepseek-ai/dsh-base",
        "@deepseek-ai/dsh-web-app",
        "dshmarket",
        "dsh-wsl-workspace"
      ],
      "patchReload": "live"
    }
  }
}
`

func TestListUserPluginsSkipsOfficial(t *testing.T) {
	home := t.TempDir()
	writeWebProfile(t, home, sampleWebProfile)
	got := listUserPlugins(home)
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got[0].Name != "dshmarket" || got[1].Name != "dsh-wsl-workspace" {
		t.Fatalf("order = %#v", got)
	}
	if got[1].Version != "^0.4.0" {
		t.Fatalf("version = %q", got[1].Version)
	}
}

func TestDisableWritesPatchLayerNotBundles(t *testing.T) {
	home := t.TempDir()
	writeWebProfile(t, home, sampleWebProfile)
	dir := filepath.Join(home, "profiles", "web")
	if err := os.WriteFile(filepath.Join(dir, "cordis.patch.yml"), []byte("# header\n[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := filepath.Join(dir, "node_modules", "dsh-wsl-workspace")
	if err := os.MkdirAll(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "cordis.patch.yml"), []byte("- insert:\n    - id: wsl-workspace\n      name: dsh-wsl-workspace\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disableProfileBundles(home, []string{"dsh-wsl-workspace", "@deepseek-ai/dsh-base"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(webProfileManifestPath(home))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if !containsString(profileBundles(doc), "dsh-wsl-workspace") {
		t.Fatal("bundle membership must stay")
	}
	if _, err := os.Stat(filepath.Join(dir, ".dsh-market", "state.json")); !os.IsNotExist(err) {
		t.Fatal("must not write market state")
	}
	patch, err := os.ReadFile(webProfilePatchPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patch), "- id: wsl-workspace\n  disabled: true\n") {
		t.Fatalf("patch = %s", patch)
	}
}

func TestRecoverPrepProgressSelectsBlamedPlugin(t *testing.T) {
	resetLocaleState(t)
	writeLocalePreference(t, "en")
	_ = loadStartupLocale()
	home := t.TempDir()
	writeWebProfile(t, home, sampleWebProfile)
	p := recoverPrepProgress(home, "Failed to load plugins\ndsh-wsl-workspace")
	if p.Stage != prepRecoverStage {
		t.Fatalf("stage = %s", p.Stage)
	}
	if p.Action != uiEN.DisableRestart {
		t.Fatalf("action = %q", p.Action)
	}
	if len(p.Items) != 2 {
		t.Fatalf("items = %#v", p.Items)
	}
	if !p.Items[0].Selected || p.Items[0].Name != "dsh-wsl-workspace" {
		t.Fatalf("blamed plugin must be first, got %#v", p.Items)
	}
	if p.Items[1].Selected {
		t.Fatalf("other plugins must stay unchecked, got %#v", p.Items)
	}
	if p.Message != uiEN.RecoverFail {
		t.Fatalf("message = %q", p.Message)
	}
	if p.Detail != "Failed to load plugins\ndsh-wsl-workspace" {
		t.Fatalf("detail = %q", p.Detail)
	}
}

func TestParseRecoverNames(t *testing.T) {
	got := parseRecoverNames("dsh-wsl-workspace\n\ndshmarket")
	if len(got) != 2 || got[0] != "dsh-wsl-workspace" || got[1] != "dshmarket" {
		t.Fatalf("got %#v", got)
	}
	got = parseRecoverNames([]any{" dshmarket ", 1})
	if len(got) != 1 || got[0] != "dshmarket" {
		t.Fatalf("got %#v", got)
	}
}

func TestPluginMentioned(t *testing.T) {
	if !pluginMentioned("failed to apply loader entry (dsh-wsl-workspace)", "dsh-wsl-workspace") {
		t.Fatal("should match")
	}
	if pluginMentioned("Failed to load plugins", "dsh-wsl-workspace") {
		t.Fatal("should not invent a match")
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
