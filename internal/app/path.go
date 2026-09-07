package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var extraBinDirsFn = defaultExtraBinDirs

func initUserPATH() {
	parts := []string{readLoginPATH(2 * time.Second), os.Getenv("PATH")}
	parts = append(parts, extraBinDirsFn()...)
	os.Setenv("PATH", mergePATH(parts...))
}

func lookNamed(name string) (string, error) {
	if p, err := lookPath(name); err == nil && p != "" {
		return p, nil
	}
	for _, dir := range extraBinDirsFn() {
		if p := lookInDir(dir, name); p != "" {
			return p, nil
		}
	}
	return "", exec.ErrNotFound
}

func lookInDir(dir, name string) string {
	if dir == "" || name == "" {
		return ""
	}
	for _, cand := range pathCandidates(name) {
		p := filepath.Join(dir, cand)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

func pathCandidates(name string) []string {
	if runtime.GOOS != "windows" {
		return []string{name}
	}
	lower := strings.ToLower(name)
	for _, ext := range pathExts() {
		if strings.HasSuffix(lower, ext) {
			return []string{name}
		}
	}
	out := []string{name}
	for _, ext := range pathExts() {
		out = append(out, name+ext)
	}
	return out
}

func pathExts() []string {
	raw := os.Getenv("PATHEXT")
	if raw == "" {
		raw = ".COM;.EXE;.BAT;.CMD"
	}
	var exts []string
	for _, ext := range strings.Split(raw, ";") {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		exts = append(exts, ext)
	}
	return exts
}

func defaultExtraBinDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	var dirs []string
	add := func(p string) {
		if p == "" {
			return
		}
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			dirs = append(dirs, p)
		}
	}
	if home != "" {
		add(filepath.Join(home, ".local", "bin"))
		add(filepath.Join(home, "go", "bin"))
		add(filepath.Join(home, ".bun", "bin"))
		add(filepath.Join(home, ".cargo", "bin"))
		add(filepath.Join(home, "Library", "pnpm"))
		if nvm := os.Getenv("NVM_BIN"); nvm != "" {
			add(nvm)
		}
		nvmRoot := filepath.Join(home, ".nvm", "versions", "node")
		if ents, err := os.ReadDir(nvmRoot); err == nil {
			for i := len(ents) - 1; i >= 0; i-- {
				if ents[i].IsDir() {
					add(filepath.Join(nvmRoot, ents[i].Name(), "bin"))
					break
				}
			}
		}
	}
	if runtime.GOOS == "darwin" {
		add("/opt/homebrew/bin")
		add("/usr/local/bin")
	}
	if runtime.GOOS == "windows" {
		if app := os.Getenv("APPDATA"); app != "" {
			add(filepath.Join(app, "npm"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			add(filepath.Join(local, "fnm"))
		}
	}
	add(dshGoBinDir())
	return dirs
}

func readLoginPATH(timeout time.Duration) string {
	if os.Getenv("DSH_SKIP_LOGIN_PATH") != "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			return ""
		}
		shell = "/bin/zsh"
	}
	cmd := exec.CommandContext(ctx, shell, "-l", "-c", "printenv PATH || echo $PATH")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func mergePATH(parts ...string) string {
	seen := map[string]bool{}
	var dirs []string
	for _, part := range parts {
		for _, dir := range splitPATH(part) {
			if dir == "" || seen[dir] {
				continue
			}
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return strings.Join(dirs, string(os.PathListSeparator))
}

func splitPATH(p string) []string {
	if p == "" {
		return nil
	}
	return strings.Split(p, string(os.PathListSeparator))
}
