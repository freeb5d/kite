//go:build windows

// Package system sets and clears the OS-level HTTP/SOCKS proxy so that
// system traffic routes through the local xray inbound (phase 1: no TUN).
package system

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const internetSettingsPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// SetProxy writes the per-user Internet Settings registry keys that
// browsers and most Windows apps read for their proxy configuration.
// This intentionally does NOT use `netsh winhttp set proxy`: that command
// requires an elevated (admin) process, and it only affects WinHTTP-based
// services (e.g. Windows Update), not the browsers users actually care
// about — HKCU Internet Settings is what those honor, and it needs no
// elevation since it's a per-user key.
func SetProxy(host string, port int) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open internet settings: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	if err := key.SetStringValue("ProxyServer", fmt.Sprintf("%s:%d", host, port)); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}
	if err := key.SetStringValue("ProxyOverride", "<local>"); err != nil {
		return fmt.Errorf("set ProxyOverride: %w", err)
	}

	notifySettingsChanged()
	return nil
}

// ClearProxy resets the Windows system proxy to direct.
func ClearProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open internet settings: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}

	notifySettingsChanged()
	return nil
}

// notifySettingsChanged tells already-running apps (browsers, etc.) to pick
// up the new proxy settings immediately instead of waiting for their own
// poll interval. Best-effort: failures here don't affect the registry
// change itself, which is what actually matters.
func notifySettingsChanged() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	setOption := wininet.NewProc("InternetSetOptionW")

	const (
		internetOptionSettingsChanged = 39
		internetOptionRefresh         = 37
	)

	setOption.Call(0, internetOptionSettingsChanged, 0, 0)
	setOption.Call(0, internetOptionRefresh, 0, 0)
}
