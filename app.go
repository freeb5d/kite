package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/freeb5d/kite/internal/system"
	"github.com/freeb5d/kite/internal/tray"
	"github.com/freeb5d/kite/internal/update"
	"github.com/freeb5d/kite/internal/xray"
	"github.com/freeb5d/kite/pkg/probe"
	"github.com/freeb5d/kite/pkg/profile"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails bound struct: every exported method on it becomes
// callable from the frontend via the generated JS bridge.
type App struct {
	ctx          context.Context
	manager      *xray.Manager
	store        *profile.Store
	updateInfo   update.Info
	killSwitchOn bool
	// quitting distinguishes a real quit (from the tray's "Quit Kite")
	// from the window's own close button, which main.go's OnBeforeClose
	// intercepts to hide to the tray instead -- see main.go.
	quitting bool
}

func NewApp() *App {
	return &App{}
}

// startup runs once the Wails runtime is ready and the window exists.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.store = profile.NewStore()
	a.manager = xray.NewManager()

	// Nothing is connected yet, so anything below is left over from a
	// previous run that crashed or was killed while connected -- without
	// this the user has no internet (proxy pointing at a dead port, or all
	// outbound traffic blocked) until they connect and disconnect again.
	update.CleanupOldBinary()
	system.RemoveStaleTUNRoutes("172.19.")
	_ = system.ClearStaleProxy("127.0.0.1", xray.HTTPInboundPort)
	if system.KillSwitchActive() {
		_ = system.DisableKillSwitch()
	}

	go tray.Start(trayIconPNG,
		func() { // Show Kite
			wailsruntime.WindowShow(a.ctx)
			wailsruntime.WindowUnminimise(a.ctx)
		},
		func() { // Disconnect
			_ = a.Disconnect()
		},
		func() { // Quit Kite
			a.quitting = true
			wailsruntime.Quit(a.ctx)
		},
	)
}

// shutdown runs when the app is closing; make sure xray is stopped and the
// system proxy is restored to direct.
func (a *App) shutdown(ctx context.Context) {
	if a.manager != nil {
		a.manager.Stop()
	}
	_ = system.ClearStaleProxy("127.0.0.1", xray.HTTPInboundPort)
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
	}
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
	// importSubscription only drops the group's old servers once the new
	// list has been fetched and parsed, so a failed refresh (offline,
	// provider down) leaves the existing servers untouched.
	return a.importSubscription(subURL, groupID)
}

// DeleteSubscriptionGroup removes every server that was imported from
// the same subscription (same extra.subGroup id).
func (a *App) DeleteSubscriptionGroup(groupID string) error {
	_, err := a.store.ReplaceWhere(inGroup(groupID), nil)
	return err
}

// EditSubscription renames a subscription group and/or changes its URL.
// A changed URL takes effect on the next sync.
func (a *App) EditSubscription(groupID, name, subURL string) error {
	name, subURL = strings.TrimSpace(name), strings.TrimSpace(subURL)
	if name == "" {
		return fmt.Errorf("name can't be empty")
	}
	if u, err := url.Parse(subURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("subscription URL must be an http(s):// link")
	}
	n, err := a.store.UpdateWhere(inGroup(groupID), func(s *profile.Server) {
		s.Extra["subGroupName"] = name
		s.Extra["subURL"] = subURL
	})
	if err == nil && n == 0 {
		err = fmt.Errorf("subscription group not found")
	}
	return err
}

// PingServer measures a server's delay in ms. mode is "tcp", "http" or
// "real" (a real request through a temporary xray-core instance).
func (a *App) PingServer(id, mode string) (int, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return 0, err
	}
	return probe.Ping(server, mode)
}

// ShareLink returns a server's standard share link (vless://, vmess://,
// trojan://, ss://) for copying into another client.
func (a *App) ShareLink(id string) (string, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return "", err
	}
	return profile.ShareLink(server)
}

// ShareSubscription returns the share links of every server in a
// subscription group, one per line -- a plain link list any client can
// import, even when the original subscription URL is private.
func (a *App) ShareSubscription(groupID string) (string, error) {
	all, err := a.store.List()
	if err != nil {
		return "", err
	}
	var links []string
	for _, s := range all {
		if s.Extra["subGroup"] == groupID {
			if link, err := profile.ShareLink(s); err == nil {
				links = append(links, link)
			}
		}
	}
	if len(links) == 0 {
		return "", fmt.Errorf("subscription group not found")
	}
	return strings.Join(links, "\n"), nil
}

func inGroup(groupID string) func(profile.Server) bool {
	return func(s profile.Server) bool { return s.Extra["subGroup"] == groupID }
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
	// Panels like 3x-ui and Marzban name the subscription via
	// Profile-Title, optionally base64-encoded as "base64:...".
	if title := strings.TrimSpace(resp.Header.Get("Profile-Title")); title != "" {
		if enc, ok := strings.CutPrefix(title, "base64:"); ok {
			if dec, err := base64.StdEncoding.DecodeString(enc); err == nil {
				title = string(dec)
			}
		}
		if title = strings.TrimSpace(title); title != "" {
			groupName = title
		}
	}
	// Refresh interval (hours) the provider asks for; the UI auto-syncs
	// subscriptions whose interval has passed.
	updateHours := strings.TrimSpace(resp.Header.Get("Profile-Update-Interval"))
	updatedAt := strconv.FormatInt(time.Now().Unix(), 10)

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

	for i := range parsed {
		if parsed[i].Extra == nil {
			parsed[i].Extra = map[string]string{}
		}
		parsed[i].Extra["subGroup"] = groupID
		parsed[i].Extra["subGroupName"] = groupName
		parsed[i].Extra["subURL"] = subURL
		parsed[i].Extra["subUpdatedAt"] = updatedAt
		if updateHours != "" {
			parsed[i].Extra["subUpdateHours"] = updateHours
		}
		if subMeta != "" {
			parsed[i].Extra["subUsage"] = subMeta
		}
		if subNotes != "" {
			parsed[i].Extra["subNotes"] = subNotes
		}
	}
	return a.store.ReplaceWhere(inGroup(groupID), parsed)
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
// killSwitch, when true, blocks all outbound traffic except Kite's own
// (which is how xray itself reaches the VPN server) and loopback while
// connected -- see internal/system/killswitch_windows.go. It's currently
// Windows-only and needs the same elevation as TUN mode.
func (a *App) Connect(serverID string, mode string, killSwitch bool) error {
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

	if killSwitch {
		if goruntime.GOOS != "windows" {
			return fmt.Errorf("kill switch is currently only supported on Windows")
		}
		if !system.IsElevated() {
			return fmt.Errorf("kill switch needs administrator privileges -- use \"Restart as admin\" and try again")
		}
	}

	if err := a.manager.Start(server, xrayMode); err != nil {
		return err
	}

	if xrayMode == xray.ModeProxy {
		if err := system.SetProxy("127.0.0.1", xray.HTTPInboundPort, xray.SOCKSInboundPort); err != nil {
			_ = a.manager.Stop()
			return err
		}
	}

	if killSwitch {
		if err := system.EnableKillSwitch(); err != nil {
			_ = system.ClearStaleProxy("127.0.0.1", xray.HTTPInboundPort)
			_ = a.manager.Stop()
			return fmt.Errorf("enable kill switch: %w", err)
		}
	}
	a.killSwitchOn = killSwitch
	tray.SetConnected(true)
	return nil
}

func (a *App) Disconnect() error {
	_ = system.ClearStaleProxy("127.0.0.1", xray.HTTPInboundPort)
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
		a.killSwitchOn = false
	}
	tray.SetConnected(false)
	return a.manager.Stop()
}

func (a *App) Status() xray.Status {
	return a.manager.Status()
}

// Traffic returns the current session's cumulative uplink/downlink byte
// counters, for the live traffic display. The frontend polls this and
// diffs successive calls to derive a speed.
func (a *App) Traffic() xray.Traffic {
	return a.manager.Traffic()
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
	// Disconnect (stop xray/TUN, clear the proxy, remove kill switch
	// firewall rules) *before* relaunching, not left for OnShutdown to do
	// after Quit() -- that cleanup can take real time (TUN adapter
	// teardown, two netsh calls for the kill switch), and every bit of it
	// happening after Quit() eats into the freshly-relaunched process's
	// fixed wait-for-old-instance-to-die buffer below.
	if a.manager != nil && a.manager.Status().State == xray.StateRunning {
		_ = a.Disconnect()
	}

	if err := system.RelaunchElevated(); err != nil {
		return err
	}
	// Quit via Wails' own shutdown path (which releases the
	// SingleInstanceLock's IPC listener as part of tearing down), not a
	// raw os.Exit -- an abrupt kill left that listener around long enough
	// that the freshly-elevated process (launched just above) could still
	// see this instance as "already running" and get treated as a second
	// instance instead of starting for real, leaving two windows open
	// with neither one actually elevated.
	a.quitting = true
	wailsruntime.Quit(a.ctx)
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

	// Two requests over one kept-alive connection: the first opens the
	// tunnel (handshakes), the second is the round trip that's reported --
	// the same way "real delay" is measured.
	var body []byte
	var delay time.Duration
	for i := 0; i < 2; i++ {
		start := time.Now()
		resp, err := client.Get("https://www.cloudflare.com/cdn-cgi/trace")
		if err != nil {
			return TestResult{}, fmt.Errorf("request through proxy failed: %w", err)
		}
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return TestResult{}, fmt.Errorf("read response: %w", err)
		}
		delay = time.Since(start)
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

// XrayVersion returns the embedded xray-core version, for the About panel.
func (a *App) XrayVersion() string {
	return xray.CoreVersion()
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
	_ = system.ClearStaleProxy("127.0.0.1", xray.HTTPInboundPort)
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
		a.killSwitchOn = false
	}

	err := update.Apply(a.updateInfo, func(p update.Progress) {
		wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]int64{
			"downloaded": p.Downloaded,
			"total":      p.Total,
		})
	})
	if err != nil {
		return err
	}

	// See RestartElevated's comment: quit via Wails' own shutdown path,
	// not a raw os.Exit, so the SingleInstanceLock's IPC listener is
	// actually released before the already-started new process gets far
	// enough to check it.
	a.quitting = true
	wailsruntime.Quit(a.ctx)
	return nil
}
