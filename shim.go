package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const shimMarker = "dsh-go-shim"

var persistShimPATH = persistShimPATHImpl

func dshGoDir() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "dsh-go")
	}
	return filepath.Join(cfg, "dsh-go")
}

func dshGoBinDir() string {
	if d := strings.TrimSpace(os.Getenv("DSH_BIN_DIR")); d != "" {
		return d
	}
	return filepath.Join(dshGoDir(), "bin")
}

func ensureDSHShim(root string) error {
	if !runtimeLooksValid(root) {
		return errors.New("runtime is missing node or @deepseek-ai/dsh")
	}
	dir := dshGoBinDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	node := filepath.Join(root, bundledNodeName())
	if err := writeDSHShim(dir, node, dshBinJS(root)); err != nil {
		return err
	}
	if os.Getenv("DSH_SKIP_LOGIN_PATH") == "" {
		if err := persistShimPATH(dir); err != nil {
			log.Printf("dsh shim PATH: %v", err)
		}
	}
	os.Setenv("PATH", mergePATH(os.Getenv("PATH"), dir))
	return nil
}

func writeDSHShim(dir, node, binjs string) error {
	if runtime.GOOS == "windows" {
		return writeFileAtomic(filepath.Join(dir, "dsh.cmd"), windowsShimBody(node, binjs), 0o644)
	}
	return writeFileAtomic(filepath.Join(dir, "dsh"), unixShimBody(node, binjs), 0o755)
}

func unixShimBody(node, binjs string) string {
	return "#!/bin/sh\n# " + shimMarker + "\nexec " + shQuote(node) + " " + shQuote(binjs) + " \"$@\"\n"
}

func windowsShimBody(node, binjs string) string {
	return "@echo off\r\nREM " + shimMarker + "\r\n" + cmdQuote(node) + " " + cmdQuote(binjs) + " %*\r\n"
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func cmdQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func writeFileAtomic(path, body string, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func isDSHGoShim(path string) bool {
	if path == "" {
		return false
	}
	if sameFilePath(filepath.Dir(path), dshGoBinDir()) {
		return true
	}
	return hasShimMarker(path)
}

func hasShimMarker(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte(shimMarker))
}

func sameFilePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

func pathHasDir(path, dir string) bool {
	want := filepath.Clean(dir)
	for _, p := range splitPATH(path) {
		if p == "" {
			continue
		}
		got := filepath.Clean(p)
		if got == want || (runtime.GOOS == "windows" && strings.EqualFold(got, want)) {
			return true
		}
	}
	return false
}

func appendUserPATH(path, dir string) (string, bool) {
	if pathHasDir(path, dir) {
		return path, false
	}
	if path == "" {
		return dir, true
	}
	return path + string(os.PathListSeparator) + dir, true
}

func launchEnv(home string, src resolvedDSH) []string {
	env := append(os.Environ(), "DSH_HOME="+home)
	if src.Kind != sourceCache && src.Kind != sourceBundled {
		return env
	}
	root := src.Path
	if root == "" || !runtimeLooksValid(root) {
		return env
	}
	if err := ensureDSHShim(root); err != nil {
		log.Printf("dsh shim: %v", err)
		return env
	}
	return prependPATH(env, dshGoBinDir(), root)
}

func prependPATH(env []string, dirs ...string) []string {
	path := ""
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if ok && strings.EqualFold(k, "PATH") {
			path = v
			continue
		}
		out = append(out, e)
	}
	parts := append(append([]string{}, dirs...), path)
	return append(out, "PATH="+mergePATH(parts...))
}
