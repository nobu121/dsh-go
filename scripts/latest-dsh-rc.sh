#!/usr/bin/env bash
# Print the newest vX.Y.Z-rc.N from deepseek-ai/deepseek-harness that is
# also on npm and differs from dsh.version. Empty stdout = nothing to do.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN="$(tr -d '[:space:]' < "$ROOT/dsh.version" 2>/dev/null || true)"
API="https://api.github.com/repos/deepseek-ai/deepseek-harness/releases?per_page=30"

releases="$(curl -fsSL -H "Accept: application/vnd.github+json" "$API")"
newest="$(printf '%s' "$releases" | python3 -c '
import json, re, sys
pin = sys.argv[1].strip()
releases = json.load(sys.stdin)
if not isinstance(releases, list):
    sys.exit(0)
pat = re.compile(r"^v(\d+)\.(\d+)\.(\d+)-rc\.(\d+)$")
found = []
for rel in releases:
    if rel.get("draft"):
        continue
    tag = rel.get("tag_name") or ""
    m = pat.match(tag)
    if not m:
        continue
    found.append((tuple(int(x) for x in m.groups()), tag[1:]))
if not found:
    sys.exit(0)
newest = max(found)[1]
if newest == pin:
    sys.exit(0)
print(newest)
' "$PIN")"

if [[ -z "${newest:-}" ]]; then
  exit 0
fi

if ! npm view "@deepseek-ai/dsh@${newest}" version >/dev/null 2>&1; then
  echo "github has ${newest} but npm does not; skip" >&2
  exit 0
fi

printf '%s\n' "$newest"
