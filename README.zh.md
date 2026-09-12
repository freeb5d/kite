<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **跨平台桌面版 V2Ray / 代理客户端。**
  Wails + Go 后端，React/Tailwind 前端，xray-core/sing-box 作为 Go 库嵌入。

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#下载)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [下载](#下载) · [功能](#功能) · [从源码构建](#从源码构建) · [架构](#架构)

  [English](README.md) · 中文 · [فارسی](README.fa.md) · [Türkçe](README.tr.md) · [العربية](README.ar.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md)
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## 下载

从 **[发布页面](https://github.com/freeb5d/kite/releases/latest)** 获取最新版本：

| 平台 | 下载文件 |
| --- | --- |
| Windows (x64) | `kite-windows-amd64.exe` |
| Linux (x64) | `kite-linux-amd64` |
| macOS (Apple Silicon) | `kite-macos-arm64` |

每个平台都是单个便携可执行文件——无需安装程序，无需管理员权限（macOS 版本是从 Wails 生成的 `.app` 包中提取出的原始二进制文件；关于这带来的 Gatekeeper 提示以及如何绕过它，见[已知限制](#已知限制--后续计划)）。安装后，程序会在启动时检查新版本，一键即可自我更新。

## 功能

- **链接格式**：`vmess://`、`vless://`、`trojan://`、`ss://` —— 粘贴分享链接即可解析并保存
- **订阅链接** —— 粘贴一个 `http(s)://` 订阅链接（V2RayN/V2RayNG/Shadowrocket 使用的 base64 链接列表格式），其中包含的所有服务器都会一次性导入，并折叠成列表中的一个可展开分组（一个订阅可能包含数百个服务器）。当服务商通过 `Subscription-Userinfo` 响应头或混在链接列表中的伪造“info”条目报告套餐/流量/到期信息时，该分组会显示这些信息，并自带一个同步按钮用于重新获取和刷新其服务器列表
- **两种可切换的引擎**，均作为 Go 库嵌入（而非外部调用的二进制文件）——完整的生命周期控制，无需解析标准输出来获取统计信息。**xray-core** 是默认引擎（支持更广泛的传输方式，尤其是带 `headerType=http` 伪装的 `tcp`）；**sing-box** 是唯一支持 TUN 模式的引擎。可随时在“关于”面板中切换（需先断开连接）
- **传输方式**：TCP（包括 xray-core 的 `headerType=http` 伪装）、WebSocket 和 gRPC，并直接从链接中检测 TLS/REALITY 安全设置
- **系统代理集成** —— 连接/断开会自动切换操作系统的 HTTP 代理（Windows 上使用每用户注册表项，无需提升权限）
- **TUN 模式（仅限 Windows，仅 sing-box）** —— 将所有系统流量通过虚拟网络适配器（内置 WinTun）路由，而不仅仅是遵循代理设置的应用。需要管理员权限并选择 sing-box 引擎；Kite 可以一键以提升权限的方式重启自身
- **Kill Switch（仅限 Windows）** —— 连接期间通过一对 Windows 防火墙规则阻止除 Kite 自身外的所有出站流量，这样即使某个应用忽略系统代理（或引擎进程崩溃），也无法在隧道之外泄漏流量。与 TUN 模式有相同的管理员权限要求
- **系统托盘** —— 关闭窗口会将其隐藏到托盘而不是退出程序，因此活动连接会继续保持；托盘菜单包含“显示 Kite”、“断开连接”和“退出 Kite”
- **内置诊断** —— “测试”按钮会通过隧道发起一次真实请求并报告实际结果；日志查看器可内联显示当前引擎自身的调试日志
- **自我更新** —— 启动时检查 GitHub Releases，一键下载、替换并重启（“关于”面板显示 Kite 版本、当前引擎及其版本）
- **深色/浅色主题**，配有可搜索的服务器列表、内联重命名和一键移除
- **8 种语言**，可在侧边栏切换（波斯语使用内置的 Vazirmatn 字体）：
  - 🇬🇧 English（默认）
  - 🇨🇳 中文
  - 🇮🇷 فارسی（波斯语）
  - 🇹🇷 Türkçe（土耳其语）
  - 🇸🇦 العربية（阿拉伯语）
  - 🇫🇷 Français（法语）
  - 🇩🇪 Deutsch（德语）
  - 🇷🇺 Русский（俄语）

## 从源码构建

### 一次性工具链准备

1. **Go 1.27+** —— https://go.dev/dl/ 。Windows 上还需要一个支持 CGO 的 C 编译器（Wails 及其部分依赖会用到）——安装 [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) 或 `winget install -e --id GoLang.Go` 加上 MSYS2 的 `mingw-w64-x86_64-gcc`。
2. **Node.js（LTS）** —— https://nodejs.org ，或 `winget install OpenJS.NodeJS.LTS`。
3. **Wails CLI**：
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### 运行

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### 构建发布版二进制文件

```bash
wails build -tags webkit2_41
```

## 架构

```
kite/
├── main.go / app.go        Wails entrypoint + the App struct (Go methods
│                            exposed to the frontend via the JS bridge)
├── internal/
│   ├── xray/                 Engine facade (package/dir keeps the historical
│   │   │                      name from when it *was* the xray-core wrapper
│   │   │                      -- see CHANGELOG). Dispatches Start/Stop/
│   │   │                      Status/Traffic to whichever concrete engine
│   │   │                      below is currently selected (settings.json),
│   │   │                      so app.go doesn't know which one is active.
│   │   └── facade.go
│   ├── engine/
│   │   ├── xraycore/          xray-core lifecycle (the default engine) --
│   │   │                      manager.go builds a core.Instance via
│   │   │                      serial.LoadJSONConfig + core.New; config.go
│   │   │                      builds the JSON from a server profile
│   │   └── singbox/           sing-box lifecycle (the only engine with TUN
│   │       ├── manager.go     support) -- Start/Stop/Restart a real box.Box
│   │       ├── config.go      Server profile -> sing-box JSON config,
│   │       │                  decoded via sing-box's own registry-aware
│   │       │                  JSON decoder (see include.Context)
│   │       ├── stats.go       Traffic counters (stub, see below)
│   │       └── tun_windows.go Writes the embedded wintun.dll next to the
│   │                          exe (TUN mode needs it alongside the binary)
│   ├── profile/              vmess/vless/trojan/ss link parsing +
│   │                         JSON-file server storage
│   ├── system/                per-OS HTTP/SOCKS system proxy toggling +
│   │   │                      TUN-mode support (Windows)
│   │   ├── proxy_windows.go   HKCU Internet Settings (no admin needed)
│   │   ├── proxy_darwin.go    networksetup
│   │   ├── proxy_linux.go     gsettings (GNOME)
│   │   ├── elevate_windows.go Check/request administrator privileges
│   │   ├── route_windows.go   Exception route so xray's own upstream
│   │   │                      connection doesn't loop through the TUN
│   │   │                      adapter it's feeding (see that file's docs)
│   │   └── wintun/            Embedded Wintun driver DLL (dual GPLv2/MIT,
│   │                          see WINTUN_LICENSE.txt in that directory)
│   └── update/                self-update: check GitHub Releases, download
│                               + swap the running executable, relaunch
├── frontend/                 Vite + React + Tailwind (CSS-variable theming
│                              for dark/light), talks to app.go via Wails
└── .github/workflows/         CI: builds + publishes GitHub Releases on
    release.yml                 any pushed `v*` tag
```

`App`（见 [app.go](app.go)）上每一个绑定的 Go 方法，一旦 `wails dev`/`wails build` 生成 `frontend/wailsjs/go/main/App.js`，就会成为前端可调用的 JS 函数。

## 已知限制 / 后续计划

- **TUN 模式仅在 sing-box 引擎下可用** —— 默认引擎 xray-core 在此没有 TUN 入站；在选择 xray-core 时切换到 TUN 模式会失败，并明确提示先在“关于”面板中切换引擎。
- **两种引擎的实时流量统计都是占位实现** —— “显示更多”面板的数字始终显示 0B/s。xray-core 的 stats.Manager 和 sing-box 的 `trafficcontrol.Manager`（通过 `experimental.clash_api`/`v2ray_api`）都还需要额外的接入工作——这是两种引擎共同的后续事项。
- **macOS 仅支持 Apple Silicon（arm64）** —— 没有 Intel 构建版本。如有需要，可在 `.github/workflows/release.yml` 中添加与 `darwin/arm64` 并列的 `darwin/amd64` 矩阵项。
- **TUN 模式目前仅限 Windows** 且仅支持 IPv4 —— 用于防止引擎自身上游连接循环回 TUN 适配器的例外路由（`internal/system/route_windows.go`）目前只覆盖已解析的 IPv4 地址；只能通过 IPv6 访问的服务器暂时无法在 TUN 模式下使用。Linux/macOS 的 TUN 支持是可行的（sing-box 自带的 `tun` 包已支持两者），只是尚未接入。
- **Linux 系统代理仅支持 GNOME**（`gsettings`）—— 其他桌面环境需要在 `internal/system/proxy_linux.go` 中实现各自的后端。
- **Kill Switch 仅限 Windows** —— 通过 `netsh advfirewall` 规则实现（`internal/system/killswitch_windows.go`）；Linux/macOS 需要各自的后端（`iptables`/`pfctl`），目前尚未实现（该开关在这些平台上被隐藏，与 TUN 模式相同）。
- **原生最小化按钮仍然只是最小化到任务栏** —— Wails v2 没有提供操作系统级最小化事件的钩子，只有窗口关闭事件（`OnBeforeClose`，托盘功能使用的就是它）。只有关闭窗口才会隐藏到托盘。
- **目前还没有自动化测试。**
- **没有代码签名** —— Windows SmartScreen 和 macOS Gatekeeper 都会对未签名的二进制文件发出警告；目前属于预期情况。在 macOS 上，首次运行下载的二进制文件需要执行 `xattr -d com.apple.quarantine kite-macos-arm64`（或右键 → 打开）来绕过 Gatekeeper，因为它没有经过公证。在 Windows 上，杀毒软件（包括 Defender）有时会在 Kite 将 `wintun.dll` 写入自身旁边后立即将其隔离，因为这类与内核相关的网络 DLL 很容易被启发式检测标记——WireGuard、v2rayN 等基于 Wintun 的应用也会遇到同样的问题。如果 TUN 模式因类似“找不到文件”的错误而失败，请将 Kite 所在文件夹加入杀毒软件的排除列表。

## 贡献

欢迎提交 Issue 和 PR。实际待办事项见上方[已知限制](#已知限制--后续计划)。

## 许可证

[MIT](LICENSE)
