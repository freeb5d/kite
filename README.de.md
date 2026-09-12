<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **Ein plattformübergreifender Desktop-Client für V2Ray/Proxy.**
  Wails + Go Backend, React/Tailwind Frontend, xray-core/sing-box als eingebettete Go-Bibliotheken.

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#downloads)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [Download](#downloads) · [Funktionen](#funktionen) · [Aus dem Quellcode bauen](#aus-dem-quellcode-bauen) · [Architektur](#architektur)

  [English](README.md) · [中文](README.zh.md) · [فارسی](README.fa.md) · [Türkçe](README.tr.md) · [العربية](README.ar.md) · [Français](README.fr.md) · Deutsch · [Русский](README.ru.md)
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## Downloads

Hol dir die neueste Version von der **[Releases-Seite](https://github.com/freeb5d/kite/releases/latest)**:

| Plattform | Download |
| --- | --- |
| Windows (x64) | `kite-windows-amd64.exe` |
| Linux (x64) | `kite-linux-amd64` |
| macOS (Apple Silicon) | `kite-macos-arm64` |

Jede Plattform wird als einzelne portable ausführbare Datei ausgeliefert — kein Installer, keine Administratorrechte nötig (der macOS-Build ist die rohe Binärdatei, die aus dem von Wails erzeugten `.app`-Bundle extrahiert wurde; siehe [Bekannte Lücken](#bekannte-lücken--nächste-schritte) für die dadurch entstehende Gatekeeper-Meldung und wie man sie umgeht). Nach der Installation prüft die App beim Start auf neue Releases und kann sich mit einem Klick selbst aktualisieren.

## Funktionen

- **Linkformate**: `vmess://`, `vless://`, `trojan://`, `ss://` — Freigabelink einfügen, wird geparst und gespeichert
- **Abonnement-URLs** — füge einen `http(s)://`-Abonnementlink ein (das base64-Linklisten-Format, das V2RayN/V2RayNG/Shadowrocket verwenden), und jeder darin enthaltene Server wird auf einmal importiert, zusammengefasst in einer einklappbaren Gruppe in der Liste (ein Abonnement kann Hunderte von Servern enthalten). Die Gruppe zeigt Tarif-/Traffic-/Ablaufinformationen an, wenn der Anbieter sie meldet (über den `Subscription-Userinfo`-Header oder die gefälschten „info“-Einträge, die manche Anbieter in die Linkliste mischen), und hat einen eigenen Sync-Button, um ihre Server erneut abzurufen und zu aktualisieren
- **Zwei wählbare Engines**, beide als Go-Bibliothek eingebettet (keine ausgelagerten Binärdateien) — volle Kontrolle über den Lebenszyklus, kein Parsen von stdout für Statistiken. **xray-core** ist der Standard (breitere Transport-Unterstützung, insbesondere `tcp` mit `headerType=http`-Tarnung); **sing-box** ist die einzige mit TUN-Modus. Jederzeit im Über-Panel wechselbar (vorher trennen)
- **Transporte**: TCP (inklusive xray-cores `headerType=http`-Tarnung), WebSocket und gRPC, mit TLS/REALITY-Sicherheitserkennung direkt aus dem Link
- **Systemproxy-Integration** — Verbinden/Trennen schaltet den HTTP-Proxy des Betriebssystems automatisch um (benutzerbezogene Registrierung unter Windows, keine Rechteerweiterung nötig)
- **TUN-Modus (Windows, nur sing-box)** — leitet den gesamten Systemverkehr über einen virtuellen Netzwerkadapter (WinTun, mitgeliefert), statt nur Apps, die eine Proxy-Einstellung beachten. Erfordert Administratorrechte und die ausgewählte sing-box-Engine; Kite kann sich mit einem Klick selbst mit erhöhten Rechten neu starten
- **Notausschalter (Windows)** — blockiert während der Verbindung jeglichen ausgehenden Verkehr außer dem von Kite selbst, über ein Paar Windows-Firewallregeln, sodass eine App, die den Systemproxy ignoriert (oder ein abgestürzter Engine-Prozess), keinen Verkehr außerhalb des Tunnels durchsickern lassen kann. Gleiche Administratoranforderung wie der TUN-Modus
- **Systemleiste** — das Schließen des Fensters versteckt es in der Systemleiste, statt die App zu beenden, sodass eine aktive Verbindung weiterläuft; das Menü enthält Kite anzeigen, Trennen und Kite beenden
- **Integrierte Diagnose** — ein Test-Button stellt eine echte Anfrage durch den Tunnel und meldet das tatsächliche Ergebnis; ein Log-Viewer zeigt das eigene Debug-Log der aktiven Engine inline an
- **Selbstaktualisierung** — prüft beim Start auf GitHub Releases, lädt mit einem Klick herunter, tauscht aus und startet neu (das Über-Panel zeigt Kites Version, die aktive Engine und deren Version)
- **Dunkle / helle Themes**, mit durchsuchbarer Serverliste, Inline-Umbenennung und Entfernen mit einem Klick
- **8 Sprachen**, umschaltbar über die Seitenleiste (Persisch verwendet die mitgelieferte Vazirmatn-Schriftart):
  - 🇬🇧 English (Standard)
  - 🇨🇳 中文 (Chinesisch)
  - 🇮🇷 فارسی (Persisch)
  - 🇹🇷 Türkçe (Türkisch)
  - 🇸🇦 العربية (Arabisch)
  - 🇫🇷 Français (Französisch)
  - 🇩🇪 Deutsch
  - 🇷🇺 Русский (Russisch)

## Aus dem Quellcode bauen

### Einmalige Toolchain-Einrichtung

1. **Go 1.27+** — https://go.dev/dl/ . Windows benötigt außerdem einen C-Compiler für CGO (von Wails und einigen seiner Abhängigkeiten genutzt) — installiere [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) oder `winget install -e --id GoLang.Go` sowie MSYS2s `mingw-w64-x86_64-gcc`.
2. **Node.js (LTS)** — https://nodejs.org , oder `winget install OpenJS.NodeJS.LTS`.
3. **Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### Ausführen

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### Eine Release-Binärdatei bauen

```bash
wails build -tags webkit2_41
```

## Architektur

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

Jede gebundene Go-Methode auf `App` (in [app.go](app.go)) wird zu einer aus dem Frontend aufrufbaren JS-Funktion, sobald `wails dev`/`wails build` `frontend/wailsjs/go/main/App.js` generiert.

## Bekannte Lücken / nächste Schritte

- **Der TUN-Modus funktioniert nur mit der sing-box-Engine** — xray-core (die Standardeinstellung) hat hier kein TUN-Inbound; der Wechsel in den TUN-Modus bei ausgewähltem xray-core schlägt mit einer klaren Fehlermeldung fehl, die dich auffordert, zuerst im Über-Panel die Engine zu wechseln.
- **Live-Traffic-Statistiken sind bei beiden Engines nur ein Platzhalter** — die Zahlen im „Mehr anzeigen“-Panel zeigen immer 0B/s. Sowohl xray-cores stats.Manager als auch sing-boxs `trafficcontrol.Manager` (über `experimental.clash_api`/`v2ray_api`) benötigen zusätzliche Verdrahtung, die noch nicht erfolgt ist — ein Folgeschritt für beide Engines.
- **macOS nur Apple Silicon (arm64)** — kein Intel-Build. Falls benötigt, füge einen `darwin/amd64`-Matrixeintrag neben `darwin/arm64` in `.github/workflows/release.yml` hinzu.
- **Der TUN-Modus ist vorerst nur unter Windows** und nur für IPv4 verfügbar — die Ausnahmeroute, die verhindert, dass die eigene Upstream-Verbindung der Engine durch den von ihr gespeisten TUN-Adapter läuft (`internal/system/route_windows.go`), deckt nur aufgelöste IPv4-Adressen ab; ein Server, der nur über IPv6 erreichbar ist, funktioniert im TUN-Modus noch nicht. TUN-Unterstützung für Linux/macOS ist möglich (sing-boxs eigenes `tun`-Paket unterstützt bereits beide), ist hier aber noch nicht angebunden.
- **Der Linux-Systemproxy deckt nur GNOME ab** (`gsettings`) — andere Desktop-Umgebungen benötigen ihr eigenes Backend in `internal/system/proxy_linux.go`.
- **Der Notausschalter ist nur für Windows** — implementiert über `netsh advfirewall`-Regeln (`internal/system/killswitch_windows.go`); Linux/macOS benötigen ihr eigenes Backend (`iptables`/`pfctl`) und sind vorerst nicht implementiert (der Schalter ist dort ausgeblendet, genau wie der TUN-Modus).
- **Der native Minimieren-Button minimiert weiterhin nur in die Taskleiste** — Wails v2 stellt keinen Hook für das Minimieren-Ereignis auf Betriebssystemebene bereit, nur das Schließen des Fensters (`OnBeforeClose`, das die Systemleisten-Funktion nutzt). Nur das Schließen des Fensters versteckt es in der Systemleiste.
- **Noch keine automatisierten Tests.**
- **Keine Code-Signierung** — sowohl Windows SmartScreen als auch macOS Gatekeeper warnen bei einer unsignierten Binärdatei; das ist vorerst zu erwarten. Auf macOS erfordert das erste Ausführen der heruntergeladenen Binärdatei `xattr -d com.apple.quarantine kite-macos-arm64` (oder Rechtsklick → Öffnen), um Gatekeeper zu umgehen, da sie nicht notariell beglaubigt ist. Unter Windows quarantänisieren Antivirenprogramme (einschließlich Defender) manchmal `wintun.dll` direkt nachdem Kite sie neben sich selbst geschrieben hat, da eine kernelnahe Netzwerk-DLL wie diese häufig heuristisch markiert wird — dasselbe Problem, auf das WireGuard, v2rayN und andere Wintun-basierte Apps stoßen. Wenn der TUN-Modus mit einem Fehler in der Art „Datei nicht gefunden“ fehlschlägt, füge Kites Ordner zu den Ausnahmen deines Antivirenprogramms hinzu.

## Mitwirken

Issues und PRs sind willkommen. Siehe [Bekannte Lücken](#bekannte-lücken--nächste-schritte) oben für das, was tatsächlich noch zu tun ist.

## Lizenz

[MIT](LICENSE)
