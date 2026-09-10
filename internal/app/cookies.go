package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func webviewUserDataPath() string {
	return filepath.Join(dshGoDir(), "webview")
}

func legacyWebviewUserDataPaths() []string {
	appdata := strings.TrimSpace(os.Getenv("APPDATA"))
	if appdata == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		p := filepath.Join(appdata, name)
		if seen[strings.ToLower(p)] {
			return
		}
		seen[strings.ToLower(p)] = true
		out = append(out, p)
	}
	add("dsh-go.exe")
	add("dsh-go-windows-amd64.exe")
	if exe, err := os.Executable(); err == nil {
		add(filepath.Base(exe))
	}
	return out
}

func purgeWebViewCookies(roots ...string) {
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			base := d.Name()
			if base == "Cookies" || strings.HasPrefix(base, "Cookies-") {
				_ = os.Remove(path)
			}
			return nil
		})
	}
}
