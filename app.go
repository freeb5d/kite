package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

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

func (a *App) RenameProfile(id string, name string) (profile.Server, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return profile.Server{}, err
	}
	server.Name = name
	if err := a.store.Update(server); err != nil {
		return profile.Server{}, err
	}
	return server, nil
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

// TestConnection makes an actual HTTP request through the local xray HTTP
// inbound so a "running" status that isn't really routing traffic (bad
// outbound handshake, unreachable server, etc.) surfaces a concrete error
// instead of looking like nothing is wrong.
func (a *App) TestConnection() (string, error) {
	if a.manager.Status().State != xray.StateRunning {
		return "", fmt.Errorf("not connected")
	}

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", xray.HTTPInboundPort))
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	const testURL = "https://cp.cloudflare.com/generate_204"
	resp, err := client.Get(testURL)
	if err != nil {
		return "", fmt.Errorf("request through proxy failed: %w", err)
	}
	defer resp.Body.Close()

	return fmt.Sprintf("OK (%s -> HTTP %d)", testURL, resp.StatusCode), nil
}

// RecentLog returns the tail of xray-core's own error log, for diagnosing a
// connection that reports "running" but isn't actually routing traffic.
func (a *App) RecentLog() (string, error) {
	data, err := os.ReadFile(xray.LogFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	const maxBytes = 8000
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data), nil
}

// version is set at build time via -ldflags "-X main.version=v1.2.3"
// (see .github/workflows/release.yml); "dev" otherwise.
var version = "dev"

func (a *App) Version() string {
	return version
}
