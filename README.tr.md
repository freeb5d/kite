<div align="center">
  <img src=".github/logo.png" alt="Kite" width="120" />

  # Kite

  **Çapraz platform bir masaüstü V2Ray/proxy istemcisi.**
  Wails + Go arka uç, React/Tailwind ön uç, Go kütüphaneleri olarak gömülü xray-core/sing-box.

  [![Release](https://img.shields.io/github/v/release/freeb5d/kite?label=release&color=6366f1)](https://github.com/freeb5d/kite/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/freeb5d/kite/release.yml?label=build)](https://github.com/freeb5d/kite/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#i̇ndir)
  [![Go Report Card](https://goreportcard.com/badge/github.com/freeb5d/kite)](https://goreportcard.com/report/github.com/freeb5d/kite)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [İndir](#i̇ndir) · [Özellikler](#özellikler) · [Kaynaktan derleme](#kaynaktan-derleme) · [Mimari](#mimari)

  [English](README.md) · [中文](README.zh.md) · [فارسی](README.fa.md) · Türkçe · [العربية](README.ar.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md)
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Kite, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Kite, dark mode" width="49%" />
</p>

---

## İndir

En son sürümü **[Releases sayfasından](https://github.com/freeb5d/kite/releases/latest)** alın:

| Platform | İndirme |
| --- | --- |
| Windows (x64) | `kite-windows-amd64.exe` |
| Linux (x64) | `kite-linux-amd64` |
| macOS (Apple Silicon) | `kite-macos-arm64` |

Her platform tek bir taşınabilir yürütülebilir dosya olarak sunulur — kurulum programı yok, yönetici izni gerekmiyor (macOS derlemesi, Wails'in ürettiği `.app` paketinden çıkarılmış ham ikili dosyadır; bunun getirdiği Gatekeeper uyarısı ve nasıl aşılacağı için [Bilinen eksikler](#bilinen-eksikler--sıradaki-adımlar) bölümüne bakın). Kurulduktan sonra başlangıçta yeni sürümleri kontrol eder ve tek tıkla kendini güncelleyebilir.

## Özellikler

- **Bağlantı biçimleri**: `vmess://`, `vless://`, `trojan://`, `ss://` — bir paylaşım bağlantısı yapıştırın, ayrıştırılıp kaydedilsin
- **Abonelik URL'leri** — bir `http(s)://` abonelik bağlantısı yapıştırın (V2RayN/V2RayNG/Shadowrocket'ın kullandığı base64 bağlantı listesi biçimi) ve içerdiği tüm sunucular tek seferde içe aktarılır, listede tek bir daraltılabilir grupta toplanır (bir abonelik yüzlerce sunucu içerebilir). Sağlayıcı bunu bildirdiğinde (`Subscription-Userinfo` başlığı üzerinden veya bazı sağlayıcıların bağlantı listesine karıştırdığı sahte "info" girdileri üzerinden) grup, plan/trafik/son kullanma bilgisini gösterir ve sunucularını yeniden çekip yenilemek için kendi senkronizasyon düğmesine sahiptir
- **İki seçilebilir motor**, ikisi de Go kütüphanesi olarak gömülü (dışarıya çağrılan ikili dosyalar değil) — tam yaşam döngüsü kontrolü, istatistikler için stdout ayrıştırmaya gerek yok. **xray-core** varsayılandır (daha geniş taşıma desteği, özellikle `headerType=http` gizlemeli `tcp`); **sing-box** ise TUN modu olan tek motordur. Hakkında panelinden istediğiniz zaman değiştirin (önce bağlantıyı kesin)
- **Taşıma yöntemleri**: TCP (xray-core'un `headerType=http` gizlemesi dahil), WebSocket ve gRPC; TLS/REALITY güvenliği doğrudan bağlantıdan algılanır
- **Sistem proxy entegrasyonu** — Bağlan/Bağlantıyı Kes, işletim sistemi HTTP proxy'sini otomatik olarak değiştirir (Windows'ta kullanıcı bazlı kayıt defteri, yükseltme gerekmez)
- **TUN modu (Windows, yalnızca sing-box)** — sadece proxy ayarına uyan uygulamalar yerine, tüm sistem trafiğini sanal bir ağ adaptörü (dahili WinTun) üzerinden yönlendirir. Yönetici izni ve sing-box motorunun seçili olmasını gerektirir; Kite kendini tek tıkla yükseltilmiş olarak yeniden başlatabilir
- **Kill switch (Windows)** — bağlıyken, bir Windows Güvenlik Duvarı kural çifti aracılığıyla Kite'ın kendisi dışındaki tüm giden trafiği engeller; böylece sistem proxy'sini yok sayan bir uygulama (veya çöken bir motor süreci) trafiği tünelin dışına sızdıramaz. TUN modu ile aynı yönetici gereksinimi
- **Sistem tepsisi** — pencereyi kapatmak, çıkmak yerine onu tepsiye gizler, böylece etkin bir bağlantı çalışmaya devam eder; tepsi menüsünde Kite'ı Göster, Bağlantıyı Kes ve Kite'tan Çık bulunur
- **Yerleşik tanılama** — Test düğmesi tünel üzerinden gerçek bir istek yapar ve gerçek sonucu bildirir; bir günlük görüntüleyici etkin motorun kendi hata ayıklama günlüğünü satır içinde gösterir
- **Kendini güncelleme** — başlangıçta GitHub Releases'i kontrol eder, tek tıkla indirir, değiştirir ve yeniden başlatır (Hakkında paneli Kite'ın sürümünü, etkin motoru ve onun sürümünü gösterir)
- **Koyu / açık temalar**, aranabilir sunucu listesi, satır içi yeniden adlandırma ve tek tıkla kaldırma ile birlikte
- **8 dil**, kenar çubuğundan değiştirilebilir (Farsça dahili Vazirmatn yazı tipini kullanır):
  - 🇬🇧 English (varsayılan)
  - 🇨🇳 中文 (Çince)
  - 🇮🇷 فارسی (Farsça)
  - 🇹🇷 Türkçe
  - 🇸🇦 العربية (Arapça)
  - 🇫🇷 Français (Fransızca)
  - 🇩🇪 Deutsch (Almanca)
  - 🇷🇺 Русский (Rusça)

## Kaynaktan derleme

### Tek seferlik araç zinciri kurulumu

1. **Go 1.27+** — https://go.dev/dl/ . Windows'ta ayrıca CGO için bir C derleyicisi gerekir (Wails ve bazı bağımlılıkları bunu kullanır) — [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) veya `winget install -e --id GoLang.Go` ile birlikte MSYS2'nin `mingw-w64-x86_64-gcc` paketini kurun.
2. **Node.js (LTS)** — https://nodejs.org , veya `winget install OpenJS.NodeJS.LTS`.
3. **Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   wails doctor
   ```

### Çalıştırma

```bash
go mod tidy
cd frontend && npm install && cd ..
wails dev
```

### Bir sürüm ikili dosyası derleme

```bash
wails build -tags webkit2_41
```

## Mimari

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

`App` üzerindeki her bağlı Go metodu ([app.go](app.go) içinde), `wails dev`/`wails build` `frontend/wailsjs/go/main/App.js` dosyasını ürettiğinde ön uçtan çağrılabilir bir JS fonksiyonuna dönüşür.

## Bilinen eksikler / sıradaki adımlar

- **TUN modu yalnızca sing-box motoruyla çalışır** — xray-core (varsayılan) burada TUN girişine sahip değildir; xray-core seçiliyken TUN moduna geçmeye çalışmak, önce Hakkında panelinden motoru değiştirmenizi söyleyen açık bir hatayla başarısız olur.
- **Her iki motorda da canlı trafik istatistikleri sahte (stub)** — "Daha fazla göster" panelindeki rakamlar her zaman 0B/s gösterir. xray-core'un stats.Manager'ı ve sing-box'ın (`experimental.clash_api`/`v2ray_api` üzerinden) `trafficcontrol.Manager`'ı, henüz yapılmamış ekstra bağlantılara ihtiyaç duyuyor — her iki motor için de bir sonraki adım.
- **macOS yalnızca Apple Silicon (arm64)** — Intel derlemesi yok. Gerekirse `.github/workflows/release.yml` içine `darwin/arm64` ile birlikte bir `darwin/amd64` matris girdisi ekleyin.
- **TUN modu şimdilik yalnızca Windows'ta** ve yalnızca IPv4 için çalışır — motorun kendi üst akış bağlantısının beslediği TUN adaptörü üzerinden döngüye girmesini engelleyen istisna yolu (`internal/system/route_windows.go`) yalnızca çözümlenmiş IPv4 adreslerini kapsar; yalnızca IPv6 üzerinden erişilebilen bir sunucu TUN modunda henüz çalışmaz. Linux/macOS TUN desteği mümkündür (sing-box'ın kendi `tun` paketi zaten ikisini de destekler) ama burada bağlanmamıştır.
- **Linux sistem proxy'si yalnızca GNOME'u kapsar** (`gsettings`) — diğer masaüstü ortamlarının `internal/system/proxy_linux.go` içinde kendi arka uçlarına ihtiyacı vardır.
- **Kill switch yalnızca Windows'ta** — `netsh advfirewall` kuralları aracılığıyla uygulanır (`internal/system/killswitch_windows.go`); Linux/macOS kendi arka uçlarına (`iptables`/`pfctl`) ihtiyaç duyar ve şimdilik uygulanmamıştır (bu platformlarda anahtar gizlidir, TUN modu gibi).
- **Yerel simge durumuna küçültme düğmesi hâlâ yalnızca görev çubuğuna küçültüyor** — Wails v2, işletim sistemi düzeyinde bir simge durumuna küçültme olayı için kanca sunmuyor, yalnızca pencere kapatma (`OnBeforeClose`, tepsi özelliğinin kullandığı) sunuyor. Yalnızca pencereyi kapatmak onu tepsiye gizler.
- **Henüz otomatik test yok.**
- **Kod imzalama yok** — Windows SmartScreen ve macOS Gatekeeper, imzasız bir ikili dosya için uyarı verir; bu şimdilik beklenen bir durumdur. macOS'ta, indirilen ikili dosyayı ilk çalıştırmak, noter onaylı olmadığı için Gatekeeper'ı aşmak amacıyla `xattr -d com.apple.quarantine kite-macos-arm64` (veya sağ tık → Aç) gerektirir. Windows'ta, antivirüs yazılımları (Defender dahil) bazen Kite, `wintun.dll` dosyasını kendi yanına yazdıktan hemen sonra onu karantinaya alır, çünkü bunun gibi çekirdeğe yakın bir ağ DLL'i genellikle sezgisel olarak işaretlenir — WireGuard, v2rayN ve diğer Wintun tabanlı uygulamaların da karşılaştığı aynı sorun. TUN modu "dosya bulunamadı" tarzı bir hatayla başarısız olursa, Kite'ın klasörünü antivirüsünüzün istisnalarına ekleyin.

## Katkıda bulunma

Issue ve PR'lar memnuniyetle karşılanır. Gerçekten yapılması gerekenler için yukarıdaki [Bilinen eksikler](#bilinen-eksikler--sıradaki-adımlar) bölümüne bakın.

## Lisans

[MIT](LICENSE)
