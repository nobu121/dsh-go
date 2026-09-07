package app

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dshVersionToken = regexp.MustCompile(`\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?`)

func parseDSHVersion(raw string) string {
	return dshVersionToken.FindString(strings.TrimSpace(raw))
}

func dshCLIVersion(exe string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	ver := parseDSHVersion(string(out))
	if ver == "" {
		return "", fmt.Errorf("could not parse dsh version from %q", bytes.TrimSpace(out))
	}
	return ver, nil
}

func compareDSHVersion(a, b string) int {
	pa, oka := parseVersionParts(a)
	pb, okb := parseVersionParts(b)
	if !oka && !okb {
		return strings.Compare(a, b)
	}
	if !oka {
		return -1
	}
	if !okb {
		return 1
	}
	for i := 0; i < 3; i++ {
		if pa.core[i] != pb.core[i] {
			if pa.core[i] < pb.core[i] {
				return -1
			}
			return 1
		}
	}
	// A release without prerelease is newer than one with.
	if pa.pre == "" && pb.pre != "" {
		return 1
	}
	if pa.pre != "" && pb.pre == "" {
		return -1
	}
	if pa.pre == pb.pre {
		return 0
	}
	if pa.rc != pb.rc && (strings.HasPrefix(pa.pre, "rc") || pa.pre == "") &&
		(strings.HasPrefix(pb.pre, "rc") || pb.pre == "") && (pa.rc > 0 || pb.rc > 0) {
		if pa.rc < pb.rc {
			return -1
		}
		return 1
	}
	return strings.Compare(pa.pre, pb.pre)
}

type versionParts struct {
	core [3]int
	pre  string
	rc   int
}

func parseVersionParts(v string) (versionParts, bool) {
	v = parseDSHVersion(v)
	if v == "" {
		return versionParts{}, false
	}
	core, pre, _ := strings.Cut(v, "-")
	var p versionParts
	p.pre = pre
	nums := strings.Split(core, ".")
	if len(nums) < 3 {
		return versionParts{}, false
	}
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(nums[i])
		if err != nil {
			return versionParts{}, false
		}
		p.core[i] = n
	}
	if strings.HasPrefix(pre, "rc.") {
		p.rc, _ = strconv.Atoi(strings.TrimPrefix(pre, "rc."))
	} else if strings.HasPrefix(pre, "rc") {
		p.rc, _ = strconv.Atoi(strings.TrimPrefix(pre, "rc"))
	}
	return p, true
}

func shouldOfferDSHUpdate(installed, target string) bool {
	if installed == "" || target == "" || installed == target {
		return false
	}
	return compareDSHVersion(installed, target) < 0
}

func globalUpgradeArgs(ver string) []string {
	return []string{"install", "-g", "@deepseek-ai/dsh@" + ver}
}

func upgradeGlobalDSH(ctx context.Context, ver string) error {
	npm, err := lookNamed("npm")
	if err != nil {
		return fmt.Errorf("未找到 npm，无法更新全局 dsh")
	}
	cmd := exec.CommandContext(ctx, npm, globalUpgradeArgs(ver)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("npm i -g 失败: %s", msg)
	}
	return nil
}

func installedDSHVersion(src resolvedDSH) string {
	switch src.Kind {
	case sourcePath:
		if src.Path == "" {
			return ""
		}
		ver, err := dshCLIVersion(src.Path)
		if err != nil {
			return ""
		}
		return ver
	case sourceCache, sourceBundled:
		return readRuntimeVersion(src.Path)
	default:
		return ""
	}
}

// canUpdateDSH covers the bundled runtime too: a newer runtime downloads into
// the cache, which resolveLaunch prefers over the bundled copy, so the runtime
// shipped inside the app can move without a new shell build.
func canUpdateDSH(kind dshSourceKind) bool {
	switch kind {
	case sourcePath, sourceCache, sourceBundled:
		return true
	}
	return false
}
