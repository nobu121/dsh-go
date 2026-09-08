# Deepseek Harness GO

**English** | [中文](README.zh-CN.md)

A **4 MB** native window for [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness). Official UI only — no extra marketplace, no second config. Shares `~/.dsh` with the `dsh` CLI.

**Download**

- macOS (Apple Silicon) — [`.dmg`](https://github.com/nobu121/dsh-go/releases/latest/download/dsh-go-darwin-arm64.dmg)
- Windows (x64) — self-installing [`.zip`](https://github.com/nobu121/dsh-go/releases/latest/download/dsh-go-windows-amd64.zip)

Checksums: [GitHub Releases](https://github.com/nobu121/dsh-go/releases/latest) · [CNB](https://cnb.cool/nobu121/dsh-go/-/releases)

Unofficial. Not affiliated with DeepSeek. The app icon and DeepSeek trademarks belong to 杭州深度求索人工智能基础技术研究有限公司.

## Features

- Opens Harness in a native window.
- Reuses a globally installed `dsh` if one is already on the machine; otherwise downloads a runtime on first launch.
- Shares profiles, plugins, and settings with the `dsh` CLI (`~/.dsh`).
- Follows the Harness theme, and the language set in Harness.
- Checks for client and runtime updates separately. Update now, or later and keep using the current versions.
- Runs as a single instance. Opening the app again focuses the existing window.
- On Windows, installs a desktop shortcut and an Installed apps entry. Uninstall removes the app, shortcut, and runtime.

## Why this client

- Native WebView, about 4 MB to download. No Chromium.
- Unobtrusive: the official UI only — no extra marketplace, plugin layer, or second config.
- Same `~/.dsh` as the `dsh` CLI. Reuses a global install when one is already there.
- Puts `dsh` on PATH, so plugin install works the same as the CLI.
- Client and runtime update separately. Mirrors include CNB, so a blocked GitHub does not stall launch.
- On Windows it appears in Installed apps and uninstalls cleanly.

## License

MIT. DeepSeek Harness remains under its own licenses.
