#!/usr/bin/env bash
# Sign the .app with Developer ID (hardened runtime) and notarize the DMG.
# No-op when MACOS_CERTIFICATE_P12 is unset.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="${1:-}"
DMG="${2:-}"

if [[ -z "${MACOS_CERTIFICATE_P12:-}" ]]; then
  echo "macos-sign-notarize: no MACOS_CERTIFICATE_P12; leaving ad-hoc signature" >&2
  exit 0
fi
if [[ -z "$APP" || ! -d "$APP" ]]; then
  echo "usage: $0 <App.app> [App.dmg]" >&2
  exit 2
fi
if [[ -z "${MACOS_CERTIFICATE_PASSWORD:-}" ]]; then
  echo "macos-sign-notarize: MACOS_CERTIFICATE_PASSWORD is empty" >&2
  exit 1
fi

tmp="$(mktemp -d)"
cleanup() {
  security delete-keychain "$KEYCHAIN" >/dev/null 2>&1 || true
  rm -rf "$tmp"
}
trap cleanup EXIT

CERT="$tmp/dev-id.p12"
KEYCHAIN="$tmp/dsh-go-signing.keychain-db"
KEYCHAIN_PWD="$(openssl rand -base64 24)"
printf '%s' "$MACOS_CERTIFICATE_P12" | base64 --decode > "$CERT"

security create-keychain -p "$KEYCHAIN_PWD" "$KEYCHAIN"
security set-keychain-settings -lut 21600 "$KEYCHAIN"
security unlock-keychain -p "$KEYCHAIN_PWD" "$KEYCHAIN"
security import "$CERT" -k "$KEYCHAIN" -P "$MACOS_CERTIFICATE_PASSWORD" -T /usr/bin/codesign -T /usr/bin/security
security list-keychain -d user -s "$KEYCHAIN"
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$KEYCHAIN_PWD" "$KEYCHAIN" >/dev/null

IDENTITY="${MACOS_SIGN_IDENTITY:-}"
if [[ -z "$IDENTITY" ]]; then
  IDENTITY="$(security find-identity -v -p codesigning "$KEYCHAIN" | awk -F'"' '/Developer ID Application/{print $2; exit}')"
fi
if [[ -z "$IDENTITY" ]]; then
  echo "macos-sign-notarize: no Developer ID Application identity in certificate" >&2
  security find-identity -v -p codesigning "$KEYCHAIN" >&2 || true
  exit 1
fi
echo "signing as $IDENTITY"

ENTITLEMENTS="$ROOT/build/darwin/entitlements.plist"
codesign --force --deep --options runtime --timestamp \
  --entitlements "$ENTITLEMENTS" \
  --sign "$IDENTITY" \
  "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"

if [[ -n "$DMG" && -f "$DMG" ]]; then
  if [[ -z "${APPLE_ID:-}" || -z "${APPLE_TEAM_ID:-}" || -z "${APPLE_APP_SPECIFIC_PASSWORD:-}" ]]; then
    echo "macos-sign-notarize: app signed; missing APPLE_ID / TEAM_ID / APP_SPECIFIC_PASSWORD, skip notarize" >&2
    exit 0
  fi
  codesign --force --sign "$IDENTITY" --timestamp "$DMG"
  xcrun notarytool submit "$DMG" \
    --apple-id "$APPLE_ID" \
    --team-id "$APPLE_TEAM_ID" \
    --password "$APPLE_APP_SPECIFIC_PASSWORD" \
    --wait
  xcrun stapler staple "$DMG"
  xcrun stapler validate "$DMG"
fi
