# Deepseek Harness GO

**English** | [中文](README.zh-CN.md)

A desktop client for [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness).
Available for **macOS** and **Windows**. This is an unofficial app, not affiliated
with DeepSeek. The app icon and DeepSeek trademarks belong to
杭州深度求索人工智能基础技术研究有限公司.

## Features

- Opens Harness in a native window.
- Reuses a globally installed `dsh` if one is already on the machine; otherwise downloads a runtime on first launch.
- Shares profiles, plugins, and settings with the `dsh` CLI (`~/.dsh`).
- Follows the Harness theme, and the language set in Harness.
- Checks for client and runtime updates separately. Update now, or later and keep using the current versions.
- Runs as a single instance. Opening the app again focuses the existing window.
- On Windows, installs a desktop shortcut and an Installed apps entry. Uninstall removes the app, shortcut, and runtime.

Installers are on [GitHub Releases](https://github.com/nobu121/dsh-go/releases).

## License

MIT. DeepSeek Harness remains under its own licenses.
