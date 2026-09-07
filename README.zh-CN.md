# Deepseek Harness GO

[English](README.md) | **中文**

[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) 的桌面客户端。
支持 **macOS** 和 **Windows**。本程序是非官方应用，与 DeepSeek 无关；应用图标及
DeepSeek 相关商标均归杭州深度求索人工智能基础技术研究有限公司所有。

## 功能

- 用原生窗口打开 Harness。
- 若本机已全局安装 `dsh`，会直接复用；否则首次启动时自动下载运行时。
- 与 `dsh` 命令行共用配置、profile 和插件（`~/.dsh`）。
- 跟随 Harness 的主题，以及在 Harness 里设置的语言。
- 客户端和运行时可分别检查更新。可以立即更新，也可以稍后，继续用当前版本。
- 单实例运行。再次打开会回到已有窗口。
- Windows 下会创建桌面快捷方式，并出现在「已安装的应用」中。卸载会去掉应用、快捷方式和运行时。

安装包在 [GitHub Releases](https://github.com/nobu121/dsh-go/releases)。

## 许可

MIT。DeepSeek Harness 仍遵循其自身许可证。
