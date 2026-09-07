#!/usr/bin/env bash
# Rewrite packaging metadata from app.version, so the shell's product version
# has one source of truth instead of being copied into three files by hand.
# Usage: sync-app-version.sh [version]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VER="${1:-}"
if [[ -z "$VER" ]]; then
  VER="$(tr -d '[:space:]' < "$ROOT/app.version")"
fi
if [[ -z "$VER" ]]; then
  echo "app.version is empty" >&2
  exit 1
fi
# Windows FILEVERSION is numeric only, so drop any prerelease suffix.
FILE_VER="${VER%%-*}"

rewrite() {
  local file="$1"
  shift
  local tmp
  tmp="$(mktemp)"
  awk "$@" "$file" > "$tmp"
  mv "$tmp" "$file"
}

rewrite "$ROOT/build/config.yml" -v ver="$VER" '
  /^  version:/ { print "  version: \"" ver "\""; next }
  { print }
'

rewrite "$ROOT/build/windows/info.json" -v ver="$VER" -v fver="$FILE_VER" '
  /"file_version"/ { sub(/"file_version": *"[^"]*"/, "\"file_version\": \"" fver "\"") }
  /"ProductVersion"/ { sub(/"ProductVersion": *"[^"]*"/, "\"ProductVersion\": \"" ver "\"") }
  { print }
'

# CFBundleShortVersionString / CFBundleVersion each carry the value on the
# following <string> line.
rewrite "$ROOT/build/darwin/Info.plist" -v ver="$VER" '
  hit { sub(/<string>[^<]*<\/string>/, "<string>" ver "</string>"); hit = 0 }
  /<key>CFBundleShortVersionString<\/key>/ { hit = 1 }
  /<key>CFBundleVersion<\/key>/ { hit = 1 }
  { print }
'

echo "synced packaging metadata to ${VER} (file_version ${FILE_VER})"
