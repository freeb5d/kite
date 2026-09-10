// Package xray wraps xray-core as an in-process Go library: it builds a
// config from a server profile, and starts/stops/restarts an xray instance
// without shelling out to an external binary.
package xray

import (
	"errors"
	"sync"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/xtls/xray-core/core"

	// Registers every protocol/transport xray-core ships (vmess, vless,
	// trojan, shadowsocks, http/socks inbounds, ws/tls, ...) with the
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
	Message string `json:"message,omitempty"`
}

// Manager owns the lifecycle of a single running xray-core instance.
type Manager struct {
	mu       sync.Mutex
	status   Status
	instance *core.Instance
}

func NewManager() *Manager {
	return &Manager{status: Status{State: StateStopped}}
}

func (m *Manager) Start(server profile.Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State == StateRunning {
		return errors.New("xray is already running; call Stop or Restart first")
	}

	pbConfig, err := BuildConfig(server)
	if err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	instance, err := core.New(pbConfig)
	if err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	if err := instance.Start(); err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	m.instance = instance
	m.status = Status{State: StateRunning, Server: server.Name}
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance == nil {
		m.status = Status{State: StateStopped}
		return nil
	}

	err := m.instance.Close()
	m.instance = nil
	m.status = Status{State: StateStopped}
	return err
}

func (m *Manager) Restart(server profile.Server) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(server)
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}
