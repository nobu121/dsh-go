#!/usr/bin/env bash
# Drop files the bundled runtime does not need at launch.
# Safe to re-run. Keeps this platform's native binaries.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="${1:-${DSH_VENDOR_DIR:-$ROOT/vendor/dsh}}"
if [[ ! -d "$DEST" ]]; then
  echo "missing $DEST" >&2
  exit 1
fi

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) keep_os=darwin ;;
  Linux) keep_os=linux ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) keep_os=win32 ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) keep_arch=arm64 ;;
  x86_64|amd64) keep_arch=x64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
keep="${keep_os}-${keep_arch}"

before="$(du -sk "$DEST" | awk '{print $1}')"

if [[ -x "$DEST/node" ]] && command -v strip >/dev/null; then
  strip -S "$DEST/node" || true
  if [[ "$keep_os" == "darwin" ]] && command -v codesign >/dev/null; then
    codesign --force --sign - "$DEST/node" >/dev/null
  fi
fi

NM="$DEST/node_modules"
if [[ -d "$NM" ]]; then
  find "$NM" \( \
    -name '*.map' -o \
    -name '*.md' -o \
    -name '*.markdown' -o \
    -name '*.pdb' -o \
    -name 'CHANGELOG*' -o \
    -name 'README*' \
  \) -type f -delete

  if [[ "$keep_os" != "win32" ]]; then
    find "$NM" \( -name '*.exe' -o -name '*.dll' \) -type f -delete
  fi

  # Type declarations and TS sources when a compiled sibling exists.
  python3 - "$NM" <<'PY'
import os, sys
root = sys.argv[1]
for dp, dns, fns in os.walk(root):
    for fn in fns:
        path = os.path.join(dp, fn)
        lower = fn.lower()
        if lower.endswith((".d.ts", ".d.mts", ".d.cts")):
            try:
                os.remove(path)
            except OSError:
                pass
            continue
        for src, compiled in ((".ts", ".js"), (".mts", ".mjs"), (".cts", ".cjs")):
            if lower.endswith(src) and not lower.endswith(".d" + src):
                sibling = path[: -len(src)] + compiled
                if os.path.exists(sibling):
                    try:
                        os.remove(path)
                    except OSError:
                        pass
                break
PY

  rm -rf "$NM/@types"

  if [[ -d "$NM/node-pty/prebuilds" ]]; then
    find "$NM/node-pty/prebuilds" -mindepth 1 -maxdepth 1 ! -name "$keep" -exec rm -rf {} +
  fi

  if [[ -d "$NM/@img/sharp-${keep_os}-${keep_arch}" || -d "$NM/@img/sharp-libvips-${keep_os}-${keep_arch}" ]]; then
    rm -rf "$NM/@img/sharp-wasm32"
  fi

  # Optional platform packages that are not this host.
  python3 - "$NM" "$keep_os" "$keep_arch" <<'PY'
import os, shutil, sys
root, keep_os, keep_arch = sys.argv[1], sys.argv[2], sys.argv[3]
plats = ("darwin", "linux", "win32", "windows")
arches = ("arm64", "x64", "amd64", "ia32", "x86", "wasm32")
for name in os.listdir(root):
    path = os.path.join(root, name)
    if not os.path.isdir(path):
        continue
    low = name.lower()
    has_plat = [p for p in plats if p in low]
    has_arch = [a for a in arches if a in low]
    if not has_plat:
        continue
    if keep_os not in has_plat and not (keep_os == "win32" and "windows" in has_plat):
        shutil.rmtree(path, ignore_errors=True)
        continue
    if has_arch and keep_arch not in has_arch and "amd64" not in has_arch:
        # x64 packages may be named amd64
        if keep_arch == "x64" and ("x64" in has_arch or "amd64" in has_arch):
            continue
        shutil.rmtree(path, ignore_errors=True)
PY

  find "$NM" -type d -empty -delete
fi

after="$(du -sk "$DEST" | awk '{print $1}')"
echo "pruned $DEST: $((before / 1024))M -> $((after / 1024))M (keep ${keep})"
