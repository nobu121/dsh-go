# dsh-go

[English](README.md) | **中文**

面向 [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) 的薄
[Wails v3](https://v3.wails.io/) 桌面壳。

壳只做三件事：

1. 查找或拉取 `dsh` 运行时，并监督该进程。
2. 用带鉴权的启动 URL 打开 dsh 自己的 Web UI。
3. 退出时干净地带走整棵进程树。

发版目标：**macOS** 和 **Windows**。

## 运行时

可安装包是 **纯壳**——包里没有 Node 和 `@deepseek-ai/dsh`。启动时壳会用它能找到的
dsh（已有的全局安装，或此前下载过的运行时），都没有就一边显示准备页一边拉取。
已经装过 dsh 的开发者直接得到盖在自己 CLI 上的原生窗口。

下载来的运行时本身没有 `dsh` 可执行文件，壳会装一个 `dsh` shim 并登记到新开的
终端里。两种情况下需要调用 `dsh` 的插件都能正常工作。

`DSH_HOME` 默认为 `~/.dsh`，与 `npm i -g @deepseek-ai/dsh` 相同，所以应用和终端
共用同一份 profile 和插件。

## 更新

启动时在准备页检查有没有新的客户端版本。「立即更新」会从同一个 GitHub / CNB
tag 装上客户端和对应运行时，并先对两个源测速。

「稍后」继续用磁盘上已有的版本。启动阶段不会因为版本不一致而拒绝一个能跑的
运行时，断网也照样能起。

## 开发

```sh
bash scripts/sync-dsh.sh   # vendor/dsh: Node + @deepseek-ai/dsh@$(cat dsh.version)
go test -mod=mod ./...
go build -mod=mod -o dsh-go .
./dsh-go
```

`DSH_EXE` 可以把壳指向某个具体的 `dsh`，`DSH_REPO` 指向 dsh 源码目录，
`DSH_RUNTIME_BASE_URL` 指向一个放运行时资产的目录（`file:///…` 可用）以测下载路径。

## 打包

```sh
wails3 task package:dist      # macOS DMG 或 Windows exe
bash scripts/sync-dsh.sh
wails3 task package:runtime   # 频道用的运行时资产
```

两种应用制品都不含运行时。配上 Apple Developer ID 与公证密钥后，CI 会签名并公证
DMG，从而去掉 macOS「仍要打开」提示。

安装包与运行时资产发在 **GitHub Releases**，并同步到 **CNB Releases**
（`https://cnb.cool/nobu121/dsh-go/-/releases`）。日常开发远程仍是 CNB（`origin`）。

## 许可

MIT。DeepSeek Harness 仍遵循其自身许可证。
