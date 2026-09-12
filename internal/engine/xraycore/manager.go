package xraycore

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"

	// Registers every inbound/outbound/proxy protocol implementation
	// (vmess, vless, trojan, shadowsocks, http, socks, freedom, ...) via
	// their init() functions -- core.New fails to build any of them
	// without this, since xray-core resolves protocols from a registry
	// rather than linking them in directly.
	_ "github.com/xtls/xray-core/main/distro/all"

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
	Mode    Mode   `json:"mode,omitempty"`
	Message string `json:"message,omitempty"`
}

type Traffic struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

// Manager owns the lifecycle of a single running xray-core instance.
//
// Unlike sing-box, xray-core (mainline) has no native TUN inbound -- Kite's
// original TUN support before the sing-box migration relied on sing-box's
// own tun package even then in practice, so Start refuses ModeTUN outright
// rather than pretending to support it. See README known gaps.
type Manager struct {
	mu       sync.Mutex
	status   Status
	instance *core.Instance
}

func NewManager() *Manager {
	return &Manager{status: Status{State: StateStopped}}
}

func (m *Manager) Start(server profile.Server, mode Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State == StateRunning {
		return errors.New("xray-core is already running; call Stop or Restart first")
	}

	if mode == ModeTUN {
		err := fmt.Errorf("TUN mode isn't supported under the xray-core engine -- switch to sing-box in Settings and try again")
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	configBytes, err := buildJSON(server)
	if err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	config, err := serial.LoadJSONConfig(bytes.NewReader(configBytes))
	if err != nil {
		err = fmt.Errorf("invalid generated xray-core config: %w", err)
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	instance, err := core.New(config)
	if err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	if err := instance.Start(); err != nil {
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	m.instance = instance
	m.status = Status{State: StateRunning, Server: server.Name, Mode: mode}
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

// Traffic always reports zero -- xray-core's stats.Manager isn't wired up
// here, matching sing-box's own stubbed Traffic() (see
// internal/engine/singbox/stats.go). Follow-up for either engine.
func (m *Manager) Traffic() Traffic {
	return Traffic{}
}
