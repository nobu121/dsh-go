//go:build windows

package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const uninstallRegKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\dsh-go`

func handleAppLifecycle() bool {
	if wantsUninstall(os.Args) {
		uninstallWindows()
		return true
	}
	if skipSelfInstall() {
		return false
	}
	return selfInstallWindows()
}

func selfInstallWindows() bool {
	src, err := currentExe()
	if err != nil {
		log.Printf("self-install: %v", err)
		return false
	}
	dest := installExePath()
	if isInstalledExe(src) {
		if err := ensureInstallMetadata(); err != nil {
			log.Printf("self-install: %v", err)
		}
		return false
	}
	if err := copyFile(src, dest); err != nil {
		log.Printf("self-install: %v", err)
		if _, statErr := os.Stat(dest); statErr != nil {
			return false
		}
	}
	if err := ensureInstallMetadata(); err != nil {
		log.Printf("self-install: %v", err)
	}
	if err := startDetached(dest); err != nil {
		log.Printf("self-install launch: %v", err)
		return false
	}
	return true
}

func ensureInstallMetadata() error {
	exe := installExePath()
	if err := writeUninstallRegistry(exe, dshGoDir()); err != nil {
		return err
	}
	desktop, err := userDesktopDir()
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(desktop, desktopShortcutOld))
	lnk := filepath.Join(desktop, desktopShortcut)
	if _, err := os.Stat(lnk); err == nil {
		return nil
	}
	return writeDesktopShortcut(exe)
}

func writeUninstallRegistry(exe, dir string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallRegKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	uninstall := quotedUninstallCmd(exe)
	vals := map[string]string{
		"DisplayName":          productDisplayName,
		"DisplayVersion":       shellVersion(),
		"Publisher":            "nobu121",
		"InstallLocation":      dir,
		"DisplayIcon":          exe,
		"UninstallString":      uninstall,
		"QuietUninstallString": uninstall,
	}
	for name, val := range vals {
		if err := k.SetStringValue(name, val); err != nil {
			return err
		}
	}
	if err := k.SetDWordValue("NoModify", 1); err != nil {
		return err
	}
	if err := k.SetDWordValue("NoRepair", 1); err != nil {
		return err
	}
	if fi, err := os.Stat(exe); err == nil {
		_ = k.SetDWordValue("EstimatedSize", uint32(fi.Size()/1024))
	}
	return nil
}

func writeDesktopShortcut(target string) error {
	desktop, err := userDesktopDir()
	if err != nil {
		return err
	}
	lnk := filepath.Join(desktop, desktopShortcut)
	ps := fmt.Sprintf(
		"$s = (New-Object -ComObject WScript.Shell).CreateShortcut(%s); $s.TargetPath = %s; $s.WorkingDirectory = %s; $s.IconLocation = %s; $s.Save()",
		psSingle(lnk),
		psSingle(target),
		psSingle(filepath.Dir(target)),
		psSingle(target+",0"),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = hiddenProcAttr(0)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shortcut: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func userDesktopDir() (string, error) {
	if p, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, 0); err == nil && p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Desktop"), nil
}

func psSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func startDetached(path string) error {
	cmd := exec.Command("cmd", "/C", "start", "", path)
	cmd.SysProcAttr = hiddenProcAttr(0)
	return cmd.Start()
}

func uninstallWindows() {
	killOtherDSHGo()
	time.Sleep(300 * time.Millisecond)
	if err := registry.DeleteKey(registry.CURRENT_USER, uninstallRegKey); err != nil {
		log.Printf("uninstall registry: %v", err)
	}
	if desktop, err := userDesktopDir(); err == nil {
		_ = os.Remove(filepath.Join(desktop, desktopShortcut))
		_ = os.Remove(filepath.Join(desktop, desktopShortcutOld))
	}
	unregisterDSHIfOurs()
	removeInstallFiles()
}

func removeInstallFiles() {
	killProcessesUnder(dshGoDir())
	time.Sleep(300 * time.Millisecond)

	dirs := uninstallDirs()
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("uninstall %s: %v", dir, err)
		}
	}

	exe, err := currentExe()
	if err == nil && pathUnderDir(exe, dshGoDir()) {
		cmd := exec.Command("cmd", "/C", delayedRemoveScript(dirs))
		cmd.SysProcAttr = hiddenProcAttr(0)
		if err := cmd.Start(); err != nil {
			log.Printf("uninstall files: %v", err)
		}
		return
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("uninstall %s: %v", dir, err)
		}
	}
}

func killOtherDSHGo() {
	runHidden("taskkill", "/F", "/IM", installExeName, "/FI", "PID ne "+strconv.Itoa(os.Getpid()))
}

func killProcessesUnder(dir string) {
	dir = filepath.Clean(dir)
	if dir == "" {
		return
	}
	ps := fmt.Sprintf(
		`$root = %s; $me = %d; Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -ne $me -and $_.ExecutablePath -and $_.ExecutablePath.StartsWith($root, [StringComparison]::OrdinalIgnoreCase) } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`,
		psSingle(dir),
		os.Getpid(),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = hiddenProcAttr(0)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("uninstall processes: %v: %s", err, strings.TrimSpace(string(out)))
	}
}

func unregisterDSHIfOurs() {
	dir := dshGoBinDir()
	for _, name := range []string{"dsh.cmd", "dsh"} {
		p := filepath.Join(dir, name)
		if isDSHGoShim(p) {
			_ = os.Remove(p)
		}
	}
	if err := removeShimPATHImpl(dir); err != nil {
		log.Printf("uninstall dsh PATH: %v", err)
	}
}
