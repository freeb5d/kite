# Changelog

All notable changes to Kite are documented here. Versions correspond to
[GitHub Releases](https://github.com/freeb5d/kite/releases).

## v0.6.8 — Fix tcp+HTTP-header-obfuscation connections

- **Fixed a real bug**: a link with `type=tcp&headerType=http` (TCP
  transport disguised behind a plaintext HTTP request, so an
  HTTP-sniffing front end in front of the real server doesn't reject
  the connection) was being sent as plain, undisguised TCP -- Kite
  never built a `tcpSettings.header` block at all. The front end saw
  what looked like garbage and answered with a plain HTTP response
  instead of proxying through, producing the same
  `unexpected response version... actually 72` error as the REALITY
  bug fixed in v0.6.7, but for a different reason. `streamSettings` now
  builds the HTTP disguise header (method/path/Host/User-Agent) for
  `network=tcp` links that specify `headerType=http`.

## v0.6.7 — Fix REALITY connections

- **Fixed a real bug**: a `security=reality` link was being sent to
  xray-core as plain `security: "tls"`. Since REALITY isn't TLS -- it
  proxies unauthenticated clients straight through to the real
  (camouflaged) destination it's impersonating -- the client ended up
  doing an ordinary TLS handshake against that real site and got back
  a plain HTTP response instead of VLESS, failing every connection
  with `unexpected response version... actually 72` (`H` from
  `HTTP/1.1`). `streamSettings` now builds a proper `realitySettings`
  block (publicKey/shortId/spiderX/fingerprint/serverName) for
  `security=reality` links instead of collapsing it into `tls`.

## v0.6.6 — Usage progress bar for subscriptions

- The subscription group card now shows a usage strip below the
  header: a filled progress bar for traffic used vs. total, and the
  plan's expiry date, instead of squeezed text on one line.

## v0.6.5 — Subscription usage info and a sync button

- The subscription group row now shows plan info when the provider
  reports it: traffic used/total and days left, parsed from the
  `Subscription-Userinfo` response header when present, falling back to
  the human-readable "info" line(s) some providers mix into the link
  list itself (previously discarded outright).
- Added a **sync button** on the subscription group (next to remove) to
  re-fetch that subscription and refresh its server list in place.

## v0.6.4 — Subscription groups, info-node filtering, and a real ID bug fix

- **Fixed a real bug**: `profile.Store.Add` assigned a server its ID on
  an internal copy and never returned it, so both `AddProfileFromLink`
  and the new `AddSubscription` handed the frontend a server with an
  empty ID. Selecting and connecting to a server added in the current
  session (before any list refresh) failed with `no server with id ""`.
  `Store.Add` now returns the stored copy, ID included.
- **Subscription grouping**: servers imported from one subscription URL
  are folded into a single collapsible row (closed by default) instead
  of flooding the list — a subscription can carry hundreds of servers.
- **Info-node filtering**: some subscription providers mix in fake
  vless://-style entries whose host is a placeholder like
  `dontUseThis` and whose name carries plan/expiry/traffic text, just
  so it shows up as a row in clients that list every node. Those are
  now recognized and skipped instead of being imported as dead servers.

## v0.6.3 — Subscription URL support

- The "Add server" field now also accepts a subscription URL
  (`http://`/`https://`), not just a single share link. Kite fetches it,
  decodes the standard base64 link-list format used by V2RayN/V2RayNG/
  Shadowrocket-compatible providers, and imports every server it can
  parse in one go. Entries that fail to parse are skipped rather than
  failing the whole import.

## v0.6.2 — Vazirmatn font for Persian

- Persian (فارسی) now renders with the bundled Vazirmatn font instead of
  the system default.

## v0.6.1 — Keep LTR layout for Persian and Arabic

- Persian and Arabic keep full text translation, but the UI layout no
  longer flips to right-to-left.

## v0.6.0 — Multi-language UI

- Added a language picker (sidebar, next to About): English (default),
  中文, فارسی, Türkçe, العربية, Français, Deutsch.

## v0.5.3 — About panel rework, MIT license

- **About panel**: fuller description, a copyable repo link, and quick
  Repository / Releases / Report-an-issue shortcuts, instead of a single
  bare GitHub link.
- Added an **MIT license** (`LICENSE`) and linked it from the README.
- Set the GitHub repo's description and topics (`v2ray`, `xray`, `vpn`,
  `vmess`, `vless`, `trojan`, `shadowsocks`, `wails`, `tun`, …) so the
  project actually surfaces in GitHub search.

## v0.5.2 — Fix country flag not rendering on Windows

- Windows' emoji font (Segoe UI Emoji) doesn't support regional-indicator
  flag emoji — it fell back to printing the raw two-letter code (e.g.
  "FI FI") instead of a flag. The Test result now renders a real flag
  image instead of relying on emoji support.

## v0.5.1 — Test result shows exit IP, country, and real delay

- The Test button's result used to just say "OK (HTTP 204)" against
  Cloudflare's `generate_204` endpoint — no proof of *which* server you
  were actually exiting through. It now hits Cloudflare's trace endpoint
  and shows the VPN server's real exit IP, its country, and the actual
  round-trip delay of the request made through the tunnel.

## v0.5.0 — TUN mode (Windows), update download progress

- **TUN mode**: a Proxy/TUN toggle now appears on Windows. TUN mode routes
  *all* system traffic through a virtual network adapter instead of just
  apps that honor a proxy setting, using xray-core's own built-in `proxy/tun`
  inbound with the bundled Wintun driver. Since creating a system-level
  network adapter needs administrator rights, Kite checks elevation and
  offers a one-click "Restart as admin" when it isn't already elevated.
  An exception route is added for the VPN server's own resolved IP(s)
  *before* xray's TUN inbound brings up its default route, so xray's own
  outbound connection doesn't loop back through the adapter it's feeding.
  Windows-only and IPv4-only for this first pass — see README known gaps.
- **Update progress**: the self-update banner now shows a live percentage
  and progress bar while downloading, instead of a static "Updating…".

## v0.4.1 — Branding and docs

- Added the Kite app logo/icon throughout the app and README.
- Rewrote the README (features, architecture, downloads, known gaps) and
  started this CHANGELOG.

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
