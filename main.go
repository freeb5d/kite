package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIconPNG []byte

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Kite",
		Width:  1024,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 17, B: 17, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		// Wails disables the browser's native right-click context menu in
		// production builds by default, which meant there was no way to
		// Paste into a text field with the mouse.
		EnableDefaultContextMenu: true,
		// Closing the window hides it to the system tray instead of
		// quitting, so the VPN connection (and kill switch, if on) keeps
		// running -- see app.go's startup/OnBeforeClose wiring and
		// internal/tray. Actually quitting happens via the tray menu's
		// "Quit Kite".
		OnBeforeClose: func(ctx context.Context) bool {
			if app.quitting {
				return false // allow the real close/quit to proceed
			}
			wailsruntime.WindowHide(ctx)
			return true // prevent close, window is just hidden
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
