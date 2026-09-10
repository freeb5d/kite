package main

import (
	"context"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/freeb5d/kite/internal/system"
	"github.com/freeb5d/kite/internal/xray"
)

// App is the Wails bound struct: every exported method on it becomes
// callable from the frontend via the generated JS bridge.
type App struct {
	ctx     context.Context
	manager *xray.Manager
	store   *profile.Store
}

func NewApp() *App {
	return &App{}
}

// startup runs once the Wails runtime is ready and the window exists.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.store = profile.NewStore()
	a.manager = xray.NewManager()
}

// shutdown runs when the app is closing; make sure xray is stopped and the
// system proxy is restored to direct.
func (a *App) shutdown(ctx context.Context) {
	if a.manager != nil {
		a.manager.Stop()
	}
	_ = system.ClearProxy()
}

// --- Server profile methods (bound to frontend) ---

func (a *App) ListProfiles() ([]profile.Server, error) {
	return a.store.List()
}

func (a *App) AddProfileFromLink(link string) (profile.Server, error) {
	server, err := profile.ParseLink(link)
	if err != nil {
		return profile.Server{}, err
	}
	if err := a.store.Add(server); err != nil {
		return profile.Server{}, err
	}
	return server, nil
}

func (a *App) DeleteProfile(id string) error {
	return a.store.Delete(id)
}

// --- Proxy control methods (bound to frontend) ---

func (a *App) Connect(serverID string) error {
	server, err := a.store.Get(serverID)
	if err != nil {
		return err
	}
	if err := a.manager.Start(server); err != nil {
		return err
	}
	if err := system.SetProxy("127.0.0.1", xray.HTTPInboundPort); err != nil {
		_ = a.manager.Stop()
		return err
	}
	return nil
}

func (a *App) Disconnect() error {
	_ = system.ClearProxy()
	return a.manager.Stop()
}

func (a *App) Status() xray.Status {
	return a.manager.Status()
}
