// Package xray wraps xray-core as an in-process Go library: it builds a
// config from a server profile, and starts/stops/restarts an xray instance
// without shelling out to an external binary.
package xray

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/freeb5d/kite/internal/system"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/stats"

	// Registers every protocol/transport xray-core ships (vmess, vless,
	// trojan, shadowsocks, http/socks/tun inbounds, ws/tls, ...) with the
	// config loaders used by BuildConfig. Without this blank import,
	// core.New fails with "unknown protocol" for everything.
	_ "github.com/xtls/xray-core/main/distro/all"
)

type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateError    State = "error"
)

type Status struct {
	State   State  `json:"state"`
	Server  string `json:"server,omitempty"`
	Mode    Mode   `json:"mode,omitempty"`
	Message string `json:"message,omitempty"`
}

// Manager owns the lifecycle of a single running xray-core instance.
type Manager struct {
	mu              sync.Mutex
	status          Status
	instance        *core.Instance
	exceptionRoutes []string
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter
}

func NewManager() *Manager {
	return &Manager{status: Status{State: StateStopped}}
}

func (m *Manager) Start(server profile.Server, mode Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State == StateRunning {
		return errors.New("xray is already running; call Stop or Restart first")
	}

	var addedRoutes []string
	if mode == ModeTUN {
		if err := prepareTUN(); err != nil {
			m.status = Status{State: StateError, Message: fmt.Sprintf("prepare TUN: %v", err)}
			return err
		}

		ips, err := net.LookupHost(server.Address)
		if err != nil {
			m.status = Status{State: StateError, Message: fmt.Sprintf("resolve %s: %v", server.Address, err)}
			return err
		}
		for _, ip := range ips {
			if net.ParseIP(ip).To4() == nil {
				continue // IPv6 exception routing isn't handled yet; see README known gaps.
			}
			if err := system.AddExceptionRoute(ip); err != nil {
				for _, r := range addedRoutes {
					_ = system.RemoveExceptionRoute(r)
				}
				m.status = Status{State: StateError, Message: err.Error()}
				return err
			}
			addedRoutes = append(addedRoutes, ip)
		}
	}

	pbConfig, err := BuildConfig(server, mode)
	if err != nil {
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	instance, err := core.New(pbConfig)
	if err != nil {
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	if err := instance.Start(); err != nil {
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	m.instance = instance
	m.exceptionRoutes = addedRoutes
	m.status = Status{State: StateRunning, Server: server.Name, Mode: mode}

	m.uplinkCounter, m.downlinkCounter = nil, nil
	if sm := instance.GetFeature(stats.ManagerType()); sm != nil {
		if manager, ok := sm.(stats.Manager); ok {
			// Registered by the "proxy" outbound because buildJSON turns on
			// policy.system.statsOutboundUplink/Downlink -- see config.go.
			m.uplinkCounter = manager.GetCounter("outbound>>>proxy>>>traffic>>>uplink")
			m.downlinkCounter = manager.GetCounter("outbound>>>proxy>>>traffic>>>downlink")
		}
	}
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, r := range m.exceptionRoutes {
		_ = system.RemoveExceptionRoute(r)
	}
	m.exceptionRoutes = nil

	if m.instance == nil {
		m.status = Status{State: StateStopped}
		return nil
	}

	err := m.instance.Close()
	m.instance = nil
	m.uplinkCounter, m.downlinkCounter = nil, nil
	m.status = Status{State: StateStopped}
	return err
}

func (m *Manager) Restart(server profile.Server, mode Mode) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(server, mode)
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}
