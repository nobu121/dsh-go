package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkipSelfInstallInDev(t *testing.T) {
	if selfInstallEnabled {
		t.Skip("production builds self-install")
	}
	if !skipSelfInstall() {
		t.Fatal("dev/test builds must not self-install")
	}
}

func TestWantsUninstall(t *testing.T) {
	if wantsUninstall([]string{"dsh-go.exe"}) {
		t.Fatal("plain launch")
	}
	if !wantsUninstall([]string{"dsh-go.exe", uninstallArg}) {
		t.Fatal("uninstall flag")
	}
}

func TestQuotedUninstallCmd(t *testing.T) {
	got := quotedUninstallCmd(`C:\Users\a\AppData\Roaming\dsh-go\dsh-go.exe`)
	if !strings.HasPrefix(got, `"`) || !strings.Contains(got, uninstallArg) {
		t.Fatalf("got %q", got)
	}
}

func TestInstallExePath(t *testing.T) {
	got := installExePath()
	if filepath.Base(got) != installExeName {
		t.Fatalf("got %q", got)
	}
	if filepath.Dir(got) != dshGoDir() {
		t.Fatalf("dir %q", got)
	}
}

func TestIsInstalledExe(t *testing.T) {
	if !isInstalledExe(installExePath()) {
		t.Fatal("install path")
	}
	other := filepath.Join(t.TempDir(), installExeName)
	if isInstalledExe(other) {
		t.Fatal("other location")
	}
}

func TestPathUnderDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dsh-go")
	if !pathUnderDir(filepath.Join(root, "dsh-go.exe"), root) {
		t.Fatal("child")
	}
	if pathUnderDir(filepath.Join(t.TempDir(), "other.exe"), root) {
		t.Fatal("other tree")
	}
}

func TestUninstallDirsIncludeRuntime(t *testing.T) {
	dirs := uninstallDirs()
	if len(dirs) == 0 {
		t.Fatal("empty")
	}
	foundRuntime, foundInstall := false, false
	runtimeDir := filepath.Clean(runtimeCacheDir())
	installDir := filepath.Clean(dshGoDir())
	for _, dir := range dirs {
		if strings.EqualFold(dir, runtimeDir) {
			foundRuntime = true
		}
		if strings.EqualFold(dir, installDir) {
			foundInstall = true
		}
	}
	if !foundRuntime {
		t.Fatalf("missing runtime %q in %v", runtimeDir, dirs)
	}
	if !foundInstall {
		t.Fatalf("missing install %q in %v", installDir, dirs)
	}
}

func TestDelayedRemoveScriptRetriesRuntime(t *testing.T) {
	script := delayedRemoveScript([]string{`C:\AppData\dsh-go\dsh-runtime`, `C:\AppData\dsh-go`})
	if !strings.Contains(script, `rd /s /q "C:\AppData\dsh-go\dsh-runtime"`) {
		t.Fatalf("runtime: %s", script)
	}
	if !strings.Contains(script, `rd /s /q "C:\AppData\dsh-go"`) {
		t.Fatalf("install: %s", script)
	}
	if !strings.Contains(script, "for /L %i") {
		t.Fatalf("retry: %s", script)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.exe")
	dst := filepath.Join(dir, "sub", "dst.exe")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Fatalf("got %q", got)
	}
	if err := copyFile(dst, dst); err != nil {
		t.Fatal(err)
	}
}
