package main

import (
	"context"
	"embed"
	"os"
	"slices"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIconPNG []byte

// relaunchWaitFlag is passed to a deliberately-relaunched process
// (elevation restart, self-update) by internal/system.RelaunchElevated
// and internal/update.Apply. Spawning a new process is near-instant,
// but the old one quitting -- and releasing the SingleInstanceLock's IPC
// listener below -- isn't quite as instant, so without a beat to let
// that finish, the new process's own lock registration could run first,
// see the (about to die) old one as "already running", and defer to it
// instead of actually starting: the old process then exits anyway,
// leaving nothing running at all.
const relaunchWaitFlag = "--kite-relaunch-wait"

func main() {
	if slices.Contains(os.Args[1:], relaunchWaitFlag) {
		time.Sleep(3 * time.Second)
	}

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
		// Since closing the window keeps Kite running in the tray instead
		// of quitting, launching the exe again (e.g. double-clicking it,
		// or a shortcut) would otherwise start a second process -- and
		// both processes fighting over the same native resources (WinTun,
		// the system proxy registry keys) is exactly what produced the
		// "operation ... already completed" native error. A second launch
		// now just asks the already-running instance to show itself.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "kite-a15e9f0a-9e3b-4a2e-8c7c-3a6b2b3f6a41",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				wailsruntime.WindowShow(app.ctx)
				wailsruntime.WindowUnminimise(app.ctx)
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
