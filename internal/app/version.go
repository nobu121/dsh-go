package app

import "strings"

// The shell and the dsh runtime version independently. app.version moves when
// the shell changes; dsh.version records the runtime this build vendors and is
// only a fallback target when the runtime channel is unreachable. The live
// target comes from the channel (see channel.go).

var shellVersionFile string
var bundledDSHFile string

// Version overrides the shell product version. Release builds set it with
// -X dsh-go/internal/app.Version=$(cat app.version).
var Version string

// UpdateRepo is the GitHub "owner/repo" the in-app updater queries.
var UpdateRepo string

// RuntimeBaseURL is the preferred directory that hosts the runtime channel.
var RuntimeBaseURL string

// SetEmbeddedVersions injects the root app.version / dsh.version files.
func SetEmbeddedVersions(shell, dsh string) {
	shellVersionFile = shell
	bundledDSHFile = dsh
}

func shellVersion() string {
	if v := strings.TrimSpace(Version); v != "" {
		return v
	}
	return strings.TrimSpace(shellVersionFile)
}

func bundledDSHVersion() string {
	return strings.TrimSpace(bundledDSHFile)
}
