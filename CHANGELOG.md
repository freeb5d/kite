# Changelog

All notable changes to Kite are documented here. Versions correspond to
[GitHub Releases](https://github.com/freeb5d/kite/releases).

## v0.4.0 — About panel, self-update, dark/light theme

- **Self-update**: Kite now checks GitHub Releases on launch. When a newer
  build is available, a banner offers a one-click update — it downloads the
  new executable, swaps it in for the running one, and relaunches
  automatically. No installer or admin rights needed, since Kite is a
  portable single-file executable.
- **About panel**: version, a link to the GitHub repo, and a manual
  "Check for updates" button (info icon in the sidebar rail).
- **Dark / light theme toggle**: the whole UI now runs on CSS custom
  properties instead of hardcoded colors, with a sun/moon toggle in the
  rail that persists your choice.

## v0.3.0 — UI redesign

- Replaced the original single-column list-and-button layout with a
  three-pane app shell: a slim icon rail, a searchable server list with
  inline rename/remove on hover, and a connect panel centered on a large
  power-dial button.

## v0.2.7 — The real connect fix

- Root-caused and fixed the actual reason connections were failing:
  xray-core's WebSocket transport shares its TLS config with Go's
  `net/http`, so a link's `alpn=http/1.1,h2,h3` (forwarded as-is) made Go
  negotiate HTTP/2 instead of performing the WebSocket upgrade, and every
  dial failed with `websocket: protocol "h2" was given but is not
  supported`. ALPN is now skipped for `ws`/`websocket` transport.

## v0.2.6 — Real connectivity diagnostics

- Added a **Test** button that makes an actual HTTP request through the
  local proxy and reports the real result (or the exact Go error) instead
  of a "running" status that says nothing about whether traffic is
  actually flowing.
- xray-core's own debug log is now written to disk and viewable inline via
  a **Show log** toggle — this is what ultimately located the ALPN bug
  above.

## v0.2.5 — TLS/transport detection fix

- `vless://`, `trojan://`, and `ss://` links use `security=`/`type=` query
  parameters, but the config builder was only checking the vmess-style
  `tls`/`network` key names. TLS was silently never enabled for those
  three link types, so every connection attempt went out in plaintext to a
  TLS-only port and hung. Both key spellings are checked now.

## v0.2.4 — Windows system proxy fix

- `netsh winhttp set proxy` requires an elevated process and only affects
  WinHTTP-based services anyway (not the browsers people actually use).
  Replaced it with writing the per-user `HKCU\...\Internet Settings`
  registry keys directly — no elevation needed, and it's what browsers and
  most Windows apps actually read.

## v0.2.3 — Self-healing profile storage

- A saved server profile with an empty ID (from very early testing, before
  ID assignment was solid) made Connect/Rename fail with
  `no server with id ""`. The profile store now backfills a fresh UUID for
  any entry missing one and persists the fix automatically on load.

## v0.2.2 — VLESS/VMess outbound config fix

- `go.mod` pins xray-core v1.8.24, whose `VLessOutboundConfig` only
  understands the classic nested `vnext` array — not the flat
  `address`/`port`/`id` shorthand added in later xray-core versions.
  Outbound configs are now built with the classic nested `vnext`/`servers`
  shape, which every xray-core version understands.
- Added **Remove** and **Rename** for saved server profiles.

## v0.2.1 — Error surfacing

- `Connect`/`Disconnect` failures were previously swallowed silently by
  the frontend — a real backend error looked exactly like "nothing
  happens." Errors now render in a banner, and the app shows its own
  version (baked in via `-ldflags` at build time) in a footer.

## v0.2.0 — Real xray-core integration

- `internal/xray/manager.go` now actually starts and stops a real
  xray-core `core.Instance`, instead of just flipping an internal status
  flag.
- `internal/xray/config.go` builds the standard Xray JSON config shape and
  runs it through xray-core's own `conf.Config.Build()` — the same path
  xray-core itself uses to load a config file.
- `app.go`'s `Connect`/`Disconnect` now call into `internal/system` to
  actually toggle the OS proxy.

## v0.1.0 — Initial scaffold

- Project structure: Wails entrypoint (`main.go`/`app.go`), `internal/xray`
  (lifecycle stubs), `internal/profile` (link parsing for `vmess://`,
  `vless://`, `trojan://`, `ss://`, plus JSON file storage), `internal/system`
  (per-OS proxy toggle stubs), and a React + Tailwind frontend.
- GitHub Actions release workflow that builds Windows and Linux binaries
  and publishes them to GitHub Releases on any pushed `v*` tag (macOS was
  dropped from the matrix after the `macos-13` runner pool showed very
  long queue times).
