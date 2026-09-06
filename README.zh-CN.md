# dsh-go

[English](README.md) | **中文**

面向 [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) 的薄
[Wails v3](https://v3.wails.io/) 桌面壳。

壳只做三件事：

1. 查找或拉取 `dsh` 运行时，并监督该进程。
2. 用带鉴权的启动 URL 打开 dsh 自己的 Web UI。
3. 退出时干净地带走整棵进程树。

可安装包是 **纯壳**。默认包里没有 Node 和 `@deepseek-ai/dsh`。启动时按这个顺序找运行时：

1. 若设置了 `DSH_EXE`，用它。
2. 用 PATH 上已有的 `dsh`（例如 `npm i -g @deepseek-ai/dsh`）。
3. 使用此前下载到 `<UserConfigDir>/dsh-go/dsh-runtime` 的缓存，且其 `VERSION` 与 [`dsh.version`](dsh.version) 一致。
4. 否则从 `DSH_RUNTIME_BASE_URL`（或发版时写入的 `RuntimeBaseURL`）下载 `dsh-runtime-<os>-<arch>.zip`。
5. 再退回 `DSH_REPO` 或 **已热过的** `npx` 缓存。默认不会冷跑 `npx`，避免首次启动卡在空 npm 缓存上。

已经装过 dsh 的开发者会得到盖在该 CLI 上的原生窗口。其他人会先看到准备页，同时拉取 runtime zip。

若正在跑的 dsh 是全局安装（或缓存 runtime）且比 `dsh.version` 旧，标题栏胶囊（`更新 dsh 到 x.y.z`）可以升级：PATH 安装走 `npm i -g`，缓存走重新下载 zip。热 npx 会话不会出现假的升级按钮。壳自己的更新胶囊（`更新到 x.y.z`）是另一条路径。

发版目标：**macOS** 和 **Windows**。

## 目录

```
*.go, *.html     壳、监督器、准备页、更新胶囊
dsh.version      产品 / 目标 @deepseek-ai/dsh 版本钉
scripts/         sync / smoke / dist / runtime-zip 脚本
build/darwin/    .app 的 Info.plist 与图标
build/windows/   exe 资源（图标、清单、版本信息）
.github/         探测 npm latest + GitHub Actions 打包/发版
```

## 开发

```sh
bash scripts/sync-dsh.sh   # vendor/dsh: Node + @deepseek-ai/dsh@$(cat dsh.version)
go test -mod=mod ./...
go build -mod=mod -o dsh-go .
./dsh-go
```

`vendor/dsh` 是本地兼容运行时（布局与缓存相同）。
`DSH_HOME` 默认为 `<UserConfigDir>/dsh-go/dsh-home`。把
`DSH_RUNTIME_BASE_URL` 指到一堆 runtime zip 的目录即可测下载路径（`file:///…` 可用）。

## 打包

```sh
wails3 task package:dist              # 纯壳 .app / zip / dmg
bash scripts/sync-dsh.sh
wails3 task package:runtime           # dsh-runtime-<os>-<arch>.zip
```

macOS 写出 `bin/dsh-go.app`、给人下的 `bin/dsh-go-darwin-<arch>.dmg`，以及给应用内更新器用的高压缩 `bin/dsh-go-darwin-<arch>.zip`（顶层只有一个 `.app`）。Windows 在 `dsh-go-windows-<arch>.zip` 里放 `dsh-go/dsh-go.exe`。这些制品都不含 `dsh-runtime`。

给人下的包以及更新器 / runtime 资产发在 **GitHub Releases**。日常开发远程仍是 CNB（`origin`）。

发版构建会写入 `-X main.UpdateRepo=$GITHUB_REPOSITORY` 和
`-X main.RuntimeBaseURL=https://github.com/<owner>/<repo>/releases/download/v<ver>`。
本地构建两项都为空：跳过应用更新器，runtime 下载需要 `DSH_RUNTIME_BASE_URL`。

## 上游版本钉

[`dsh.version`](dsh.version) 是产品版本，也是下载和辅助升级的目标 dsh 版本。CI 跟随 npm 上 `@deepseek-ai/dsh` 的 `latest` dist-tag（目前是 rc；他们发布正式 `x.y.z` 后会自动跟上）。`alpha` 标签不跟。

## 许可

MIT。DeepSeek Harness 仍遵循其自身许可证。
