// Package xray is Kite's engine facade (package/directory keeps the
// historical name from when it directly wrapped xray-core -- renaming it
// wasn't worth the churn across the rest of the codebase). It dispatches
// every call to whichever concrete engine is currently selected:
// internal/engine/xraycore (the default -- supports more link shapes, no
// TUN) or internal/engine/singbox (the only one with TUN support). The
// public API here is exactly what it was when this package *was* the
// sing-box implementation, so app.go needed zero changes when this facade
// was introduced.
package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/freeb5d/kite/internal/engine/singbox"
	"github.com/freeb5d/kite/internal/engine/xraycore"
	"github.com/freeb5d/kite/internal/profile"
)

const (
	HTTPInboundPort  = xraycore.HTTPInboundPort
	SOCKSInboundPort = xraycore.SOCKSInboundPort
)

type Mode = string

const (
	ModeProxy Mode = "proxy"
	ModeTUN   Mode = "tun"
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

// Engine identifies which concrete engine a Manager is currently backed
// by -- persisted so the choice survives a restart.
type Engine string

const (
	// EngineXrayCore is the default: xray-core supports more link shapes
	// (notably tcp+headerType=http, which sing-box has no equivalent of)
	// but has no TUN support here.
	EngineXrayCore Engine = "xray"
	// EngineSingBox is the only engine with TUN support.
	EngineSingBox Engine = "singbox"
)

// Manager owns the lifecycle of a single running proxy engine instance,
// delegating to whichever concrete engine is currently selected (see
// CurrentEngine/SetEngine). Only one of singbox/xrayCore is non-nil at a
// time.
type Manager struct {
	mu      sync.Mutex
	engine  Engine
	singbox *singbox.Manager
	xrayCore *xraycore.Manager
}

func NewManager() *Manager {
	return &Manager{}
}

// activeLocked returns (and lazily creates) the concrete manager for
// whatever engine is currently selected. Must be called with m.mu held.
func (m *Manager) activeLocked() Engine {
	engine := CurrentEngine()
	if engine == m.engine {
		return engine
	}
	// The selected engine changed since the last call -- this only ever
	// happens while stopped (the frontend won't let you change engines
	// while connected), so there's no running instance to tear down.
	m.engine = engine
	switch engine {
	case EngineSingBox:
		if m.singbox == nil {
			m.singbox = singbox.NewManager()
		}
	default:
		if m.xrayCore == nil {
			m.xrayCore = xraycore.NewManager()
		}
	}
	return engine
}

func (m *Manager) Start(server profile.Server, mode Mode) error {
	m.mu.Lock()
	engine := m.activeLocked()
	m.mu.Unlock()

	if engine == EngineSingBox {
		return m.singbox.Start(server, mode)
	}
	return m.xrayCore.Start(server, mode)
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	engine := m.engine
	sb, xc := m.singbox, m.xrayCore
	m.mu.Unlock()

	if engine == EngineSingBox && sb != nil {
		return sb.Stop()
	}
	if xc != nil {
		return xc.Stop()
	}
	return nil
}

func (m *Manager) Restart(server profile.Server, mode Mode) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(server, mode)
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	engine := m.engine
	sb, xc := m.singbox, m.xrayCore
	m.mu.Unlock()

	if engine == EngineSingBox {
		if sb == nil {
			return Status{State: StateStopped}
		}
		s := sb.Status()
		return Status{State: State(s.State), Server: s.Server, Mode: s.Mode, Message: s.Message}
	}
	if xc == nil {
		return Status{State: StateStopped}
	}
	s := xc.Status()
	return Status{State: State(s.State), Server: s.Server, Mode: s.Mode, Message: s.Message}
}

func (m *Manager) Traffic() Traffic {
	m.mu.Lock()
	engine := m.engine
	sb, xc := m.singbox, m.xrayCore
	m.mu.Unlock()

	if engine == EngineSingBox {
		if sb == nil {
			return Traffic{}
		}
		t := sb.Traffic()
		return Traffic{Uplink: t.Uplink, Downlink: t.Downlink}
	}
	if xc == nil {
		return Traffic{}
	}
	t := xc.Traffic()
	return Traffic{Uplink: t.Uplink, Downlink: t.Downlink}
}

// LogFilePath returns where the *currently selected* engine's own debug
// log is written -- not necessarily the engine that produced the last
// connection, if the selection changed since.
func LogFilePath() string {
	if CurrentEngine() == EngineSingBox {
		return singbox.LogFilePath()
	}
	return xraycore.LogFilePath()
}

// CoreVersion returns the embedded version of the currently selected
// engine, for the About panel.
func CoreVersion() string {
	if CurrentEngine() == EngineSingBox {
		return singbox.CoreVersion()
	}
	return xraycore.CoreVersion()
}

// EngineName returns a human-readable label for the currently selected
// engine, for the About panel.
func EngineName() string {
	if CurrentEngine() == EngineSingBox {
		return "sing-box"
	}
	return "xray-core"
}

// --- Engine selection persistence ---

type settings struct {
	Engine Engine `json:"engine"`
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	kiteDir := filepath.Join(dir, "kite")
	_ = os.MkdirAll(kiteDir, 0o700)
	return filepath.Join(kiteDir, "settings.json")
}

// CurrentEngine returns the persisted engine choice, defaulting to
// xray-core if nothing's been saved yet (or the file is unreadable).
func CurrentEngine() Engine {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return EngineXrayCore
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return EngineXrayCore
	}
	if s.Engine != EngineSingBox {
		return EngineXrayCore
	}
	return s.Engine
}

// SetEngine persists the engine choice for future connections (and future
// launches). It's rejected while a connection is active -- the caller
// (App.SetEngine) checks that before calling this.
func SetEngine(engine Engine) error {
	if engine != EngineXrayCore && engine != EngineSingBox {
		return fmt.Errorf("unknown engine %q", engine)
	}
	data, err := json.MarshalIndent(settings{Engine: engine}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0o600)
}
