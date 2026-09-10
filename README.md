# Kite

Cross-platform desktop V2Ray/Xray client (Wails + React/Tailwind, xray-core as a Go library).

## Status

The project structure is scaffolded by hand (Go, Node, and the Wails CLI were not
installed on the machine this was created on, so no `go`/`npm`/`wails` command has
been run yet). Everything below still needs to happen once those tools are installed.

## One-time toolchain setup

1. **Go** — https://go.dev/dl/ (1.21+). Windows also needs a C compiler for CGO
   (xray-core and some Wails dependencies use it) — install
   [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or use `winget install -e --id GoLang.Go`
   plus MSYS2's `mingw-w64-x86_64-gcc`.
2. **Node.js** (LTS) — https://nodejs.org, or `winget install OpenJS.NodeJS.LTS`.
3. **Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
   Then run `wails doctor` to confirm all platform dependencies (WebView2 on Windows
   is usually already present) are satisfied.

## First run after tools are installed

```bash
go mod tidy          # resolves the real xray-core/wails/uuid versions into go.sum
cd frontend && npm install && cd ..
wails dev             # generates frontend/wailsjs/*, starts the dev app with hot reload
```

`wails dev` is what generates `frontend/wailsjs/go/main/App.js` — the JS bridge that
`frontend/src/App.jsx` imports from. The app won't compile until that first run.

## What's here

- [go.mod](go.mod) — module `github.com/freeb5d/kite`, deps listed but not yet resolved
  (run `go mod tidy` to generate `go.sum`).
- [main.go](main.go) / [app.go](app.go) — Wails entrypoint and the bound `App` struct
  (Go methods callable from the frontend).
- [internal/xray](internal/xray) — `Manager` (start/stop/restart lifecycle),
  `BuildConfig` (profile → xray-core config), `Traffic` (stats). The actual
  xray-core instance wiring is stubbed with `TODO`s pending `go mod tidy`.
- [internal/profile](internal/profile) — link parsers for `vmess://`, `vless://`,
  `trojan://`, `ss://`, and a flat JSON file store under the OS config dir.
- [internal/system](internal/system) — per-OS system HTTP/SOCKS proxy toggling
  (`proxy_windows.go` via `netsh winhttp`, `proxy_darwin.go` via `networksetup`,
  `proxy_linux.go` via `gsettings`). Not yet wired into `Connect`/`Disconnect`.
- [frontend/](frontend) — Vite + React + Tailwind, with a minimal `App.jsx` UI:
  add a share link, list servers, connect/disconnect, status display.
- [wails.json](wails.json) — tells the Wails CLI how to build the frontend.

## Releases

`.github/workflows/release.yml` builds Kite for Windows, macOS, and Linux on
GitHub-hosted runners (which already have Go/Node) and attaches the binaries
to a GitHub Release. It fires on any pushed tag matching `v*`:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Known gaps / next steps

- `internal/system` proxy calls aren't called from `App.Connect`/`Disconnect` yet.
- `internal/xray/manager.go` doesn't actually start an xray-core instance yet —
  `BuildConfig` produces a placeholder `Config` type, not real xray-core conf types.
- No tests yet.
- TUN mode is deliberately out of scope for phase 1 (system proxy only).
