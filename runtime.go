package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	runtimeDirName = "dsh-runtime"
)

func runtimeBaseURL() string {
	urls := runtimeBaseURLs()
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func runtimeBaseURLs() []string {
	if u := strings.TrimSpace(os.Getenv("DSH_RUNTIME_BASE_URL")); u != "" {
		return []string{strings.TrimRight(u, "/")}
	}
	seen := make(map[string]bool)
	var out []string
	add := func(u string) {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(RuntimeBaseURL)
	if repo := strings.TrimSpace(UpdateRepo); repo != "" {
		add("https://github.com/" + repo + "/releases/download/v" + currentVersion())
	}
	return out
}

func runtimeCacheDir() string {
	if d := strings.TrimSpace(os.Getenv("DSH_RUNTIME_DIR")); d != "" {
		return d
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "dsh-go", runtimeDirName)
	}
	return filepath.Join(cfg, "dsh-go", runtimeDirName)
}

func runtimeOSArch() (string, string) {
	osName := runtime.GOOS
	switch osName {
	case "darwin", "windows", "linux":
	default:
		osName = runtime.GOOS
	}
	archName := runtime.GOARCH
	switch archName {
	case "amd64", "arm64":
	default:
		archName = runtime.GOARCH
	}
	return osName, archName
}

func runtimeAssetName() string {
	osName, archName := runtimeOSArch()
	return "dsh-runtime-" + osName + "-" + archName + ".zip"
}

func runtimeLooksValid(root string) bool {
	node := filepath.Join(root, bundledNodeName())
	if fi, err := os.Stat(node); err != nil || fi.IsDir() {
		return false
	}
	if _, err := os.Stat(dshBinJS(root)); err != nil {
		return false
	}
	return true
}

func readRuntimeVersion(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func cacheRuntimeCommand() ([]string, bool) {
	root := runtimeCacheDir()
	if !runtimeLooksValid(root) {
		return nil, false
	}
	if readRuntimeVersion(root) != currentVersion() {
		return nil, false
	}
	return runtimeNodeCommand(root), true
}

func runtimeNodeCommand(root string) []string {
	return append([]string{filepath.Join(root, bundledNodeName()), dshBinJS(root)}, webFlags()...)
}

func fetchCachedRuntime(ctx context.Context, onPrep PrepReporter) error {
	bases := runtimeBaseURLs()
	if len(bases) == 0 {
		return errors.New("DSH_RUNTIME_BASE_URL is empty")
	}
	var last error
	for i, base := range bases {
		if err := fetchCachedRuntimeFrom(ctx, onPrep, base); err != nil {
			last = err
			if i+1 < len(bases) {
				log.Printf("runtime download from %s failed: %v; trying fallback", base, err)
			}
			continue
		}
		return nil
	}
	return last
}

func fetchCachedRuntimeFrom(ctx context.Context, onPrep PrepReporter, base string) error {
	pin := currentVersion()
	asset := runtimeAssetName()
	zipURL := base + "/" + asset
	dest := runtimeCacheDir()

	reportPrep(onPrep, PrepProgress{Stage: "download", Message: "正在下载 DeepSeek Harness…"})

	body, total, err := openRemote(ctx, zipURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	defer body.Close()
	if total < 0 {
		total = 0
	}
	reportPrep(onPrep, PrepProgress{
		Stage:   "download",
		Message: "正在下载 DeepSeek Harness…",
		Total:   total,
	})

	tmpZip, err := os.CreateTemp("", "dsh-runtime-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmpZip.Name()
	defer os.Remove(tmpName)

	hash := sha256.New()
	pr := &progressReader{r: io.TeeReader(body, hash), total: total, fn: func(n, tot int64) {
		reportPrep(onPrep, PrepProgress{
			Stage:   "download",
			Message: "正在下载 DeepSeek Harness…",
			Bytes:   n,
			Total:   tot,
		})
	}}
	if _, err := io.Copy(tmpZip, pr); err != nil {
		tmpZip.Close()
		return err
	}
	if err := tmpZip.Close(); err != nil {
		return err
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	if want, ok, err := fetchChecksum(ctx, base+"/SHA256SUMS", asset); err != nil {
		return err
	} else if ok && !strings.EqualFold(want, sum) {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}

	reportPrep(onPrep, PrepProgress{Stage: "unpack", Message: "正在解压运行时…"})
	tmpDir := dest + ".tmp"
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := unzipRuntime(tmpName, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if !runtimeLooksValid(tmpDir) {
		_ = os.RemoveAll(tmpDir)
		return errors.New("downloaded runtime is missing node or @deepseek-ai/dsh")
	}
	if ver := readRuntimeVersion(tmpDir); ver != "" && ver != pin {
		_ = os.RemoveAll(tmpDir)
		return fmt.Errorf("downloaded runtime VERSION %s != pin %s", ver, pin)
	}
	if ver := readRuntimeVersion(tmpDir); ver == "" {
		if err := os.WriteFile(filepath.Join(tmpDir, "VERSION"), []byte(pin+"\n"), 0o644); err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(tmpDir, dest); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	fn    func(n, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.fn != nil {
		p.fn(p.n, p.total)
	}
	return n, err
}

func openRemote(ctx context.Context, raw string) (io.ReadCloser, int64, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, 0, err
	}
	if u.Scheme == "file" || u.Scheme == "" {
		path := fileURLPath(u)
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, err
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, 0, err
		}
		return f, fi.Size(), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("%s", resp.Status)
	}
	return resp.Body, resp.ContentLength, nil
}

func fileURLPath(u *url.URL) string {
	if u.Host != "" && runtime.GOOS == "windows" {
		return filepath.FromSlash(u.Host + "/" + strings.TrimPrefix(u.Path, "/"))
	}
	return filepath.FromSlash(u.Path)
}

func fetchChecksum(ctx context.Context, sumsURL, asset string) (string, bool, error) {
	body, _, err := openRemote(ctx, sumsURL)
	if err != nil {
		return "", false, nil
	}
	defer body.Close()
	b, err := io.ReadAll(body)
	if err != nil {
		return "", false, err
	}
	want, ok := checksumForAsset(string(b), asset)
	if !ok {
		return "", false, nil
	}
	return want, true, nil
}

func checksumForAsset(sums, asset string) (string, bool) {
	for _, line := range strings.Split(sums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) == asset {
			return fields[0], true
		}
	}
	return "", false
}

func unzipRuntime(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	rootPrefix := zipRootPrefix(r.File)
	dest = filepath.Clean(dest)
	for _, f := range r.File {
		name := f.Name
		if rootPrefix != "" {
			if name == rootPrefix || name == strings.TrimSuffix(rootPrefix, "/") {
				continue
			}
			if !strings.HasPrefix(name, rootPrefix) {
				continue
			}
			name = strings.TrimPrefix(name, rootPrefix)
		}
		if name == "" {
			continue
		}
		target, err := safeJoin(dest, name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

func zipRootPrefix(files []*zip.File) string {
	var top string
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		rel := strings.SplitN(name, "/", 2)
		if rel[0] == "" {
			continue
		}
		if top == "" {
			top = rel[0]
			continue
		}
		if rel[0] != top {
			return ""
		}
	}
	if top == "" {
		return ""
	}
	// Only strip a single wrapping directory if every entry lives under it
	// and that directory is not the runtime itself (node / VERSION at root).
	for _, keep := range []string{bundledNodeName(), "VERSION", "node_modules"} {
		for _, f := range files {
			base := strings.TrimPrefix(f.Name, "./")
			if base == keep || strings.HasPrefix(base, keep+"/") {
				return ""
			}
		}
	}
	return top + "/"
}

func safeJoin(dir, name string) (string, error) {
	cleaned := filepath.Clean(filepath.Join(dir, filepath.FromSlash(name)))
	prefix := dir + string(os.PathSeparator)
	if cleaned != dir && !strings.HasPrefix(cleaned, prefix) {
		return "", fmt.Errorf("illegal path in zip: %s", name)
	}
	return cleaned, nil
}

func extractZipFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	mode := f.Mode()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
