# Changelog

All notable changes to Kite are documented here. Versions correspond to
[GitHub Releases](https://github.com/freeb5d/kite/releases).

## v0.9.3 — Force http/1.1 ALPN for WebSocket transport

WS links whose `alpn` param included `h2` (a common default from subscription
generators, alongside `http/1.1`/`h3`) still failed after v0.9.2's SNI fix: the TLS
handshake completed, but the connection was closed right after by the server -- a
TLS-terminating edge that prefers h2 (Cloudflare, notably) negotiated it from the
offered ALPN list, turning the connection into an HTTP/2 stream that sing-box's plain
HTTP/1.1 WebSocket-upgrade client can't work over. `internal/xray/config.go` now
forces `alpn: ["http/1.1"]` whenever the transport is `ws`, ignoring whatever ALPN
list the link specified for that case.

## v0.9.2 — Default TLS SNI to the server address

Plain `security=tls` links with no `sni` param (and `host` present but empty, as some
subscription generators emit) connected past the handshake but were closed by the server
around ~400ms in, because `internal/xray/config.go` left sing-box's outbound `tls.server_name`
unset in that case -- sing-box sends no SNI extension at all unless `server_name` is set,
which most TLS-terminating servers/CDNs reject since they can't route the connection. Under
xray-core this same link worked because it silently defaulted SNI to the server's own
address; sing-box now does the same.

## v0.9.1 — Fix REALITY connections after the sing-box migration

VLESS+REALITY servers failed to connect after v0.9.0 with a `connection download closed:
unknown version: 72` / `EOF` error in the log. Cause: REALITY only works because the
client's TLS handshake mimics a real browser (uTLS) instead of Go's own `crypto/tls` --
when a share link's `fp` (fingerprint) parameter was empty, `internal/xray/config.go`
wasn't enabling uTLS at all, so the reality server couldn't recognize the connection as
legitimate and fell back to serving its camouflage site in plaintext (the "unknown
version" error is Go's TLS parser choking on that plaintext HTTP response). Now defaults
`fp` to `chrome` whenever REALITY is enabled and the link didn't specify one.

## v0.9.0 — Switch engine from xray-core to sing-box

Kite's embedded proxy engine is now [sing-box](https://github.com/SagerNet/sing-box)
instead of xray-core. `internal/xray` keeps its historical name/package path (renaming it
wasn't worth the churn across the rest of the codebase) but its `manager.go`/`config.go`
are rewritten from scratch against sing-box's Go API: a `box.Box` instance built from
`option.Options` decoded via sing-box's own registry-aware JSON decoder
(`include.Context` + `json.UnmarshalContext`), instead of xray-core's `core.Instance` +
`conf.Config`.

- vmess, vless, trojan, and shadowsocks over tcp/ws/grpc, with TLS/REALITY/uTLS, all
  carried over.
- TUN mode carried over (sing-box's own `tun` package), including the retry-on-
  adapter-still-releasing logic and the antivirus/wintun.dll error hint from v0.8.5-8.
- **Regression**: sing-box has no equivalent of xray-core's `type=tcp&headerType=http`
  disguise -- a link relying on that specific obfuscation won't connect anymore. See
  README known gaps.
- **Regression**: live traffic stats (the "Show more" panel) are a stub again, always
  reading 0B/s -- xray-core's stats manager doesn't have a sing-box equivalent without
  enabling its clash-api, which isn't wired up yet. See README known gaps.
- `App.XrayVersion()` (About panel) now reads the resolved `sing-box` module version from
  the Go build info instead of xray-core's `core.Version()`.

## v0.8.8 — Retry TUN reconnect instead of guessing a fixed wait

- v0.8.7's fixed 800ms pause after a TUN disconnect wasn't reliably
  long enough -- reconnecting could still hit the previous session's
  Wintun adapter not finished releasing yet, with the same "already
  been completed" error. Replaced the guess with a real retry: a
  TUN-mode `Start()` now retries up to 5 times with increasing backoff
  specifically on that error, rebuilding a fresh `core.Instance` each
  attempt, instead of hoping one fixed delay was enough. If it's still
  failing after every retry, the error now says plainly that the
  previous session hadn't finished releasing its adapter in time.

## v0.8.7 — Fix reconnecting in TUN mode right after disconnecting

- **Fixed a real bug**: xray-core's TUN inbound doesn't finish tearing
  down the Wintun adapter/session synchronously within `Close()` --
  reconnecting in TUN mode immediately after disconnecting could hit
  the adapter/session still being released and fail with "An attempt
  was made to perform an initialization operation when initialization
  has already been completed." `Manager.Stop()` now pauses briefly
  after a TUN-mode disconnect to give it time to actually finish first.

## v0.8.6 — Fix relaunch still not reopening when connected

- **Fixed a real bug**: v0.8.4's fixed 1.5s wait for the old process to
  quit before the new one registers itself wasn't always enough --
  `RestartElevated` didn't disconnect (stop xray/TUN, clear the proxy,
  remove kill switch firewall rules) before relaunching, leaving all of
  that for `OnShutdown` to do *after* `Quit()`, eating into the new
  process's wait window. If you were connected (TUN and/or kill switch
  especially) when restarting elevated or updating, that cleanup could
  easily take longer than 1.5s, so the new process still saw the old
  one as "alive" and gave up -- same "nothing reopens" failure as
  before, just needing a slower shutdown to trigger it.
  `RestartElevated` now disconnects proactively before relaunching
  (`ApplyUpdate` already stopped xray/cleared the proxy, but was
  missing the kill switch cleanup -- added), and the wait window is
  bumped to 3s for extra margin.

## v0.8.5 — Clearer error when antivirus deletes wintun.dll

- TUN mode failing with a raw, unhelpful OS error like "The system
  cannot find the file specified" almost always means antivirus
  software (Windows Defender included) quarantined `wintun.dll` right
  after Kite wrote it next to itself -- a kernel-adjacent networking
  DLL like this commonly gets flagged heuristically, the same issue
  WireGuard/v2rayN and other Wintun-based apps run into. Kite now
  detects this specific case and wraps the error with a clear
  explanation and the fix (add Kite's folder to antivirus exclusions)
  instead of leaving the user to guess at a generic Windows error.

## v0.8.4 — Fix Restart as admin not reopening Kite at all

- **Fixed a real bug**: v0.8.3 fixed the *old* process exiting the wrong
  way, but the race was still there from the other side -- spawning the
  new (elevated, or updated) process is near-instant, while the old one
  quitting isn't *quite* as instant. The new process's own
  `SingleInstanceLock` registration could still run first, see the
  (about to die) old process as "already running", and defer to it
  instead of actually starting -- then the old process finished quitting
  anyway, leaving nothing running at all: click "Restart as admin", UAC
  prompt, accept, and no Kite window ever comes back.
  `RelaunchElevated`/`update.Apply` now pass a `--kite-relaunch-wait`
  flag; the new process waits 1.5s before registering itself, giving
  the old one time to actually finish quitting first.

## v0.8.3 — Fix duplicate windows from Restart as admin / self-update

Two real bugs, both around the same root cause:

- Since closing the window keeps Kite running in the tray (v0.8.0)
  instead of quitting, launching the exe again (e.g. double-clicking
  it, or a "Restart as admin"/self-update relaunch) risked a second
  process colliding with the still-running first one over native
  resources -- WinTun, the system proxy registry keys -- producing
  native errors like "An attempt was made to perform an initialization
  operation when initialization has already been completed" and two
  overlapping windows. Added Wails' `SingleInstanceLock`: a second
  launch now just asks the already-running instance to show itself.
- That exposed a second bug: "Restart as admin" and the self-update
  relaunch both exited the old process via a raw `os.Exit(0)`, which
  bypasses Wails' own shutdown path -- including whatever releases the
  `SingleInstanceLock`'s IPC listener. The freshly-launched new process
  (elevated, or updated) could still see the dying old one as "already
  running" and get treated as a second instance instead of actually
  starting, leaving two windows open with neither one elevated or
  updated. Both now quit via `wailsruntime.Quit`, matching how the
  tray's own Quit already worked correctly.

## v0.8.1 — Fix tray right-click menu

- The system tray added in v0.8.0 didn't show its menu on
  right-click -- `energye/systray` doesn't do that automatically, it
  has to be wired up explicitly via `SetOnRClick` calling
  `menu.ShowMenu()`.

## v0.8.0 — Kill switch, system tray, right-click paste fix

- **Fixed right-click paste**: Wails disables the browser's native
  right-click context menu in production builds by default, so
  right-click did nothing anywhere in the app -- including pasting a
  link into the Add Server field. Re-enabled via
  `options.App.EnableDefaultContextMenu: true`.
- **Persian text now always uses Vazirmatn**, not just when Persian is
  the active app language -- e.g. "فارسی" as a label inside the
  language picker while some other language is selected now renders
  correctly too (`[lang='fa']` alongside the existing `[data-lang='fa']`).
- **Kill switch (Windows)**: a new toggle next to Proxy/TUN blocks all
  outbound traffic except Kite's own while connected, via a
  `netsh advfirewall` rule pair (allow Kite's own process, block
  everything else) -- loopback is exempt from Windows Firewall
  filtering regardless, so the local HTTP/SOCKS proxy keeps working.
  Needs administrator privileges, same as TUN mode. The block rules are
  only removed on a deliberate Disconnect (or app quit) -- not if xray
  crashes while connected, which is the point of a kill switch.
- **System tray**: closing the window now hides it to the tray instead
  of quitting, so an active connection (and kill switch, if on) keeps
  running. The tray icon's menu has Show Kite, Disconnect (shown only
  while connected), and Quit Kite; left-clicking the icon also restores
  the window. Only "Quit Kite" actually exits the app.

## v0.7.1 — Real traffic stats

- **Traffic stats are wired up for real** — `internal/xray/stats.go`'s
  `Traffic()` used to always return zero. `buildJSON` now turns on
  `policy.system.statsOutboundUplink`/`Downlink` so xray-core registers
  the "proxy" outbound's traffic counters, and `Manager` reads them
  from its own `stats.Manager` feature.
- The connect panel gets a collapsible "Show more" panel (closed by
  default) with **Live traffic** (current up/down speed, polled once a
  second while connected) and **Total traffic** (cumulative session
  bytes).

## v0.7.0 — Show the embedded xray-core version

- The About panel now shows the embedded xray-core version next to
  Kite's own version (e.g. "v0.7.0 · xray-core 26.3.27"), via
  `core.Version()` from xray-core itself.

## v0.6.9 — macOS builds are back

- Re-enabled macOS in the release matrix, on `macos-14` (Apple Silicon)
  instead of the old `macos-13` (Intel), which had very long GitHub
  runner queue times. Ships as `kite-macos-arm64` — the raw binary
  pulled out of the `.app` bundle Wails produces, unsigned and
  unnotarized, so the first run needs
  `xattr -d com.apple.quarantine kite-macos-arm64` or right-click →
  Open to get past Gatekeeper.
- Self-update now recognizes `darwin` and looks for the
  `kite-macos-arm64` release asset (previously only Windows/Linux were
  wired up, so self-update silently had nothing to offer on macOS).

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
