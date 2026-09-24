package xray

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/infra/conf/serial"

	// Registers every inbound/outbound/proxy protocol implementation via
	// their init() functions -- core.New can't build any of them otherwise.
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/freeb5d/kite/internal/system"
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
type Manager struct {
	mu              sync.Mutex
	status          Status
	instance        *core.Instance
	exceptionRoutes []string
	tunAddr         string
	tunSeq          int
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
		return errors.New("xray-core is already running; call Stop or Restart first")
	}

	err := m.start(server, mode)
	if err != nil {
		m.removeRoutes()
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}
	m.status = Status{State: StateRunning, Server: server.Name, Mode: mode}
	return nil
}

func (m *Manager) start(server profile.Server, mode Mode) error {
	if mode == ModeTUN {
		if err := prepareTUN(); err != nil {
			return fmt.Errorf("prepare TUN: %w", err)
		}
		pinned, ips, err := pinServerAddress(server)
		if err != nil {
			return err
		}
		server = pinned
		for _, ip := range ips {
			if err := system.AddExceptionRoute(ip); err != nil {
				return err
			}
			m.exceptionRoutes = append(m.exceptionRoutes, ip)
		}
	}

	// A fresh adapter name per connection: xray doesn't always release the
	// previous adapter on Close, and reopening one with a live session
	// fails with "initialization has already been completed".
	m.tunSeq++
	tunName := fmt.Sprintf("kite-tun-%d", m.tunSeq)
	tunAddr := TUNAddress(m.tunSeq)

	configBytes, err := buildJSON(server, mode, tunName)
	if err != nil {
		return err
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(configBytes))
	if err != nil {
		return fmt.Errorf("invalid generated xray-core config: %w", err)
	}

	// A Wintun adapter from a TUN session that just ended isn't always
	// released yet, so creating it again fails with "initialization has
	// already been completed". Retry with a fresh instance each time.
	attempts := 1
	if mode == ModeTUN {
		attempts = 5
	}
	var instance *core.Instance
	for attempt := 1; attempt <= attempts; attempt++ {
		instance, err = core.New(config)
		if err == nil {
			err = instance.Start()
			if err != nil {
				_ = instance.Close()
			}
		}
		if err == nil || attempt == attempts || !isAdapterStillReleasing(err) {
			break
		}
		time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
	}
	if err != nil {
		switch {
		case mode == ModeTUN && missingTUNDLL():
			return fmt.Errorf("%w -- wintun.dll went missing right after Kite wrote it, most likely quarantined by antivirus/Windows Defender; try adding Kite's folder to your antivirus exclusions", err)
		case mode == ModeTUN && isAdapterStillReleasing(err):
			return fmt.Errorf("%w -- the previous TUN session didn't finish releasing its network adapter in time; wait a few seconds and try connecting again", err)
		}
		return err
	}

	if mode == ModeTUN {
		m.tunAddr = tunAddr
		if err := system.SetupTUNInterface(tunName, tunAddr, TUNMask, TUNDNS); err != nil {
			system.TeardownTUNInterface(tunAddr)
			m.tunAddr = ""
			_ = instance.Close()
			return fmt.Errorf("configure TUN adapter: %w", err)
		}
	}

	m.instance = instance
	m.uplinkCounter, m.downlinkCounter = nil, nil
	if sm, ok := instance.GetFeature(stats.ManagerType()).(stats.Manager); ok && sm != nil {
		m.uplinkCounter = getOrRegisterCounter(sm, "outbound>>>proxy>>>traffic>>>uplink")
		m.downlinkCounter = getOrRegisterCounter(sm, "outbound>>>proxy>>>traffic>>>downlink")
	}
	return nil
}

// getOrRegisterCounter stands in for stats.Manager.GetOrRegisterCounter,
// which the xray-core version pinned in go.mod doesn't have yet.
func getOrRegisterCounter(m stats.Manager, name string) stats.Counter {
	if c := m.GetCounter(name); c != nil {
		return c
	}
	c, err := m.RegisterCounter(name)
	if err != nil {
		return nil
	}
	return c
}

// pinServerAddress resolves the server's hostname up front and dials its
// IP instead, keeping the hostname for SNI and the Host header. In TUN mode
// xray-core resolving the hostname itself would go through Windows DNS,
// which is routed into the tunnel -- which needs that very connection.
// Returns the IPv4 addresses that need an exception route.
func pinServerAddress(server profile.Server) (profile.Server, []string, error) {
	host := server.Address
	ips, err := net.LookupHost(host)
	if err != nil {
		return server, nil, fmt.Errorf("resolve %s: %w", host, err)
	}
	var v4 []string
	for _, ip := range ips {
		if net.ParseIP(ip).To4() != nil {
			v4 = append(v4, ip)
		}
	}
	if len(v4) == 0 {
		return server, nil, fmt.Errorf("%s has no IPv4 address; TUN mode is IPv4-only for now", host)
	}
	if net.ParseIP(host) == nil {
		extra := make(map[string]string, len(server.Extra)+2)
		for k, v := range server.Extra {
			extra[k] = v
		}
		if extra["sni"] == "" {
			extra["sni"] = firstNonEmpty(extra["host"], host)
		}
		if extra["host"] == "" {
			extra["host"] = host
		}
		server.Extra = extra
		server.Address = v4[0]
	}
	return server, v4, nil
}

func (m *Manager) removeRoutes() {
	for _, ip := range m.exceptionRoutes {
		_ = system.RemoveExceptionRoute(ip)
	}
	m.exceptionRoutes = nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Routes first: if xray leaves the adapter behind, traffic must not
	// keep flowing into it.
	if m.tunAddr != "" {
		system.TeardownTUNInterface(m.tunAddr)
		m.tunAddr = ""
	}
	var err error
	if m.instance != nil {
		err = m.instance.Close()
		m.instance = nil
	}
	m.removeRoutes()
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

// Traffic reads the "proxy" outbound's cumulative byte counters from
// xray-core's stats.Manager (enabled via the policy block in config.go).
func (m *Manager) Traffic() Traffic {
	m.mu.Lock()
	defer m.mu.Unlock()

	var t Traffic
	if m.uplinkCounter != nil {
		t.Uplink = m.uplinkCounter.Value()
	}
	if m.downlinkCounter != nil {
		t.Downlink = m.downlinkCounter.Value()
	}
	return t
}

// TUNAddress is the adapter address for the n-th TUN connection. A new one
// each time, since a lingering adapter from a previous connection may still
// hold the old address.
func TUNAddress(n int) string {
	return fmt.Sprintf("172.19.%d.1", n%250)
}

func isAdapterStillReleasing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "already been completed")
}
