//go:build windows

// Package system sets and clears the OS-level HTTP/SOCKS proxy so that
// system traffic routes through the local xray inbound (phase 1: no TUN).
package system

import (
	"fmt"
	"os/exec"
)

// SetProxy points the Windows system proxy (via the registry, through
// `netsh winhttp` for simplicity) at the local xray HTTP inbound.
func SetProxy(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	cmd := exec.Command("netsh", "winhttp", "set", "proxy", addr)
	return cmd.Run()
}

// ClearProxy resets the Windows system proxy to direct.
func ClearProxy() error {
	cmd := exec.Command("netsh", "winhttp", "reset", "proxy")
	return cmd.Run()
}
