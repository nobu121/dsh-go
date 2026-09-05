# dsh-go

A thin [Wails v3](https://v3.wails.io/) desktop shell for
[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness).

The shell owns three jobs:

1. Supervise the bundled `dsh` process.
2. Open dsh's own Web UI at the authenticated startup URL.
3. Exit cleanly, taking the process tree with it.

Users do not install Node or npx. Production builds ship official Node 24 plus
`@deepseek-ai/dsh` at the pin in [`dsh.version`](dsh.version). GitHub Actions
follows the newest upstream `-rc` tag, waits until that version is on npm,
smokes the runtime, and publishes updater zips. The app downloads an update in
the background and shows a title-bar capsule (`更新到 x.y.z`) when it is ready
to restart.

## Development

```sh
bash scripts/sync-dsh.sh   # vendor/dsh: Node + @deepseek-ai/dsh@$(cat dsh.version)
go build -o dsh-go .
./dsh-go
```

Without `vendor/dsh`, the shell falls back to `DSH_REPO` or
`scripts/run-dsh.sh` (`npx @deepseek-ai/dsh@pin`). `DSH_HOME` defaults to
`<UserConfigDir>/dsh-go/dsh-home`.

## Package

```sh
bash scripts/sync-dsh.sh
wails3 task package:update-zip
```

macOS writes `bin/dsh-go.app` (runtime under `Contents/Resources/dsh-runtime/`)
and `bin/dsh-go-darwin-<arch>.zip`. Windows writes a single top-level folder
zip: `dsh-go/dsh-go.exe` + `dsh-go/dsh-runtime/`.

Release builds set `-X main.UpdateRepo=owner/dsh-go` so the in-app updater
can see GitHub Releases. Local builds leave `UpdateRepo` empty and skip checks.

## Upstream pin

[`dsh.version`](dsh.version) is the product version and the bundled dsh
version. CI only follows `vX.Y.Z-rc.N` from
[deepseek-ai/deepseek-harness](https://github.com/deepseek-ai/deepseek-harness/releases)
and skips tags that are not on npm yet. Alpha tags are ignored.

## License

MIT. DeepSeek Harness remains under its own licenses.
