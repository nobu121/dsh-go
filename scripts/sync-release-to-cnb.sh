#!/usr/bin/env bash
# Mirror a GitHub (or local) release directory onto the CNB repo release.
# Usage: sync-release-to-cnb.sh <tag> <asset-dir>
# Env: CNB_TOKEN (required), CNB_REPO (default nobu121/dsh-go),
#      CNB_API (default https://api.cnb.cool), CNB_TARGET (default main)
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <tag> <asset-dir>" >&2
  exit 2
fi

TAG="$1"
DIR="$2"
CNB_API="${CNB_API:-https://api.cnb.cool}"
CNB_REPO="${CNB_REPO:-nobu121/dsh-go}"
CNB_TARGET="${CNB_TARGET:-main}"
TOKEN="${CNB_TOKEN:-}"

if [[ -z "$TOKEN" ]]; then
  echo "CNB_TOKEN is empty; skip CNB release sync" >&2
  exit 0
fi
if [[ ! -d "$DIR" ]]; then
  echo "asset dir not found: $DIR" >&2
  exit 1
fi

accept="application/vnd.cnb.api+json"
auth="token ${TOKEN}"

cnb_api() {
  local method="$1" path="$2"
  shift 2
  curl -sS -X "$method" \
    -H "Accept: ${accept}" \
    -H "Authorization: ${auth}" \
    -H "Content-Type: application/json" \
    "${CNB_API}${path}" \
    "$@"
}

json_field() {
  python3 -c 'import json,sys; print(json.load(sys.stdin).get(sys.argv[1]) or "")' "$1"
}

status_body="$(mktemp)"
trap 'rm -f "$status_body"' EXIT

code="$(curl -sS -o "$status_body" -w '%{http_code}' \
  -H "Accept: ${accept}" \
  -H "Authorization: ${auth}" \
  "${CNB_API}/${CNB_REPO}/-/releases/tags/${TAG}")"

if [[ "$code" == "200" ]]; then
  rid="$(json_field id < "$status_body")"
else
  notes="Shell-only dsh-go ${TAG#v} plus on-demand dsh-runtime zips for macOS (arm64) and Windows (amd64)."
  payload="$(python3 -c '
import json,sys
print(json.dumps({
  "tag_name": sys.argv[1],
  "name": sys.argv[1],
  "body": sys.argv[2],
  "target_commitish": sys.argv[3],
  "prerelease": True,
  "make_latest": "true",
}))
' "$TAG" "$notes" "$CNB_TARGET")"
  cnb_api POST "/${CNB_REPO}/-/releases" --data-binary "$payload" > "$status_body"
  rid="$(json_field id < "$status_body")"
fi

if [[ -z "$rid" || "$rid" == "None" ]]; then
  echo "failed to resolve CNB release id for ${TAG}:" >&2
  cat "$status_body" >&2
  exit 1
fi

echo "CNB release ${TAG} id=${rid}"

shopt -s nullglob
assets=("$DIR"/dsh-go-* "$DIR"/dsh-runtime-* "$DIR"/SHA256SUMS)
if [[ ${#assets[@]} -eq 0 ]]; then
  echo "no release assets in $DIR" >&2
  exit 1
fi

for f in "${assets[@]}"; do
  [[ -f "$f" ]] || continue
  name="$(basename "$f")"
  size="$(wc -c < "$f" | tr -d '[:space:]')"
  echo "upload ${name} (${size} bytes)"
  up_payload="$(python3 -c 'import json,sys; print(json.dumps({"asset_name":sys.argv[1],"size":int(sys.argv[2]),"overwrite":True}))' "$name" "$size")"
  cnb_api POST "/${CNB_REPO}/-/releases/${rid}/asset-upload-url" --data-binary "$up_payload" > "$status_body"
  upload_url="$(json_field upload_url < "$status_body")"
  verify_url="$(json_field verify_url < "$status_body")"
  if [[ -z "$upload_url" || -z "$verify_url" ]]; then
    echo "asset-upload-url failed for ${name}:" >&2
    cat "$status_body" >&2
    exit 1
  fi
  curl -sS -X PUT --data-binary "@${f}" \
    -H "Content-Type: application/octet-stream" \
    "$upload_url" >/dev/null
  code="$(curl -sS -o "$status_body" -w '%{http_code}' -X POST \
    -H "Accept: ${accept}" \
    -H "Authorization: ${auth}" \
    "$verify_url")"
  if [[ "$code" != "200" && "$code" != "201" && "$code" != "204" ]]; then
    echo "confirm failed for ${name} (HTTP ${code}):" >&2
    cat "$status_body" >&2
    exit 1
  fi
done

echo "synced ${TAG} to https://cnb.cool/${CNB_REPO}/-/releases/${TAG}"
