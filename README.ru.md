<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **Кроссплатформенный десктопный клиент V2Ray / прокси.**
  Бэкенд на Wails + Go, фронтенд на React/Tailwind, xray-core/sing-box встроены как библиотеки Go.

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#скачать)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [Скачать](#скачать) · [Возможности](#возможности) · [Сборка из исходников](#сборка-из-исходников) · [Архитектура](#архитектура)

  [English](README.md) · [中文](README.zh.md) · [فارسی](README.fa.md) · [Türkçe](README.tr.md) · [العربية](README.ar.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · Русский
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## Скачать

Возьмите последнюю сборку на **[странице релизов](https://github.com/freeb5d/kite/releases/latest)**:

| Платформа | Файл |
| --- | --- |
| Windows (x64) | `kite-windows-amd64.exe` |
| Linux (x64) | `kite-linux-amd64` |
| macOS (Apple Silicon) | `kite-macos-arm64` |

Каждая платформа поставляется в виде одного переносимого исполняемого файла — без установщика, без прав администратора (сборка для macOS — это «сырой» бинарник, извлечённый из `.app`-пакета, который создаёт Wails; о предупреждении Gatekeeper, которое это вызывает, и как его обойти, см. [Известные ограничения](#известные-ограничения--дальнейшие-планы)). После установки приложение проверяет новые релизы при запуске и может обновиться в один клик.

## Возможности

- **Форматы ссылок**: `vmess://`, `vless://`, `trojan://`, `ss://` — вставьте ссылку для добавления, она будет разобрана и сохранена
- **Ссылки на подписку** — вставьте ссылку `http(s)://` на подписку (формат base64-списка ссылок, используемый V2RayN/V2RayNG/Shadowrocket), и все содержащиеся в ней серверы будут импортированы разом, объединённые в одну сворачиваемую группу в списке (подписка может содержать сотни серверов). Группа показывает информацию о тарифе/трафике/сроке действия, если провайдер её сообщает (через заголовок `Subscription-Userinfo` или фиктивные записи «info», которые некоторые провайдеры подмешивают в список ссылок), и имеет собственную кнопку синхронизации для повторного получения и обновления своих серверов
- **Два движка на выбор**, оба встроены как библиотеки Go (а не вызываемые внешние бинарники) — полный контроль жизненного цикла, без разбора стандартного вывода ради статистики. **xray-core** — движок по умолчанию (более широкая поддержка транспортов, особенно `tcp` с маскировкой `headerType=http`); **sing-box** — единственный с поддержкой режима TUN. Переключайте в любой момент в панели «О программе» (сначала отключитесь)
- **Транспорты**: TCP (включая маскировку `headerType=http` от xray-core), WebSocket и gRPC, с определением безопасности TLS/REALITY прямо из ссылки
- **Интеграция с системным прокси** — Подключение/Отключение автоматически переключает HTTP-прокси ОС (пользовательский реестр в Windows, без повышения прав)
- **Режим TUN (только Windows, только sing-box)** — направляет весь системный трафик через виртуальный сетевой адаптер (встроенный WinTun), а не только приложения, учитывающие настройку прокси. Требует прав администратора и выбранного движка sing-box; Kite может перезапустить себя с повышенными правами одним кликом
- **Kill switch (только Windows)** — блокирует весь исходящий трафик, кроме трафика самого Kite, пока активно соединение, с помощью пары правил брандмауэра Windows, так что приложение, игнорирующее системный прокси (или упавший процесс движка), не сможет утечь за пределы туннеля. Те же требования к правам администратора, что и у режима TUN
- **Значок в системном трее** — закрытие окна прячет его в трей вместо выхода из программы, так что активное соединение продолжает работать; меню трея содержит «Показать Kite», «Отключить» и «Выйти из Kite»
- **Встроенная диагностика** — кнопка «Тест» делает реальный запрос через туннель и показывает фактический результат; просмотрщик логов показывает отладочный лог активного движка прямо в приложении
- **Самообновление** — проверяет GitHub Releases при запуске, одним кликом скачивает, заменяет и перезапускается (панель «О программе» показывает версию Kite, активный движок и его версию)
- **Тёмная / светлая темы**, с поиском по списку серверов, переименованием на месте и удалением в один клик
- **8 языков**, переключаются в боковой панели (персидский использует встроенный шрифт Vazirmatn):
  - 🇬🇧 English (по умолчанию)
  - 🇨🇳 中文 (китайский)
  - 🇮🇷 فارسی (персидский)
  - 🇹🇷 Türkçe (турецкий)
  - 🇸🇦 العربية (арабский)
  - 🇫🇷 Français (французский)
  - 🇩🇪 Deutsch (немецкий)
  - 🇷🇺 Русский

## Сборка из исходников

### Однократная настройка инструментов

1. **Go 1.27+** — https://go.dev/dl/ . В Windows также нужен C-компилятор для CGO (используется Wails и некоторыми его зависимостями) — установите [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) или `winget install -e --id GoLang.Go`, а также `mingw-w64-x86_64-gcc` из MSYS2.
2. **Node.js (LTS)** — https://nodejs.org , или `winget install OpenJS.NodeJS.LTS`.
3. **Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### Запуск

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### Сборка релизного бинарника

```bash
wails build -tags webkit2_41
```

## Архитектура

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

Каждый привязанный Go-метод `App` (в [app.go](app.go)) становится вызываемой из фронтенда JS-функцией, как только `wails dev`/`wails build` генерирует `frontend/wailsjs/go/main/App.js`.

## Известные ограничения / дальнейшие планы

- **Режим TUN работает только с движком sing-box** — у xray-core (движка по умолчанию) здесь нет входящего TUN; переключение в режим TUN при выбранном xray-core завершается понятной ошибкой, предлагающей сначала сменить движок в панели «О программе».
- **Статистика трафика в реальном времени — заглушка на обоих движках** — цифры в панели «Показать больше» всегда показывают 0Б/с. И stats.Manager у xray-core, и `trafficcontrol.Manager` у sing-box (через `experimental.clash_api`/`v2ray_api`) требуют дополнительной интеграции, которая ещё не выполнена — следующий шаг для обоих движков.
- **macOS только для Apple Silicon (arm64)** — сборки для Intel нет. Если нужна, добавьте запись матрицы `darwin/amd64` рядом с `darwin/arm64` в `.github/workflows/release.yml`.
- **Режим TUN пока доступен только в Windows** и только для IPv4 — маршрут-исключение, предотвращающий зацикливание собственного восходящего соединения движка через адаптер TUN (`internal/system/route_windows.go`), покрывает только разрешённые IPv4-адреса; сервер, доступный только по IPv6, пока не будет работать в режиме TUN. Поддержка TUN для Linux/macOS в принципе возможна (собственный пакет `tun` в sing-box уже поддерживает обе платформы), но пока не подключена здесь.
- **Системный прокси в Linux поддерживает только GNOME** (`gsettings`) — для других окружений рабочего стола нужен собственный бэкенд в `internal/system/proxy_linux.go`.
- **Kill switch только для Windows** — реализован через правила `netsh advfirewall` (`internal/system/killswitch_windows.go`); для Linux/macOS нужен собственный бэкенд (`iptables`/`pfctl`), пока не реализовано (переключатель на этих платформах скрыт, как и режим TUN).
- **Родная кнопка сворачивания всё ещё просто сворачивает в панель задач** — Wails v2 не предоставляет хук для события сворачивания на уровне ОС, только закрытие окна (`OnBeforeClose`, которое использует функция трея). Только закрытие окна прячет его в трей.
- **Автоматических тестов пока нет.**
- **Нет подписи кода** — и Windows SmartScreen, и macOS Gatekeeper будут предупреждать о неподписанном бинарнике; это ожидаемо на данный момент. На macOS первый запуск скачанного бинарника требует `xattr -d com.apple.quarantine kite-macos-arm64` (или ПКМ → «Открыть»), чтобы обойти Gatekeeper, поскольку файл не нотаризован. В Windows антивирусы (включая Defender) иногда помещают `wintun.dll` в карантин сразу после того, как Kite записывает её рядом с собой, поскольку такая близкая к ядру сетевая DLL часто помечается эвристически — с этой же проблемой сталкиваются WireGuard, v2rayN и другие приложения на основе Wintun. Если режим TUN завершается ошибкой вида «файл не найден», добавьте папку Kite в исключения вашего антивируса.

## Участие в разработке

Issue и PR приветствуются. Актуальный список того, что реально осталось сделать, — в разделе [Известные ограничения](#известные-ограничения--дальнейшие-планы) выше.

## Лицензия

[MIT](LICENSE)
