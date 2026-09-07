package app

// The runtime channel decouples dsh runtime releases from shell releases. It
// lives at a fixed, never version-scoped location holding runtime.json plus the
// per-platform runtime zips, so a new dsh ships without rebuilding the shell.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	// runtimeChannelTag is a GitHub release tag that is deliberately not a
	// version. It is published as a prerelease so the shell's own updater,
	// which reads /releases/latest, never sees it.
	runtimeChannelTag  = "runtime-latest"
	runtimeChannelFile = "runtime.json"
	runtimeChannelTTL  = 6 * time.Hour
)

type runtimeChannel struct {
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
	var m runtimeChannel
	if err := json.Unmarshal(b, &m); err != nil {
		return "", fmt.Errorf("parse %s: %w", runtimeChannelFile, err)
	}
	v := parseDSHVersion(m.Version)
	if v == "" {
		return "", fmt.Errorf("%s has no usable version", runtimeChannelFile)
	}
	return v, nil
}
