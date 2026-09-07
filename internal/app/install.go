package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	uninstallArg       = "--uninstall"
	skipSelfInstallEnv = "DSH_SKIP_SELF_INSTALL"
	installExeName     = "dsh-go.exe"
	productDisplayName = "Deepseek Harness GO"
	desktopShortcut    = productDisplayName + ".lnk"
	desktopShortcutOld = "dsh-go.lnk"
)

func wantsUninstall(args []string) bool {
	for _, a := range args {
		if a == uninstallArg {
			return true
		}
	}
	return false
}

func skipSelfInstall() bool {
	return !selfInstallEnabled || os.Getenv(skipSelfInstallEnv) != ""
}

func installExePath() string {
	return filepath.Join(dshGoDir(), installExeName)
}

func currentExe() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return filepath.Abs(p)
}

func copyFile(src, dst string) error {
	if sameFilePath(src, dst) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func quotedUninstallCmd(exe string) string {
	return `"` + exe + `" ` + uninstallArg
}

func isInstalledExe(exe string) bool {
	if sameFilePath(exe, installExePath()) {
		return true
	}
	return sameFilePath(filepath.Dir(exe), dshGoDir()) &&
		strings.EqualFold(filepath.Base(exe), installExeName)
}

func uninstallDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range []string{runtimeCacheDir(), dshGoDir()} {
		dir = filepath.Clean(dir)
		if dir == "" || dir == "." || seen[strings.ToLower(dir)] {
			continue
		}
		seen[strings.ToLower(dir)] = true
		out = append(out, dir)
	}
	return out
}

func delayedRemoveScript(dirs []string) string {
	var b strings.Builder
	b.WriteString("ping -n 4 127.0.0.1 >nul")
	for _, dir := range dirs {
		q := `"` + dir + `"`
		b.WriteString(" & for /L %i in (1,1,10) do @if exist ")
		b.WriteString(q)
		b.WriteString(" (rd /s /q ")
		b.WriteString(q)
		b.WriteString(" & ping -n 2 127.0.0.1 >nul)")
	}
	return b.String()
}

func pathUnderDir(path, dir string) bool {
	path = filepath.Clean(path)
	root := filepath.Clean(dir)
	if sameFilePath(path, root) {
		return true
	}
	prefix := root + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return true
	}
	return len(path) >= len(prefix) && strings.EqualFold(path[:len(prefix)], prefix)
}
