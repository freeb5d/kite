package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/freeb5d/kite/internal/system"
	"github.com/freeb5d/kite/internal/update"
	"github.com/freeb5d/kite/internal/xray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails bound struct: every exported method on it becomes
// callable from the frontend via the generated JS bridge.
type App struct {
	ctx        context.Context
	manager    *xray.Manager
	store      *profile.Store
	updateInfo update.Info
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
	return a.store.Add(server)
}

// AddSubscription fetches a subscription URL (the standard V2RayN/
// V2RayNG/Shadowrocket-style base64 link list most providers publish)
// and adds every server it can parse out of it. It succeeds as long as
// at least one server was added, even if some entries in the
// subscription couldn't be parsed.
func (a *App) AddSubscription(subURL string) ([]profile.Server, error) {
	return a.importSubscription(subURL, uuid.NewString())
}

// RefreshSubscription re-fetches the subscription that a given server
// group was originally imported from, replacing that group's servers
// with the freshly parsed list (same group ID, so the UI's expand/
// collapse state and position don't reset).
func (a *App) RefreshSubscription(groupID string) ([]profile.Server, error) {
	all, err := a.store.List()
	if err != nil {
		return nil, err
	}
	var subURL string
	for _, s := range all {
		if s.Extra["subGroup"] == groupID {
			subURL = s.Extra["subURL"]
			break
		}
	}
	if subURL == "" {
		return nil, fmt.Errorf("subscription group not found")
	}

	if err := a.DeleteSubscriptionGroup(groupID); err != nil {
		return nil, err
	}
	return a.importSubscription(subURL, groupID)
}

// DeleteSubscriptionGroup removes every server that was imported from
// the same subscription (same extra.subGroup id).
func (a *App) DeleteSubscriptionGroup(groupID string) error {
	all, err := a.store.List()
	if err != nil {
		return err
	}
	for _, s := range all {
		if s.Extra["subGroup"] == groupID {
			if err := a.store.Delete(s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) importSubscription(subURL, groupID string) ([]profile.Server, error) {
	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}
	// Some subscription providers gate on a client-looking User-Agent.
	req.Header.Set("User-Agent", "Kite/1.0 (compatible; v2rayN/6.0)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching subscription: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("reading subscription: %w", err)
	}

	parsed, notes, errs := profile.ParseSubscription(string(body))
	if len(parsed) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("no valid servers found in subscription: %w", errs[0])
		}
		return nil, fmt.Errorf("no valid servers found in subscription")
	}

	groupName := subURL
	if u, err := url.Parse(subURL); err == nil && u.Host != "" {
		groupName = u.Host
	}

	// Usage/expiry, when the provider reports it via the informal but
	// widely-adopted Subscription-Userinfo response header, plus any
	// human-readable "info" lines mixed into the link list itself
	// (see profile.isInfoNode) -- both get attached to every server in
	// the group so the UI can show plan/expiry/traffic without a
	// separate subscriptions store.
	var subMeta string
	if usage, ok := profile.ParseSubscriptionUserinfo(resp.Header.Get("Subscription-Userinfo")); ok {
		if b, err := json.Marshal(usage); err == nil {
			subMeta = string(b)
		}
	}
	var subNotes string
	if len(notes) > 0 {
		if b, err := json.Marshal(notes); err == nil {
			subNotes = string(b)
		}
	}

	added := make([]profile.Server, 0, len(parsed))
	for _, server := range parsed {
		if server.Extra == nil {
			server.Extra = map[string]string{}
		}
		server.Extra["subGroup"] = groupID
		server.Extra["subGroupName"] = groupName
		server.Extra["subURL"] = subURL
		if subMeta != "" {
			server.Extra["subUsage"] = subMeta
		}
		if subNotes != "" {
			server.Extra["subNotes"] = subNotes
		}
		stored, err := a.store.Add(server)
		if err != nil {
			continue
		}
		added = append(added, stored)
	}
	if len(added) == 0 {
		return nil, fmt.Errorf("found %d server(s) but failed to save any", len(parsed))
	}
	return added, nil
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

// Connect starts xray-core against the given server profile in the given
// mode ("proxy" or "tun"; anything else is treated as "proxy"). TUN mode
// is currently Windows-only and requires the process to already be
// running elevated -- see Platform/IsElevated/RestartElevated below.
func (a *App) Connect(serverID string, mode string) error {
	server, err := a.store.Get(serverID)
	if err != nil {
		return err
	}

	xrayMode := xray.ModeProxy
	if mode == string(xray.ModeTUN) {
		if goruntime.GOOS != "windows" {
			return fmt.Errorf("TUN mode is currently only supported on Windows")
		}
		if !system.IsElevated() {
			return fmt.Errorf("TUN mode needs administrator privileges -- use \"Restart as admin\" and try again")
		}
		xrayMode = xray.ModeTUN
	}

	if err := a.manager.Start(server, xrayMode); err != nil {
		return err
	}

	if xrayMode == xray.ModeProxy {
		if err := system.SetProxy("127.0.0.1", xray.HTTPInboundPort); err != nil {
			_ = a.manager.Stop()
			return err
		}
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

// Platform reports the OS Kite is running on, so the frontend can hide
// the TUN mode option where it isn't supported yet.
func (a *App) Platform() string {
	return goruntime.GOOS
}

// IsElevated reports whether Kite is currently running with administrator
// privileges (relevant to TUN mode, which needs them).
func (a *App) IsElevated() bool {
	return system.IsElevated()
}

// RestartElevated relaunches Kite with a UAC prompt and exits this
// process on success. Only returns when the relaunch itself failed
// (including the user cancelling the prompt).
func (a *App) RestartElevated() error {
	if err := system.RelaunchElevated(); err != nil {
		return err
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}

// TestResult is what the Test button reports: the exit IP the target site
// actually sees (i.e. the VPN server's IP, not Kite's), the two-letter
// country code Cloudflare's edge resolved it to, and the real round-trip
// time of the whole request (DNS/TCP/TLS/HTTP) made through the tunnel.
type TestResult struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	DelayMs int64  `json:"delayMs"`
}

// TestConnection makes an actual HTTP request through the local xray HTTP
// inbound so a "running" status that isn't really routing traffic (bad
// outbound handshake, unreachable server, etc.) surfaces a concrete error
// instead of looking like nothing is wrong. It hits Cloudflare's own
// trace endpoint rather than a generic 204 check, since that response
// includes the client IP and country as Cloudflare's edge sees them --
// i.e. exactly what a site would see you connecting from.
func (a *App) TestConnection() (TestResult, error) {
	if a.manager.Status().State != xray.StateRunning {
		return TestResult{}, fmt.Errorf("not connected")
	}

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", xray.HTTPInboundPort))
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	start := time.Now()
	resp, err := client.Get("https://www.cloudflare.com/cdn-cgi/trace")
	if err != nil {
		return TestResult{}, fmt.Errorf("request through proxy failed: %w", err)
	}
	defer resp.Body.Close()
	delay := time.Since(start)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestResult{}, fmt.Errorf("read response: %w", err)
	}

	result := TestResult{DelayMs: delay.Milliseconds()}
	for _, line := range strings.Split(string(body), "\n") {
		if ip, ok := strings.CutPrefix(line, "ip="); ok {
			result.IP = strings.TrimSpace(ip)
		}
		if loc, ok := strings.CutPrefix(line, "loc="); ok {
			result.Country = strings.TrimSpace(loc)
		}
	}
	if result.IP == "" {
		return TestResult{}, fmt.Errorf("unexpected response from connectivity check")
	}

	return result, nil
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

// --- Self-update methods (bound to frontend) ---

// CheckForUpdate queries GitHub Releases for a newer build. The result is
// cached on the App so a subsequent ApplyUpdate doesn't need the frontend
// to round-trip the (unexported) download URL back to us.
func (a *App) CheckForUpdate() (update.Info, error) {
	info, err := update.Check(version)
	if err != nil {
		return update.Info{}, err
	}
	a.updateInfo = info
	return info, nil
}

// ApplyUpdate downloads and installs the build found by the most recent
// CheckForUpdate, then relaunches. On success this does not return: the
// new process has already started and this one is about to exit. It only
// returns when something failed before that point.
func (a *App) ApplyUpdate() error {
	if !a.updateInfo.Available {
		return fmt.Errorf("no update available; call CheckForUpdate first")
	}

	if a.manager != nil {
		_ = a.manager.Stop()
	}
	_ = system.ClearProxy()

	err := update.Apply(a.updateInfo, func(p update.Progress) {
		wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]int64{
			"downloaded": p.Downloaded,
			"total":      p.Total,
		})
	})
	if err != nil {
		return err
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}
