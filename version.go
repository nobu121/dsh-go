package main

import (
	_ "embed"
	"strings"
)

// The shell and the dsh runtime version independently. app.version moves when
// the shell changes; dsh.version records the runtime this build vendors and is
// only a fallback target when the runtime channel is unreachable. The live
// target comes from the channel (see channel.go).

//go:embed app.version
var shellVersionFile string

//go:embed dsh.version
var bundledDSHFile string

// Version overrides the shell product version. Release builds set it with
// -X main.Version=$(cat app.version).
var Version string

// UpdateRepo is the GitHub "owner/repo" the in-app updater queries.
// Release builds set it with -X main.UpdateRepo=$GITHUB_REPOSITORY.
var UpdateRepo string

// RuntimeBaseURL is the preferred directory that hosts the runtime channel
// (runtime.json, dsh-runtime-*.zip, SHA256SUMS). Overridden by
// DSH_RUNTIME_BASE_URL, or -X main.RuntimeBaseURL=. When UpdateRepo is set,
// the GitHub runtime channel tag is tried next if this host fails.
var RuntimeBaseURL string

func shellVersion() string {
	if v := strings.TrimSpace(Version); v != "" {
		return v
	}
	return strings.TrimSpace(shellVersionFile)
}

func bundledDSHVersion() string {
	return strings.TrimSpace(bundledDSHFile)
}
