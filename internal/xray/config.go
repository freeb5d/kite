// Package xray runs xray-core (github.com/xtls/xray-core) as an in-process
// Go library: it builds a config from a server profile and starts/stops an
// xray-core instance, in proxy mode (local HTTP/SOCKS) or TUN mode.
package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/freeb5d/kite/pkg/profile"
	"github.com/freeb5d/kite/pkg/xrayconf"
)

// Local proxy ports (proxy mode). Defaults; the user can change them
// (SetPorts), and the choice is kept in ports.json in Kite's config folder.
var (
	HTTPInboundPort  = 2080
	SOCKSInboundPort = 2081
)

func portsFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "kite", "ports.json")
}

func init() {
	var p struct{ HTTP, SOCKS int }
	if b, err := os.ReadFile(portsFile()); err == nil && json.Unmarshal(b, &p) == nil && validPorts(p.HTTP, p.SOCKS) == nil {
		HTTPInboundPort, SOCKSInboundPort = p.HTTP, p.SOCKS
	}
}

func validPorts(http, socks int) error {
	if http < 1024 || http > 65535 || socks < 1024 || socks > 65535 {
		return fmt.Errorf("ports must be between 1024 and 65535")
	}
	if http == socks {
		return fmt.Errorf("HTTP and SOCKS ports must be different")
	}
	return nil
}

// SetPorts changes the local proxy ports for the next connection and saves them.
func SetPorts(http, socks int) error {
	if err := validPorts(http, socks); err != nil {
		return err
	}
	HTTPInboundPort, SOCKSInboundPort = http, socks
	b, _ := json.Marshal(struct{ HTTP, SOCKS int }{http, socks})
	_ = os.MkdirAll(filepath.Dir(portsFile()), 0o700)
	return os.WriteFile(portsFile(), b, 0o600)
}

const (
	// TUN adapter settings. Each connection gets its own 172.19.N.1/30
	// (see manager.go); the kill switch allows all of 172.19.0.0/16.
	TUNMask = "255.255.255.252"
	TUNDNS  = "1.1.1.1"
)

// Mode selects what xray-core listens on: a local HTTP/SOCKS proxy
// ("proxy") or a TUN adapter capturing all system traffic ("tun").
type Mode = string

const (
	ModeProxy Mode = "proxy"
	ModeTUN   Mode = "tun"
)

// CoreVersion returns the embedded xray-core version, for the About panel.
// xray-core doesn't expose a runtime version constant, so this reads the
// resolved module version straight from the Go module graph instead.
func CoreVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/xtls/xray-core" {
				return strings.TrimPrefix(dep.Version, "v")
			}
		}
	}
	return "unknown"
}

// LogFilePath returns where xray-core's own error log is written.
func LogFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	logDir := filepath.Join(dir, "kite", "logs")
	_ = os.MkdirAll(logDir, 0o700)
	return filepath.Join(logDir, "xray.log")
}

func buildJSON(server profile.Server, mode Mode, tunName string, sshBridgePort int) ([]byte, error) {
	return xrayconf.Build(server, xrayconf.Options{
		HTTPPort:      HTTPInboundPort,
		SOCKSPort:     SOCKSInboundPort,
		TUN:           mode == ModeTUN,
		TUNName:       tunName,
		LogPath:       LogFilePath(),
		SSHBridgePort: sshBridgePort,
	})
}

var firstNonEmpty = xrayconf.FirstNonEmpty
