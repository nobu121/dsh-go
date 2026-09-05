# dsh-go

A thin [Wails v3](https://v3.wails.io/) desktop shell for
[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness).

The shell owns three jobs:

1. Find or fetch a `dsh` runtime, then supervise that process.
2. Open dsh's own Web UI at the authenticated startup URL.
3. Exit cleanly, taking the process tree with it.

The installable app is **shell-only**. Node and `@deepseek-ai/dsh` are not
inside the default package. At startup the shell:

1. Uses `DSH_EXE` if set.
2. Uses a `dsh` already on `PATH` (for example `npm i -g @deepseek-ai/dsh`).
3. Uses a previously downloaded runtime in
   `<UserConfigDir>/dsh-go/dsh-runtime` when its `VERSION` matches
   [`dsh.version`](dsh.version).
4. Otherwise downloads `dsh-runtime-<os>-<arch>.zip` from
   `DSH_RUNTIME_BASE_URL` (or the release-time `RuntimeBaseURL`).
5. Falls back to `DSH_REPO` or a **warm** `npx` cache. Cold `npx` is not
   the default, so a first launch does not stall on an empty npm cache.

Developers who already installed dsh get a native window over that CLI.
Everyone else gets a prep page while the runtime zip is fetched.

If the running dsh is a global install (or the cached runtime) older than
`dsh.version`, a title-bar capsule (`更新 dsh 到 x.y.z`) can upgrade it:
`npm i -g` for PATH installs, or a fresh zip for the cache. A warm-npx
session does not get a fake upgrade button. The shell's own updater
capsule (`更新到 x.y.z`) is separate.

Supported release targets: **macOS** and **Windows**.

## Layout

```
*.go, *.html     shell, supervisor, prep page, updater capsule
dsh.version      product / target @deepseek-ai/dsh pin
scripts/         sync / smoke / dist / runtime-zip helpers
build/darwin/    Info.plist + icons for the .app
build/windows/   exe resources (icon, manifest, version info)
.github/         detect rc + package updater zips
.agents/skills/  CNB project skills
```

## Development

```sh
bash scripts/sync-dsh.sh   # vendor/dsh: Node + @deepseek-ai/dsh@$(cat dsh.version)
go test -mod=mod ./...
go build -mod=mod -o dsh-go .
./dsh-go
```

`vendor/dsh` is a local compatibility runtime (same layout as the cache).
`DSH_HOME` defaults to `<UserConfigDir>/dsh-go/dsh-home`. Point
`DSH_RUNTIME_BASE_URL` at a directory of runtime zips to exercise the
download path (`file:///…` works).

## Package

```sh
wails3 task package:dist              # shell-only .app / zip / dmg
bash scripts/sync-dsh.sh
wails3 task package:runtime           # dsh-runtime-<os>-<arch>.zip
```

macOS writes `bin/dsh-go.app`, a people-facing `bin/dsh-go-darwin-<arch>.dmg`,
and a max-deflate `bin/dsh-go-darwin-<arch>.zip` for the in-app updater
(single top-level `.app`). Windows writes `dsh-go/dsh-go.exe` in
`dsh-go-windows-<arch>.zip`. Neither artifact includes `dsh-runtime`.

Release builds can set `-X main.UpdateRepo=owner/dsh-go` and
`-X main.RuntimeBaseURL=https://…`. Local builds leave both empty:
the app updater is skipped, and runtime download needs
`DSH_RUNTIME_BASE_URL`.

## Upstream pin

[`dsh.version`](dsh.version) is the product version and the target dsh
version for downloads and assisted upgrades. CI only follows `vX.Y.Z-rc.N`
from
[deepseek-ai/deepseek-harness](https://github.com/deepseek-ai/deepseek-harness/releases)
and skips tags that are not on npm yet. Alpha tags are ignored.

## License

MIT. DeepSeek Harness remains under its own licenses.
