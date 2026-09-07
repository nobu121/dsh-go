package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var runtimeHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

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
	add(cnbReleaseBase(runtimeChannelTag))
	add(githubReleaseBase(runtimeChannelTag))
	return out
}

func runtimeCacheDir() string {
	if d := strings.TrimSpace(os.Getenv("DSH_RUNTIME_DIR")); d != "" {
		return d
	}
	return filepath.Join(dshGoDir(), runtimeDirName)
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

// readRuntimeVersion prefers the VERSION stamp written at publish time and
// falls back to the installed package, so a runtime is always self-describing
// and never has to match a version baked into the shell.
func readRuntimeVersion(root string) string {
	if b, err := os.ReadFile(filepath.Join(root, "VERSION")); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	return readPackageVersion(filepath.Join(root, "node_modules", "@deepseek-ai", "dsh", "package.json"))
}

func readPackageVersion(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Version)
}

// cacheRuntimeCommand accepts any structurally valid cache regardless of
// version. Upgrades are offered separately (see offerDSHUpdate), so a shell
// update never invalidates a working runtime and an offline launch never
// stalls on a version mismatch.
func cacheRuntimeCommand() ([]string, bool) {
	root := runtimeCacheDir()
	if !runtimeLooksValid(root) {
		return nil, false
	}
	return runtimeNodeCommand(root), true
}

func runtimeNodeCommand(root string) []string {
	return append([]string{filepath.Join(root, bundledNodeName()), dshBinJS(root)}, webFlags()...)
}

func fetchCachedRuntime(ctx context.Context, onPrep PrepReporter) error {
	bases := orderBySpeed(ctx, runtimeBaseURLs(), runtimeChannelFile)
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

func throttlePrep(fn PrepReporter, minInterval time.Duration) PrepReporter {
	if fn == nil {
		return nil
	}
	var mu sync.Mutex
	var last time.Time
	return func(p PrepProgress) {
		if p.Total <= 0 {
			fn(p)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		done := p.Total > 0 && p.Bytes >= p.Total
		if !done && !last.IsZero() && now.Sub(last) < minInterval {
			return
		}
		last = now
		fn(p)
	}
}

func fetchCachedRuntimeFrom(ctx context.Context, onPrep PrepReporter, base string) error {
	asset := runtimeAssetName()
	zipURL := base + "/" + asset
	dest := runtimeCacheDir()
	onPrep = throttlePrep(onPrep, 150*time.Millisecond)

	reportPrep(onPrep, PrepProgress{Stage: "download", Message: currentUI().Downloading})

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
		Message: currentUI().Downloading,
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
			Message: currentUI().Downloading,
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

	reportPrep(onPrep, PrepProgress{Stage: "unpack", Message: currentUI().Installing})
	tmpDir := dest + ".tmp"
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := unzipRuntime(tmpName, tmpDir, func(n, tot int64) {
		reportPrep(onPrep, PrepProgress{
			Stage:   "unpack",
			Message: currentUI().Installing,
			Bytes:   n,
			Total:   tot,
		})
	}); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if !runtimeLooksValid(tmpDir) {
		_ = os.RemoveAll(tmpDir)
		return errors.New("downloaded runtime is missing node or @deepseek-ai/dsh")
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
	applyDownloadUA(req)
	resp, err := runtimeHTTPClient.Do(req)
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

func unzipRuntime(zipPath, dest string, onProg func(n, total int64)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	rootPrefix := zipRootPrefix(r.File)
	dest = filepath.Clean(dest)
	var total int64
	for _, f := range r.File {
		if name, ok := unzipEntryName(f.Name, rootPrefix); ok && name != "" && !f.FileInfo().IsDir() {
			total += int64(f.UncompressedSize64)
		}
	}
	var done int64
	emit := func(n int64) {
		if onProg == nil || total <= 0 {
			return
		}
		if n > total {
			n = total
		}
		onProg(n, total)
	}
	emit(0)
	for _, f := range r.File {
		name, ok := unzipEntryName(f.Name, rootPrefix)
		if !ok || name == "" {
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
		base := done
		if err := copyZipFile(f, target, func(n int64) { emit(base + n) }); err != nil {
			return err
		}
		done += int64(f.UncompressedSize64)
		emit(done)
	}
	return nil
}

func unzipEntryName(name, rootPrefix string) (string, bool) {
	if rootPrefix != "" {
		if name == rootPrefix || name == strings.TrimSuffix(rootPrefix, "/") {
			return "", false
		}
		if !strings.HasPrefix(name, rootPrefix) {
			return "", false
		}
		name = strings.TrimPrefix(name, rootPrefix)
	}
	if name == "" {
		return "", false
	}
	return name, true
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
	return copyZipFile(f, target, nil)
}

func copyZipFile(f *zip.File, target string, onBytes func(n int64)) error {
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
	var src io.Reader = rc
	if onBytes != nil {
		src = &progressReader{r: rc, fn: func(n, _ int64) { onBytes(n) }}
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
