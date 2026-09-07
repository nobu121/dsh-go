# dsh-go

**English** | [中文](README.zh-CN.md)

A thin [Wails v3](https://v3.wails.io/) desktop shell for
[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness).

The shell owns three jobs:

1. Find or fetch a `dsh` runtime, then supervise that process.
2. Open dsh's own Web UI at the authenticated startup URL.
3. Exit cleanly, taking the process tree with it.

Supported release targets: **macOS** and **Windows**.

## Runtime

The installable app is **shell-only** — Node and `@deepseek-ai/dsh` are not
in the package. On launch the shell uses whatever dsh it can find (an
existing global install, or a runtime it downloaded earlier) and otherwise
fetches a runtime while showing a prep page. Developers who already have
dsh installed just get a native window over their own CLI.

A downloaded runtime has no `dsh` binary of its own, so the shell installs
a `dsh` shim and registers it for new terminals. Plugins that shell out to
`dsh` work in both cases.

`DSH_HOME` defaults to `~/.dsh`, the same location `npm i -g @deepseek-ai/dsh`
uses, so the app and the terminal share one set of profiles and plugins.

## Updates

The client and the dsh runtime version independently. On launch the prep
page checks both, and one **立即更新** installs whichever is new (or both)
after a speed test between CNB and GitHub.

- **Client** (`app.version`) — manual full releases (`v0.2.0`, …).
- **Runtime** — rolling `runtime-latest` channel. A new dsh does not need a
  new client.

`稍后` continues with the version already on disk. Startup never rejects a
working runtime over its version, so an offline launch still works.

## Development

```sh
bash scripts/sync-dsh.sh   # vendor/dsh: Node + @deepseek-ai/dsh@$(cat dsh.version)
go test -mod=mod ./...
go build -mod=mod -o dsh-go .
./dsh-go
```

`DSH_EXE` points the shell at a specific `dsh`, `DSH_REPO` at a dsh checkout,
and `DSH_RUNTIME_BASE_URL` at a directory of runtime assets (`file:///…`
works) to exercise the download path.

## Package

```sh
wails3 task package:dist      # macOS DMG or Windows exe
bash scripts/sync-dsh.sh
wails3 task package:runtime   # runtime assets for the channel
```

Neither app artifact includes a runtime. Add Apple Developer ID + notary
secrets to have CI sign and notarize the DMG and skip the macOS Gatekeeper
prompt.

Packages and runtime assets are published on **GitHub Releases** and
mirrored to **CNB Releases** (`https://cnb.cool/nobu121/dsh-go/-/releases`).
Daily development remote stays on CNB (`origin`).

## License

MIT. DeepSeek Harness remains under its own licenses.
