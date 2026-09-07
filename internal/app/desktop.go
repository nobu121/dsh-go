package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func desktopAssetName() string {
	osName, arch := runtimeOSArch()
	switch osName {
	case "darwin":
		return "dsh-go-darwin-" + arch + ".dmg"
	case "windows":
		return "dsh-go-windows-" + arch + ".zip"
	default:
		return "dsh-go-" + osName + "-" + arch
	}
}

func desktopReleaseMirrors(version string) []string {
	tag := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	seen := map[string]bool{}
	var out []string
	add := func(u string) {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(cnbReleaseBase(tag))
	add(githubReleaseBase(tag))
	return out
}

func latestReleaseVersion(ctx context.Context) string {
	repo := updateRepoOrDefault()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return ""
	}
	applyDownloadUA(req)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := runtimeHTTPClient.Do(req)
	if err != nil {
		log.Printf("latest release: %v", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("latest release: %s", resp.Status)
		return ""
	}
	var payload struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&payload); err != nil {
		return ""
	}
	if payload.Prerelease {
		return ""
	}
	return parseDSHVersion(payload.TagName)
}

func downloadDesktopUpdate(ctx context.Context, version string, onPrep PrepReporter) (string, error) {
	target := strings.TrimPrefix(strings.TrimSpace(version), "v")
	path, err := downloadDesktopVersion(ctx, target, onPrep)
	if err == nil {
		return path, nil
	}
	if latest := latestReleaseVersion(ctx); latest != "" && latest != target {
		log.Printf("desktop %s unavailable (%v); trying latest %s", target, err, latest)
		return downloadDesktopVersion(ctx, latest, onPrep)
	}
	return "", err
}

func downloadDesktopVersion(ctx context.Context, version string, onPrep PrepReporter) (string, error) {
	asset := desktopAssetName()
	bases := orderBySpeed(ctx, desktopReleaseMirrors(version), "SHA256SUMS")
	if len(bases) == 0 {
		return "", fmt.Errorf("no download mirrors")
	}
	dest := filepath.Join(os.TempDir(), asset)
	var last error
	for i, base := range bases {
		reportPrep(onPrep, PrepProgress{
			Stage:   "update",
			Message: currentUI().Downloading,
		})
		if err := downloadReleaseAsset(ctx, base, asset, dest, onPrep); err != nil {
			last = err
			if i+1 < len(bases) {
				log.Printf("desktop download from %s failed: %v; trying fallback", mirrorName(base), err)
			}
			continue
		}
		return dest, nil
	}
	return "", last
}

func downloadReleaseAsset(ctx context.Context, base, asset, dest string, onPrep PrepReporter) error {
	onPrep = throttlePrep(onPrep, 150*time.Millisecond)
	body, total, err := openRemote(ctx, strings.TrimRight(base, "/")+"/"+asset)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	defer body.Close()
	if total < 0 {
		total = 0
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), asset+".*.part")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
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
	if _, err := io.Copy(tmp, pr); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	if want, ok, err := fetchChecksum(ctx, strings.TrimRight(base, "/")+"/SHA256SUMS", asset); err != nil {
		return err
	} else if ok && !strings.EqualFold(want, sum) {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}

	_ = os.Remove(dest)
	if err := os.Rename(tmpName, dest); err != nil {
		return err
	}
	return nil
}

// desktopUpdateScript replaces the running Windows exe after it exits, then
// relaunches it. (goto) 2>nul lets the .cmd delete itself without leaving a
// visible "找不到批处理文件" console.
func desktopUpdateScript(current, src string) string {
	return fmt.Sprintf(""+
		"@echo off\r\n"+
		"set \"CUR=%s\"\r\n"+
		"set \"NEW=%s\"\r\n"+
		"for /l %%%%i in (1,1,30) do (\r\n"+
		"  move /y \"%%CUR%%\" \"%%CUR%%.old\" >nul 2>nul && goto replaced\r\n"+
		"  ping -n 2 127.0.0.1 >nul\r\n"+
		")\r\n"+
		"exit /b 1\r\n"+
		":replaced\r\n"+
		"move /y \"%%NEW%%\" \"%%CUR%%\" >nul\r\n"+
		"start \"\" \"%%CUR%%\"\r\n"+
		"(goto) 2>nul & del \"%%~f0\"\r\n",
		current, src)
}

func unpackDesktopExe(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var chosen *zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if !strings.EqualFold(filepath.Ext(base), ".exe") {
			continue
		}
		if strings.EqualFold(base, "dsh-go.exe") {
			chosen = f
			break
		}
		if chosen == nil {
			chosen = f
		}
	}
	if chosen == nil {
		return "", fmt.Errorf("zip has no exe: %s", zipPath)
	}

	tmp, err := os.CreateTemp("", "dsh-go-update-*.exe")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := extractZipFile(chosen, tmpName); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	return tmpName, nil
}
