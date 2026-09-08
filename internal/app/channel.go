package app

// The runtime and client channels decouple dsh runtime releases and client
// releases from client rebuilds. Each lives at a fixed, never version-scoped
// location holding a small JSON manifest (plus, for the runtime, the
// per-platform zips), so a new version ships without rebuilding the shell.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// runtimeChannelTag is a GitHub release tag that is deliberately not a
	// version. It is published as a prerelease so it can never be mistaken
	// for a client release by GitHub's /releases/latest or the CNB "latest"
	// download.
	runtimeChannelTag  = "runtime-latest"
	runtimeChannelFile = "runtime.json"
	runtimeChannelTTL  = 6 * time.Hour

	// clientChannelTag is the fixed release tag that advertises the current
	// client version through client.json, written on every client release and
	// mirrored to both GitHub and CNB. It is also published as a prerelease
	// so it never shadows a real client release. The shell races both mirrors
	// (see latestClientVersion) instead of querying the GitHub API, which
	// keeps launches fast where GitHub is slow or unreachable.
	clientChannelTag  = "client-latest"
	clientChannelFile = "client.json"
	clientChannelTTL  = 6 * time.Hour
)

type channelManifest struct {
	Version string `json:"version"`
}

var (
	channelMu  sync.Mutex
	channelVer string
	channelAt  time.Time
)

// targetDSHVersion reports the dsh version the runtime channel advertises,
// memoised for runtimeChannelTTL. It falls back to the last known answer and
// then to the vendored pin, so an unreachable channel never blocks a launch —
// but it does hit the network, so callers must not sit on the startup path.
func targetDSHVersion(ctx context.Context) string {
	if v := parseDSHVersion(os.Getenv("DSH_RUNTIME_VERSION")); v != "" {
		return v
	}

	channelMu.Lock()
	if channelVer != "" && time.Since(channelAt) < runtimeChannelTTL {
		v := channelVer
		channelMu.Unlock()
		return v
	}
	channelMu.Unlock()

	v, err := fetchChannelVersion(ctx)

	channelMu.Lock()
	defer channelMu.Unlock()
	if err != nil {
		log.Printf("runtime channel: %v", err)
		if channelVer != "" {
			return channelVer
		}
		return bundledDSHVersion()
	}
	channelVer, channelAt = v, time.Now()
	return v
}

func fetchChannelVersion(ctx context.Context) (string, error) {
	bases := orderBySpeed(ctx, runtimeBaseURLs(), runtimeChannelFile)
	if len(bases) == 0 {
		return "", errors.New("no runtime channel configured")
	}
	var last error
	for _, base := range bases {
		v, err := fetchChannelVersionFrom(ctx, base+"/"+runtimeChannelFile)
		if err != nil {
			last = err
			continue
		}
		return v, nil
	}
	return "", last
}

func fetchChannelVersionFrom(ctx context.Context, url string) (string, error) {
	body, _, err := openRemote(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer body.Close()
	b, err := io.ReadAll(io.LimitReader(body, 64<<10))
	if err != nil {
		return "", err
	}
	var m channelManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return "", fmt.Errorf("parse %s: %w", runtimeChannelFile, err)
	}
	v := parseDSHVersion(m.Version)
	if v == "" {
		return "", fmt.Errorf("%s has no usable version", runtimeChannelFile)
	}
	return v, nil
}

// --- client channel ---------------------------------------------------------

var (
	clientMu  sync.Mutex
	clientVer string
	clientAt  time.Time
)

// clientReleaseBases returns the directories that host client.json. CNB comes
// first (it is the mirror reachable without a proxy in mainland China); the
// GitHub release is the fallback, so the check works for users outside CNB
// too. DSH_CLIENT_BASE_URL overrides both, mirroring DSH_RUNTIME_BASE_URL.
func clientReleaseBases() []string {
	if u := strings.TrimSpace(os.Getenv("DSH_CLIENT_BASE_URL")); u != "" {
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
	add(cnbReleaseBase(clientChannelTag))
	add(githubReleaseBase(clientChannelTag))
	return out
}

// latestClientVersion reports the client version the channel advertises,
// memoised for clientChannelTTL exactly like targetDSHVersion. The mirrors are
// probed concurrently and fetched in order of responsiveness, so the first
// healthy mirror answers: an unreachable GitHub never stalls a launch when CNB
// responds, and vice versa. On total failure it serves the last known answer,
// then "" (callers treat "" as "no channel information").
func latestClientVersion(ctx context.Context) string {
	clientMu.Lock()
	if clientVer != "" && time.Since(clientAt) < clientChannelTTL {
		v := clientVer
		clientMu.Unlock()
		return v
	}
	clientMu.Unlock()

	v, err := fetchClientChannelVersion(ctx)

	clientMu.Lock()
	defer clientMu.Unlock()
	if err != nil {
		log.Printf("client channel: %v", err)
		return clientVer
	}
	clientVer, clientAt = v, time.Now()
	return v
}

func fetchClientChannelVersion(ctx context.Context) (string, error) {
	bases := orderBySpeed(ctx, clientReleaseBases(), clientChannelFile)
	if len(bases) == 0 {
		return "", errors.New("no client channel configured")
	}
	var last error
	for _, base := range bases {
		v, err := fetchClientChannelVersionFrom(ctx, base+"/"+clientChannelFile)
		if err != nil {
			last = err
			continue
		}
		return v, nil
	}
	return "", last
}

func fetchClientChannelVersionFrom(ctx context.Context, url string) (string, error) {
	body, _, err := openRemote(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer body.Close()
	b, err := io.ReadAll(io.LimitReader(body, 64<<10))
	if err != nil {
		return "", err
	}
	var m channelManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return "", fmt.Errorf("parse %s: %w", clientChannelFile, err)
	}
	v := parseDSHVersion(m.Version)
	if v == "" {
		return "", fmt.Errorf("%s has no usable version", clientChannelFile)
	}
	return v, nil
}
