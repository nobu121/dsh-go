package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	defaultGitHubRepo  = "nobu121/dsh-go"
	defaultCNBRepo     = "nobu121/dsh-go"
	downloadUserAgent  = "dsh-go"
	mirrorProbeTimeout = 2500 * time.Millisecond
)

func updateRepoOrDefault() string {
	if r := strings.TrimSpace(UpdateRepo); r != "" {
		return r
	}
	return defaultGitHubRepo
}

func cnbReleaseBase(tag string) string {
	return "https://cnb.cool/" + defaultCNBRepo + "/-/releases/download/" + strings.TrimPrefix(tag, "/")
}

func githubReleaseBase(tag string) string {
	return "https://github.com/" + updateRepoOrDefault() + "/releases/download/" + strings.TrimPrefix(tag, "/")
}

func mirrorName(base string) string {
	switch {
	case strings.Contains(base, "cnb.cool"):
		return "CNB"
	case strings.Contains(base, "github.com"):
		return "GitHub"
	default:
		return base
	}
}

func localMirror(base string) bool {
	return strings.HasPrefix(base, "file:") || strings.HasPrefix(base, "file://")
}

func applyDownloadUA(req *http.Request) {
	req.Header.Set("User-Agent", downloadUserAgent)
}

func probeMirror(ctx context.Context, base, probe string) error {
	raw := strings.TrimRight(base, "/") + "/" + strings.TrimPrefix(probe, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	applyDownloadUA(req)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := runtimeHTTPClient.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64))
	resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}

func orderBySpeed(ctx context.Context, bases []string, probe string) []string {
	if len(bases) < 2 {
		return bases
	}
	for _, base := range bases {
		if localMirror(base) {
			return bases
		}
	}

	type scored struct {
		base string
		d    time.Duration
		err  error
	}
	ctx, cancel := context.WithTimeout(ctx, mirrorProbeTimeout)
	defer cancel()
	ch := make(chan scored, len(bases))
	for _, base := range bases {
		go func(base string) {
			start := time.Now()
			err := probeMirror(ctx, base, probe)
			ch <- scored{base, time.Since(start), err}
		}(base)
	}
	scores := make([]scored, 0, len(bases))
	for range bases {
		scores = append(scores, <-ch)
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if (scores[i].err == nil) != (scores[j].err == nil) {
			return scores[i].err == nil
		}
		return scores[i].d < scores[j].d
	})
	out := make([]string, 0, len(scores))
	for _, s := range scores {
		if s.err != nil {
			log.Printf("mirror %s: %v", mirrorName(s.base), s.err)
		} else {
			log.Printf("mirror %s: %s", mirrorName(s.base), s.d.Round(time.Millisecond))
		}
		out = append(out, s.base)
	}
	if scores[0].err == nil {
		log.Printf("using %s", mirrorName(scores[0].base))
	}
	return out
}
