# Deepseek Harness GO

[English](README.md) | **中文**

给已经会用 `dsh` 的人一个 **4 MB** 原生窗口。[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) 的官方界面，配置就是 `~/.dsh`，不另做市场、不另起一套设置。国内走 CNB。

**下载**

- macOS（Apple Silicon）— [`.dmg`](https://github.com/nobu121/dsh-go/releases/latest/download/dsh-go-darwin-arm64.dmg)
- Windows（x64）— 自安装 [`.zip`](https://github.com/nobu121/dsh-go/releases/latest/download/dsh-go-windows-amd64.zip)

校验和：[GitHub Releases](https://github.com/nobu121/dsh-go/releases/latest) · [CNB](https://cnb.cool/nobu121/dsh-go/-/releases)

非官方应用，与 DeepSeek 无关。应用图标及 DeepSeek 相关商标归杭州深度求索人工智能基础技术研究有限公司所有。

## 功能

- 用原生窗口打开 Harness。
- 若本机已全局安装 `dsh`，会直接复用；否则首次启动时自动下载运行时。
- 与 `dsh` 命令行共用配置、profile 和插件（`~/.dsh`）。
- 跟随 Harness 的主题，以及在 Harness 里设置的语言。
- 客户端和运行时可分别检查更新。可以立即更新，也可以稍后，继续用当前版本。
- 单实例运行。再次打开会回到已有窗口。
- Windows 下会创建桌面快捷方式，并出现在「已安装的应用」中。卸载会去掉应用、快捷方式和运行时。

## 为什么用这个

- 系统 WebView，安装包大约 4 MB，不带 Chromium。
- 低侵略性：窗口就是官方 UI，不另做市场、不改插件、不另起一套配置。
- 和 `dsh` 共用 `~/.dsh`；本机已有命令行就不会再装一份。
- 自动注入 `dsh` 命令，插件安装体验和命令行一样。
- 客户端和运行时分开更新。国内走 CNB，GitHub 不通也不会卡在启动。
- Windows 会出现在「已安装的应用」里，卸载干净。

## 许可

MIT。DeepSeek Harness 仍遵循其自身许可证。
