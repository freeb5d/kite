// Package xray wraps xray-core as an in-process Go library: it builds a
// config from a server profile, and starts/stops/restarts an xray instance
// without shelling out to an external binary.
package xray

import (
	"errors"
	"sync"

	"github.com/freeb5d/kite/internal/profile"
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
	mu     sync.Mutex
	status Status
	// instance will hold the running *core.Instance from xray-core once
	// config.go builds a real config and startup wires it in.
	instance interface{}
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

	cfg, err := BuildConfig(server)
	if err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	// TODO: instantiate xray-core with cfg and keep the handle in m.instance.
	_ = cfg

	m.status = Status{State: StateRunning, Server: server.Name}
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State != StateRunning {
		m.status = Status{State: StateStopped}
		return nil
	}

	// TODO: close m.instance cleanly.
	m.instance = nil
	m.status = Status{State: StateStopped}
	return nil
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
