<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **یک کلاینت دسکتاپ چندسکویی برای V2Ray / پروکسی.**
  بک‌اند Wails + Go، فرانت‌اند React/Tailwind، با xray-core/sing-box به‌صورت کتابخانه‌ی Go تعبیه‌شده.

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#دانلود)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [دانلود](#دانلود) · [ویژگی‌ها](#ویژگیها) · [ساخت از سورس](#ساخت-از-سورس) · [معماری](#معماری)

  [English](README.md) · [中文](README.zh.md) · فارسی · [Türkçe](README.tr.md) · [العربية](README.ar.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md)
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## دانلود

آخرین نسخه را از **[صفحه‌ی انتشارها](https://github.com/freeb5d/kite/releases/latest)** بگیرید:

| پلتفرم | فایل دانلود |
| --- | --- |
| ویندوز (x64) | `kite-windows-amd64.exe` |
| لینوکس (x64) | `kite-linux-amd64` |
| مک‌اواس (Apple Silicon) | `kite-macos-arm64` |

هر پلتفرم به‌صورت یک فایل اجرایی پرتابل ارائه می‌شود — بدون نصب‌کننده و بدون نیاز به دسترسی ادمین(نسخه‌ی مک‌اواس همان باینری خام استخراج‌شده از بسته‌ی `.app` است که Wails می‌سازد؛ برای پیام Gatekeeper که این موضوع ایجاد می‌کند و راه دور زدن آن به [محدودیت‌های شناخته‌شده](#محدودیت‌های-شناخته‌شده--قدم‌های-بعدی) مراجعه کنید). پس از نصب، هنگام اجرا نسخه‌های جدید را بررسی می‌کند و می‌تواند خودش را با یک کلیک به‌روزرسانی کند.

## ویژگی‌ها

- **فرمت لینک‌ها**: `vmess://`، `vless://`، `trojan://`، `ss://` — لینک اشتراک‌گذاری را جای‌گذاری کنید تا تجزیه و ذخیره شود
- **لینک‌های اشتراک (Subscription)** — یک لینک اشتراک `http(s)://` (همان فرمت لیست لینک base64 که V2RayN/V2RayNG/Shadowrocket استفاده می‌کنند) را جای‌گذاری کنید تا همه‌ی سرورهای آن یک‌جا وارد شده و در یک گروه قابل‌جمع‌شدن در لیست جای بگیرند (یک اشتراک می‌تواند صدها سرور داشته باشد). وقتی سرویس‌دهنده اطلاعات پلن/ترافیک/انقضا را گزارش کند (از طریق هدر `Subscription-Userinfo` یا ورودی‌های ساختگی «info» که برخی سرویس‌دهنده‌ها داخل لیست لینک قرار می‌دهند) این گروه آن را نشان می‌دهد، و یک دکمه‌ی همگام‌سازی مخصوص خودش دارد برای واکشی و به‌روزرسانی دوباره‌ی سرورهایش
- **دو موتور قابل‌انتخاب**، هر دو به‌صورت کتابخانه‌ی Go تعبیه‌شده (نه فایل باینری جدا) — کنترل کامل چرخه‌ی عمر، بدون نیاز به پارس‌کردن خروجی استاندارد برای آمار. **xray-core** پیش‌فرض است (پشتیبانی گسترده‌تر از انتقال‌ها، به‌ویژه `tcp` با پوشش `headerType=http`)؛ **sing-box** تنها موتوری‌ست که حالت TUN را دارد. هر زمان از پنل «درباره» قابل تعویض است (ابتدا قطع اتصال کنید)
- **انتقال‌ها**: TCP (شامل پوشش `headerType=http` مربوط به xray-core)، وب‌سوکت و gRPC، همراه با تشخیص امنیت TLS/REALITY مستقیم از لینک
- **یکپارچگی با پروکسی سیستم** — اتصال/قطع اتصال به‌طور خودکار پروکسی HTTP سیستم‌عامل را تغییر می‌دهد (روی ویندوز از رجیستری هر کاربر، بدون نیاز به دسترسی بالا)
- **حالت TUN (فقط ویندوز، فقط sing-box)** — تمام ترافیک سیستم را از طریق یک آداپتور شبکه‌ی مجازی (WinTun، همراه برنامه) عبور می‌دهد، نه فقط برنامه‌هایی که تنظیمات پروکسی را رعایت می‌کنند. نیاز به دسترسی ادمین و انتخاب موتور sing-box دارد؛ Kite می‌تواند خودش را با یک کلیک با دسترسی بالا مجدداً اجرا کند
- **Kill Switch (فقط ویندوز)** — در زمان اتصال، تمام ترافیک خروجی به‌جز ترافیک خود Kite را از طریق یک جفت قانون فایروال ویندوز مسدود می‌کند، تا برنامه‌ای که پروکسی سیستم را نادیده می‌گیرد (یا فرآیند موتور کرش کرده) نتواند بیرون از تونل ترافیک درز کند. همان نیاز به دسترسی ادمین مانند حالت TUN
- **آیکن سینی سیستم** — بستن پنجره آن را به‌جای خروج، به سینی سیستم پنهان می‌کند تا اتصال فعال ادامه یابد؛ منوی سینی شامل نمایش Kite، قطع اتصال، و خروج از Kite است
- **ابزارهای تشخیصی داخلی** — دکمه‌ی Test یک درخواست واقعی از طریق تونل ارسال کرده و نتیجه‌ی واقعی را گزارش می‌دهد؛ نمایشگر لاگ، لاگ اشکال‌زدایی موتور فعال را به‌صورت درون‌خطی نشان می‌دهد
- **به‌روزرسانی خودکار** — هنگام اجرا، GitHub Releases را بررسی می‌کند، با یک کلیک دانلود، جایگزین و مجدداً اجرا می‌کند (پنل «درباره» نسخه‌ی Kite، موتور فعال و نسخه‌ی آن را نشان می‌دهد)
- **تم تیره/روشن**، همراه با لیست سرور قابل‌جست‌وجو، تغییرنام درون‌خطی، و حذف با یک کلیک
- **۸ زبان**، قابل تغییر از نوار کناری (فارسی از فونت داخلی Vazirmatn استفاده می‌کند):
  - 🇬🇧 English (پیش‌فرض)
  - 🇨🇳 中文 (چینی)
  - 🇮🇷 فارسی
  - 🇹🇷 Türkçe (ترکی)
  - 🇸🇦 العربية (عربی)
  - 🇫🇷 Français (فرانسوی)
  - 🇩🇪 Deutsch (آلمانی)
  - 🇷🇺 Русский (روسی)

## ساخت از سورس

### آماده‌سازی یک‌باره‌ی ابزارها

1. **Go نسخه‌ی ۱.۲۷ به بالا** — https://go.dev/dl/ . در ویندوز همچنین به یک کامپایلر C برای CGO نیاز است (Wails و برخی وابستگی‌هایش از آن استفاده می‌کنند) — [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) یا `winget install -e --id GoLang.Go` به همراه `mingw-w64-x86_64-gcc` از MSYS2 را نصب کنید.
2. **Node.js (نسخه‌ی LTS)** — https://nodejs.org ، یا `winget install OpenJS.NodeJS.LTS`.
3. **Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### اجرا

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### ساخت فایل باینری انتشار

```bash
wails build -tags webkit2_41
```

## معماری

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

هر متد Go متصل روی `App` (در [app.go](app.go))، به‌محض این‌که `wails dev`/`wails build` فایل `frontend/wailsjs/go/main/App.js` را تولید کند، به یک تابع JS قابل‌فراخوانی از فرانت‌اند تبدیل می‌شود.

## محدودیت‌های شناخته‌شده / قدم‌های بعدی

- **حالت TUN فقط زیر موتور sing-box کار می‌کند** — xray-core (پیش‌فرض) در اینجا ورودی TUN ندارد؛ تغییر به حالت TUN وقتی xray-core انتخاب شده، با خطایی واضح که می‌گوید ابتدا موتور را در پنل «درباره» عوض کنید، شکست می‌خورد.
- **آمار زنده‌ی ترافیک در هر دو موتور فقط یک نمونه‌ی جایگزین است** — عددهای پنل «نمایش بیشتر» همیشه ۰B/s نشان می‌دهند. هم stats.Manager مربوط به xray-core و هم `trafficcontrol.Manager` مربوط به sing-box (از طریق `experimental.clash_api`/`v2ray_api`) به اتصال بیشتری نیاز دارند که هنوز انجام نشده — یک کار باقی‌مانده برای هر دو موتور.
- **macOS فقط روی Apple Silicon (arm64)** — بدون نسخه‌ی اینتل. در صورت نیاز، یک ورودی ماتریس `darwin/amd64` را کنار `darwin/arm64` در `.github/workflows/release.yml` اضافه کنید.
- **حالت TUN فعلاً فقط روی ویندوز** و فقط IPv4 کار می‌کند — مسیر استثنایی که مانع از حلقه‌زدن اتصال بالادستی خود موتور از طریق آداپتور TUN می‌شود (`internal/system/route_windows.go`) فقط آدرس‌های IPv4 حل‌شده را پوشش می‌دهد؛ سروری که فقط از طریق IPv6 در دسترس است هنوز در حالت TUN کار نمی‌کند. پشتیبانی TUN برای Linux/macOS ممکن است (بسته‌ی `tun` خود sing-box از هر دو پشتیبانی می‌کند) اما هنوز اینجا وصل نشده.
- **پروکسی سیستم لینوکس فقط GNOME را پوشش می‌دهد** (`gsettings`) — محیط‌های دسکتاپ دیگر به بک‌اند مخصوص خودشان در `internal/system/proxy_linux.go` نیاز دارند.
- **Kill Switch فقط روی ویندوز** — از طریق قوانین `netsh advfirewall` پیاده‌سازی شده (`internal/system/killswitch_windows.go`)؛ لینوکس/مک‌اواس به بک‌اند مخصوص خودشان (`iptables`/`pfctl`) نیاز دارند و فعلاً پیاده‌سازی نشده‌اند (کلید تغییر آن در این پلتفرم‌ها پنهان است، مثل حالت TUN).
- **دکمه‌ی مینیمایز بومی هنوز فقط به نوار وظیفه مینیمایز می‌کند** — Wails v2 هوکی برای رویداد مینیمایز در سطح سیستم‌عامل ارائه نمی‌دهد، فقط بستن پنجره (`OnBeforeClose`، که ویژگی سینی از آن استفاده می‌کند). فقط بستن پنجره آن را به سینی پنهان می‌کند.
- **فعلاً هیچ تست خودکاری وجود ندارد.**
- **بدون امضای کد** — هم Windows SmartScreen و هم macOS Gatekeeper نسبت به فایل باینری بدون امضا هشدار می‌دهند؛ این فعلاً طبیعی است. در مک‌اواس، اجرای اولین بار فایل دانلودشده نیاز به `xattr -d com.apple.quarantine kite-macos-arm64` (یا کلیک راست ← Open) دارد تا از Gatekeeper عبور کند، چون تأییدنشده (notarized) است. در ویندوز، آنتی‌ویروس‌ها (از جمله Defender) گاهی درست بعد از این‌که Kite فایل `wintun.dll` را کنار خودش می‌نویسد، آن را قرنطینه می‌کنند، چون یک DLL شبکه‌ای نزدیک به هسته مثل این معمولاً به‌صورت اکتشافی علامت‌گذاری می‌شود — همان مشکلی که WireGuard، v2rayN و دیگر برنامه‌های مبتنی بر Wintun هم دارند. اگر حالت TUN با خطایی شبیه «فایل پیدا نشد» شکست خورد، پوشه‌ی Kite را به لیست استثناهای آنتی‌ویروس اضافه کنید.

## مشارکت

Issue و PR خوش‌آمد است. برای دیدن کارهای واقعاً باقی‌مانده به [محدودیت‌های شناخته‌شده](#محدودیت‌های-شناخته‌شده--قدم‌های-بعدی) در بالا مراجعه کنید.

## مجوز

[MIT](LICENSE)
