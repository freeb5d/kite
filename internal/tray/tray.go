// Package tray puts a Kite icon in the system tray so the app keeps
// running (and the VPN stays connected) when its window is closed,
// instead of quitting -- with a menu to reopen the window, disconnect,
// or actually quit.
package tray

import (
	"bytes"
	"encoding/binary"
	"image/png"
	goruntime "runtime"

	"github.com/energye/systray"
)

var (
	disconnectItem *systray.MenuItem
	ready          = make(chan struct{})
)

// Start blocks running the tray's native event loop until Quit is
// called, so call it in its own goroutine. iconPNG is used as-is on
// macOS/Linux; on Windows it's wrapped in a minimal ICO container first,
// since Windows tray icons need that format, not raw PNG.
func Start(iconPNG []byte, onShow, onDisconnect, onQuit func()) {
	systray.Run(func() {
		icon := iconPNG
		if goruntime.GOOS == "windows" {
			icon = pngToICO(iconPNG)
		}
		systray.SetIcon(icon)
		systray.SetTitle("Kite")
		systray.SetTooltip("Kite")
		// Left-click the tray icon to restore the window, in addition to
		// the "Show Kite" menu item (right-click still opens the menu).
		systray.SetOnClick(func(menu systray.IMenu) { onShow() })

		mShow := systray.AddMenuItem("Show Kite", "Show the Kite window")
		mShow.Click(onShow)

		systray.AddSeparator()
		disconnectItem = systray.AddMenuItem("Disconnect", "Disconnect the VPN")
		disconnectItem.Hide()
		disconnectItem.Click(onDisconnect)

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit Kite", "Quit Kite")
		mQuit.Click(func() {
			onQuit()
			systray.Quit()
		})

		close(ready)
	}, func() {})
}

// SetConnected shows or hides the tray's Disconnect item to match the
// real connection state.
func SetConnected(connected bool) {
	select {
	case <-ready:
	default:
		return // tray isn't set up yet
	}
	if disconnectItem == nil {
		return
	}
	if connected {
		disconnectItem.Show()
	} else {
		disconnectItem.Hide()
	}
}

// pngToICO wraps a single PNG image in a minimal single-entry ICO
// container. Windows Vista+ supports PNG-compressed icon entries
// directly, so this avoids needing a real BMP-based ICO encoder or a
// separately-maintained .ico asset file.
func pngToICO(pngData []byte) []byte {
	width, height := 0, 0
	if cfg, err := png.DecodeConfig(bytes.NewReader(pngData)); err == nil {
		width, height = cfg.Width, cfg.Height
	}
	// The ICO width/height byte is 0 to mean "256"; anything else is
	// stored as its literal byte value 1-255.
	entryWidth, entryHeight := byte(width), byte(height)
	if width <= 0 || width >= 256 {
		entryWidth = 0
	}
	if height <= 0 || height >= 256 {
		entryHeight = 0
	}

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: icon
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // image count

	buf.WriteByte(entryWidth)
	buf.WriteByte(entryHeight)
	buf.WriteByte(0) // color palette count (0 = no palette)
	buf.WriteByte(0) // reserved
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))              // color planes
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32))             // bits per pixel
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pngData)))   // image data size
	_ = binary.Write(&buf, binary.LittleEndian, uint32(6+16))           // offset: header + one entry

	buf.Write(pngData)
	return buf.Bytes()
}
