package singbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"

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

// Manager owns the lifecycle of a single running sing-box instance.
type Manager struct {
	mu              sync.Mutex
	status          Status
	instance        *box.Box
	cancel          context.CancelFunc
	exceptionRoutes []string
}

func NewManager() *Manager {
	return &Manager{status: Status{State: StateStopped}}
}

func (m *Manager) Start(server profile.Server, mode Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State == StateRunning {
		return errors.New("sing-box is already running; call Stop or Restart first")
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

	configBytes, err := buildJSON(server, mode)
	if err != nil {
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = include.Context(ctx)

	var opts option.Options
	if err := singjson.UnmarshalContext(ctx, configBytes, &opts); err != nil {
		cancel()
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		err = fmt.Errorf("invalid generated sing-box config: %w", err)
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	// TUN mode specifically gets retried: sing-box's TUN inbound doesn't
	// finish tearing down the previous Wintun adapter/session
	// synchronously within Close(), so reconnecting soon after a TUN
	// disconnect can hit that still-being-released adapter/session and
	// fail with a Windows "An attempt was made to perform an
	// initialization operation when initialization has already been
	// completed" error. Retry rather than guess a fixed pause -- each
	// attempt rebuilds a fresh instance, since one that failed to
	// Start() isn't safely reusable.
	attempts := 1
	if mode == ModeTUN {
		attempts = 5
	}

	var instance *box.Box
	for attempt := 1; attempt <= attempts; attempt++ {
		instance, err = box.New(box.Options{Context: ctx, Options: opts})
		if err != nil {
			break
		}
		err = instance.Start()
		if err == nil {
			break
		}
		if attempt < attempts && isAdapterStillReleasing(err) {
			time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		cancel()
		for _, r := range addedRoutes {
			_ = system.RemoveExceptionRoute(r)
		}
		if mode == ModeTUN && missingTUNDLL() {
			// Kite writes wintun.dll next to itself right before this
			// (prepareTUN), so this specific failure almost always means
			// something deleted it in between -- overwhelmingly antivirus/
			// Windows Defender quarantining it. Wintun-based apps (Kite,
			// WireGuard, v2rayN, ...) commonly need an AV exclusion for
			// their install folder because a kernel-adjacent networking
			// DLL like this gets flagged heuristically.
			err = fmt.Errorf("%w -- wintun.dll went missing right after Kite wrote it, most likely quarantined by antivirus/Windows Defender; try adding Kite's folder to your antivirus exclusions", err)
		} else if mode == ModeTUN && isAdapterStillReleasing(err) {
			err = fmt.Errorf("%w -- the previous TUN session didn't finish releasing its network adapter in time; wait a few seconds and try connecting again", err)
		}
		m.status = Status{State: StateError, Message: err.Error()}
		return err
	}

	m.instance = instance
	m.cancel = cancel
	m.exceptionRoutes = addedRoutes
	m.status = Status{State: StateRunning, Server: server.Name, Mode: mode}
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
	if m.cancel != nil {
		m.cancel()
	}
	m.instance = nil
	m.status = Status{State: StateStopped}
	// A subsequent TUN-mode Start() retries through the "adapter still
	// releasing" window itself (see isAdapterStillReleasing) rather than
	// this method guessing a fixed wait up front.
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

// isAdapterStillReleasing reports whether err looks like the Windows
// "already initialized" error a Wintun adapter/session that hasn't
// finished being released by the previous connection produces.
func isAdapterStillReleasing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "already been completed")
}
