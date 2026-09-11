//go:build windows

package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	killSwitchAllowRule = "Kite Kill Switch Allow"
	killSwitchBlockRule = "Kite Kill Switch Block"
)

// EnableKillSwitch blocks all outbound traffic except from Kite's own
// process (which hosts the embedded xray-core and needs to reach the VPN
// server) and loopback -- Windows Firewall never filters 127.0.0.1
// traffic regardless of rules, so apps that honor the system HTTP/SOCKS
// proxy (which listens on loopback) keep working. Anything that dials
// out directly instead of through the tunnel -- the traffic a kill
// switch exists to stop -- gets cut off rather than silently leaking,
// including if xray crashes while connected.
//
// Requires administrator privileges, same as TUN mode: creating firewall
// rules needs them.
func EnableKillSwitch() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate running executable: %w", err)
	}

	if err := netsh(
		"advfirewall", "firewall", "add", "rule",
		"name="+killSwitchAllowRule,
		"dir=out", "action=allow", "enable=yes",
		"program="+exePath,
	); err != nil {
		return fmt.Errorf("add kill switch allow rule: %w", err)
	}

	if err := netsh(
		"advfirewall", "firewall", "add", "rule",
		"name="+killSwitchBlockRule,
		"dir=out", "action=block", "enable=yes",
	); err != nil {
		_ = removeFirewallRule(killSwitchAllowRule)
		return fmt.Errorf("add kill switch block rule: %w", err)
	}
	return nil
}

// DisableKillSwitch removes the firewall rules EnableKillSwitch added.
// Safe to call even if they were never added.
func DisableKillSwitch() error {
	errAllow := removeFirewallRule(killSwitchAllowRule)
	errBlock := removeFirewallRule(killSwitchBlockRule)
	if errBlock != nil {
		return errBlock
	}
	return errAllow
}

func removeFirewallRule(name string) error {
	err := netsh("advfirewall", "firewall", "delete", "rule", "name="+name)
	if err != nil && strings.Contains(err.Error(), "No rules match") {
		return nil
	}
	return err
}

func netsh(args ...string) error {
	out, err := exec.Command("netsh", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
