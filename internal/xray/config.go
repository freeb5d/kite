// Package xray runs xray-core (github.com/xtls/xray-core) as an in-process
// Go library: it builds a config from a server profile and starts/stops an
// xray-core instance, in proxy mode (local HTTP/SOCKS) or TUN mode.
package xray

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/freeb5d/kite/pkg/profile"
	"github.com/freeb5d/kite/pkg/xrayconf"
)

const (
	HTTPInboundPort  = 2080
	SOCKSInboundPort = 2081

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

func buildJSON(server profile.Server, mode Mode, tunName string) ([]byte, error) {
	return xrayconf.Build(server, xrayconf.Options{
		HTTPPort:  HTTPInboundPort,
		SOCKSPort: SOCKSInboundPort,
		TUN:       mode == ModeTUN,
		TUNName:   tunName,
		LogPath:   LogFilePath(),
	})
}

var firstNonEmpty = xrayconf.FirstNonEmpty
