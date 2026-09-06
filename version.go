package main

import (
	_ "embed"
	"strings"
)

//go:embed dsh.version
var pinnedVersion string

// Version is the running product version (same string as the bundled dsh pin).
// Release builds override it with -X main.Version=.
var Version string

// UpdateRepo is the GitHub "owner/repo" the in-app updater queries.
// Release builds set it with -X main.UpdateRepo=$GITHUB_REPOSITORY.
var UpdateRepo string

// RuntimeBaseURL is the preferred directory that hosts dsh-runtime-*.zip
// (and optional SHA256SUMS). Overridden by DSH_RUNTIME_BASE_URL, or
// -X main.RuntimeBaseURL=. When UpdateRepo is set, GitHub Releases is
// tried next if this host fails.
var RuntimeBaseURL string

func currentVersion() string {
	if strings.TrimSpace(Version) != "" {
		return strings.TrimSpace(Version)
	}
	return strings.TrimSpace(pinnedVersion)
}
