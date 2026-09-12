<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **عميل سطح مكتب متعدد المنصات لبروتوكول V2Ray / بروكسي.**
  خلفية Wails + Go، واجهة أمامية React/Tailwind، مع تضمين xray-core/sing-box كمكتبات Go.

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#تنزيل)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [تنزيل](#تنزيل) · [الميزات](#الميزات) · [البناء من المصدر](#البناء-من-المصدر) · [البنية](#البنية)

  [English](README.md) · [中文](README.zh.md) · [فارسی](README.fa.md) · [Türkçe](README.tr.md) · العربية · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md)
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## تنزيل

احصل على أحدث إصدار من **[صفحة الإصدارات](https://github.com/freeb5d/kite/releases/latest)**:

| المنصة | التنزيل |
| --- | --- |
| ويندوز (x64) | `kite-windows-amd64.exe` |
| لينكس (x64) | `kite-linux-amd64` |
| ماك أو إس (Apple Silicon) | `kite-macos-arm64` |

يُشحن كل نظام أساسي كملف تنفيذي محمول واحد — بدون مثبّت، وبدون الحاجة لصلاحيات المسؤول (نسخة macOS هي الملف الثنائي الخام المستخرج من حزمة `.app` التي ينتجها Wails؛ راجع [الفجوات المعروفة](#الفجوات-المعروفة--الخطوات-التالية) لمعرفة تنبيه Gatekeeper الناتج عن ذلك وكيفية تجاوزه). بعد التثبيت، يتحقق البرنامج من وجود إصدارات جديدة عند بدء التشغيل ويمكنه تحديث نفسه بنقرة واحدة.

## الميزات

- **صيغ الروابط**: `vmess://`، `vless://`، `trojan://`، `ss://` — الصق رابط مشاركة ليتم تحليله وحفظه
- **روابط الاشتراك** — الصق رابط اشتراك `http(s)://` (صيغة قائمة الروابط بترميز base64 التي تستخدمها V2RayN/V2RayNG/Shadowrocket) ليتم استيراد جميع الخوادم التي يحتويها دفعة واحدة، مجمّعة في مجموعة واحدة قابلة للطي في القائمة (يمكن أن يحتوي الاشتراك على مئات الخوادم). تعرض المجموعة معلومات الخطة/حجم البيانات/تاريخ الانتهاء عندما يبلغ عنها المزوّد (عبر ترويسة `Subscription-Userinfo`، أو إدخالات "info" الوهمية التي يضيفها بعض المزوّدين ضمن قائمة الروابط)، ولديها زر مزامنة خاص بها لإعادة الجلب وتحديث خوادمها
- **محركان قابلان للاختيار**، كلاهما مُضمَّن كمكتبة Go (وليس ملفًا ثنائيًا منفصلًا) — تحكّم كامل بدورة الحياة، دون الحاجة لتحليل المخرجات القياسية للحصول على الإحصاءات. **xray-core** هو الافتراضي (دعم أوسع لطرق النقل، خصوصًا `tcp` مع تمويه `headerType=http`)؛ و**sing-box** هو المحرك الوحيد الذي يدعم وضع TUN. يمكن التبديل بينهما في أي وقت من لوحة "حول" (بعد قطع الاتصال أولًا)
- **طرق النقل**: TCP (بما في ذلك تمويه `headerType=http` الخاص بـ xray-core)، وWebSocket، وgRPC، مع اكتشاف أمان TLS/REALITY مباشرة من الرابط
- **تكامل بروكسي النظام** — يقوم الاتصال/قطع الاتصال بتبديل بروكسي HTTP الخاص بنظام التشغيل تلقائيًا (سجل خاص بكل مستخدم على ويندوز، دون الحاجة لصلاحيات مرتفعة)
- **وضع TUN (ويندوز فقط، sing-box فقط)** — يوجّه كل حركة مرور النظام عبر محول شبكة افتراضي (WinTun، مضمّن)، بدلًا من التطبيقات التي تحترم إعداد البروكسي فقط. يتطلب صلاحيات المسؤول واختيار محرك sing-box؛ يمكن لـ Kite إعادة تشغيل نفسه بصلاحيات مرتفعة بنقرة واحدة
- **مفتاح الإيقاف (Kill switch) (ويندوز فقط)** — يحظر كل حركة المرور الصادرة باستثناء حركة مرور Kite نفسها أثناء الاتصال، عبر زوج من قواعد جدار حماية ويندوز، بحيث لا يستطيع تطبيق يتجاهل بروكسي النظام (أو عملية محرك متعطلة) تسريب حركة المرور خارج النفق. نفس متطلبات صلاحيات المسؤول كوضع TUN
- **أيقونة النظام** — إغلاق النافذة يخفيها إلى شريط النظام بدلًا من الخروج، بحيث يستمر أي اتصال نشط بالعمل؛ تحتوي قائمة الأيقونة على إظهار Kite وقطع الاتصال والخروج من Kite
- **تشخيص مدمج** — زر الاختبار يقوم بطلب حقيقي عبر النفق ويبلغ عن النتيجة الفعلية؛ عارض السجلات يعرض سجل التصحيح الخاص بالمحرك النشط مباشرة داخل التطبيق
- **تحديث ذاتي** — يتحقق من GitHub Releases عند الإقلاع، وبنقرة واحدة يقوم بالتنزيل والاستبدال وإعادة التشغيل (تعرض لوحة "حول" إصدار Kite والمحرك النشط وإصداره)
- **سمات داكنة / فاتحة**، مع قائمة خوادم قابلة للبحث، وإعادة تسمية مباشرة، وإزالة بنقرة واحدة
- **8 لغات**، يمكن تبديلها من الشريط الجانبي (الفارسية تستخدم خط Vazirmatn المضمّن):
  - 🇬🇧 English (الافتراضية)
  - 🇨🇳 中文 (الصينية)
  - 🇮🇷 فارسی (الفارسية)
  - 🇹🇷 Türkçe (التركية)
  - 🇸🇦 العربية
  - 🇫🇷 Français (الفرنسية)
  - 🇩🇪 Deutsch (الألمانية)
  - 🇷🇺 Русский (الروسية)

## البناء من المصدر

### إعداد سلسلة الأدوات لمرة واحدة

1. **Go 1.27 أو أحدث** — https://go.dev/dl/ . يحتاج ويندوز أيضًا إلى مترجم C لدعم CGO (يستخدمه Wails وبعض تبعياته) — ثبّت [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) أو `winget install -e --id GoLang.Go` إلى جانب حزمة `mingw-w64-x86_64-gcc` من MSYS2.
2. **Node.js (النسخة طويلة الأمد LTS)** — https://nodejs.org ، أو `winget install OpenJS.NodeJS.LTS`.
3. **واجهة سطر أوامر Wails**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### التشغيل

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### بناء ملف ثنائي للإصدار

```bash
wails build -tags webkit2_41
```

## البنية

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

تتحول كل دالة Go مرتبطة على `App` (في [app.go](app.go)) إلى دالة JS قابلة للاستدعاء من الواجهة الأمامية بمجرد أن يُنشئ `wails dev`/`wails build` ملف `frontend/wailsjs/go/main/App.js`.

## الفجوات المعروفة / الخطوات التالية

- **وضع TUN يعمل فقط مع محرك sing-box** — لا يملك xray-core (الافتراضي) مدخل TUN هنا؛ التبديل إلى وضع TUN أثناء اختيار xray-core يفشل مع رسالة خطأ واضحة تطلب منك تبديل المحرك أولًا من لوحة "حول".
- **إحصاءات حركة المرور المباشرة عبارة عن نموذج وهمي في كلا المحركين** — أرقام لوحة "عرض المزيد" تبقى دائمًا 0B/s. يحتاج كل من stats.Manager الخاص بـ xray-core و`trafficcontrol.Manager` الخاص بـ sing-box (عبر `experimental.clash_api`/`v2ray_api`) إلى ربط إضافي لم يُنجز بعد — خطوة تالية لكلا المحركين.
- **macOS يدعم فقط Apple Silicon (arm64)** — لا يوجد بناء لمعالجات Intel. إذا لزم الأمر، أضف إدخال مصفوفة `darwin/amd64` بجانب `darwin/arm64` في `.github/workflows/release.yml`.
- **وضع TUN حاليًا لويندوز فقط** ويدعم IPv4 فقط في الوقت الحالي — مسار الاستثناء الذي يمنع اتصال المحرك الصاعد الخاص به من الدوران عبر محول TUN (`internal/system/route_windows.go`) يغطي فقط عناوين IPv4 التي تم حلها؛ خادم لا يمكن الوصول إليه إلا عبر IPv6 لن يعمل في وضع TUN بعد. دعم TUN على Linux/macOS ممكن (حزمة `tun` الخاصة بـ sing-box تدعم كليهما بالفعل) لكنه غير مفعّل هنا بعد.
- **بروكسي نظام Linux يغطي GNOME فقط** (`gsettings`) — بيئات سطح المكتب الأخرى تحتاج إلى تطبيق خلفية خاصة بها في `internal/system/proxy_linux.go`.
- **مفتاح الإيقاف لويندوز فقط** — تم تنفيذه عبر قواعد `netsh advfirewall` (`internal/system/killswitch_windows.go`)؛ تحتاج Linux/macOS إلى تطبيق خلفية خاصة بها (`iptables`/`pfctl`) وهي غير مُنفَّذة حاليًا (يُخفى المفتاح في هذه الأنظمة، مثل وضع TUN).
- **زر التصغير الأصلي لا يزال يقوم فقط بالتصغير إلى شريط المهام** — لا يوفر Wails v2 خطافًا لحدث التصغير على مستوى نظام التشغيل، فقط إغلاق النافذة (`OnBeforeClose`، الذي تستخدمه ميزة أيقونة النظام). إغلاق النافذة فقط هو ما يخفيها إلى شريط النظام.
- **لا توجد اختبارات آلية بعد.**
- **لا يوجد توقيع للكود** — سيحذّر كل من Windows SmartScreen وmacOS Gatekeeper من ملف ثنائي غير موقّع؛ وهذا متوقع حاليًا. على macOS، يحتاج تشغيل الملف الثنائي الذي تم تنزيله لأول مرة إلى `xattr -d com.apple.quarantine kite-macos-arm64` (أو النقر بزر الماوس الأيمن ← فتح) لتجاوز Gatekeeper، لأنه غير موثّق رسميًا (notarized). على ويندوز، تقوم برامج مكافحة الفيروسات (بما فيها Defender) أحيانًا بحجر `wintun.dll` مباشرة بعد أن يكتبه Kite بجانب نفسه، لأن مكتبة DLL شبكية قريبة من النواة كهذه غالبًا ما تُصنَّف بشكل استكشافي على أنها مشبوهة — وهي نفس المشكلة التي تواجهها WireGuard وv2rayN وتطبيقات أخرى قائمة على Wintun. إذا فشل وضع TUN مع خطأ يشبه "الملف غير موجود"، أضف مجلد Kite إلى استثناءات برنامج مكافحة الفيروسات لديك.

## المساهمة

الـ Issues والـ PRs مرحّب بها. راجع [الفجوات المعروفة](#الفجوات-المعروفة--الخطوات-التالية) أعلاه لمعرفة ما تبقى فعليًا.

## الرخصة

[MIT](LICENSE)
