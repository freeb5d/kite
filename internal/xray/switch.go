package xray

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/infra/conf"

	"github.com/freeb5d/kite/internal/system"
	"github.com/freeb5d/kite/pkg/conntrack"
	"github.com/freeb5d/kite/pkg/profile"
	"github.com/freeb5d/kite/pkg/xrayconf"
)

// ErrCannotSwitch means the running connection can't swap servers in place;
// the caller should disconnect and connect instead.
var ErrCannotSwitch = errors.New("cannot switch in place")

// Switch moves a running TUN connection to another server without
// recreating the TUN adapter: only xray's "proxy" outbound is replaced.
// (xray-core never releases a Wintun adapter on Close, so a full
// reconnect costs one adapter every time.) SSH servers need a different
// DNS setup and their own bridge, so they always reconnect fully.
func (m *Manager) Switch(server profile.Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State != StateRunning || m.status.Mode != ModeTUN || m.instance == nil ||
		m.sshBridge != nil || server.Protocol == "ssh" {
		return ErrCannotSwitch
	}

	pinned, ips, err := pinServerAddress(server)
	if err != nil {
		return err
	}
	ob, err := xrayconf.Outbound(pinned)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(ob)
	if err != nil {
		return err
	}
	var detour conf.OutboundDetourConfig
	if err := json.Unmarshal(raw, &detour); err != nil {
		return err
	}
	handlerCfg, err := detour.Build()
	if err != nil {
		return err
	}

	// Route the new server's IPs outside the tunnel before using them.
	old := m.exceptionRoutes
	var added []string
	for _, ip := range ips {
		if !contains(old, ip) {
			if err := system.AddExceptionRoute(ip); err != nil {
				for _, a := range added {
					_ = system.RemoveExceptionRoute(a)
				}
				return err
			}
			added = append(added, ip)
		}
	}

	undo := func(err error) error {
		for _, a := range added {
			_ = system.RemoveExceptionRoute(a)
		}
		return err
	}
	obm, ok := m.instance.GetFeature(outbound.ManagerType()).(outbound.Manager)
	if !ok {
		return undo(ErrCannotSwitch)
	}
	if err := obm.RemoveHandler(context.Background(), "proxy"); err != nil {
		return undo(err)
	}
	if err := core.AddOutboundHandler(m.instance, handlerCfg); err != nil {
		// The proxy outbound is gone now; the UI falls back to a full
		// reconnect, which rebuilds everything.
		return undo(err)
	}
	// Connections still open through the old server would keep using it.
	conntrack.CloseAll()

	for _, ip := range old {
		if !contains(ips, ip) {
			_ = system.RemoveExceptionRoute(ip)
		}
	}
	m.exceptionRoutes = ips
	m.status = Status{State: StateRunning, Server: server.Name, Mode: ModeTUN, Since: time.Now().UnixMilli()}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
