//go:build windows

package system

import (
	"fmt"
	"os"
	"strings"
)

const (
	killSwitchAllowKite = "Kite Kill Switch Allow"
	killSwitchAllowTUN  = "Kite Kill Switch Allow TUN"
	killSwitchAllowDNS  = "Kite Kill Switch Allow DNS"
	// Rule name used by versions before the policy-based kill switch;
	// still removed on disable so an upgrade can't leave it behind.
	legacyKillSwitchBlock = "Kite Kill Switch Block"

	// Must match the address sing-box gives its TUN adapter
	// (internal/engine/singbox TUNGateway) -- traffic other apps send into
	// the tunnel leaves with this as its local address.
	tunSubnet = "172.19.0.0/30"
)

// EnableKillSwitch blocks all outbound traffic except Kite's own (which
// is how the embedded engine reaches the VPN server), traffic routed into
// the TUN adapter, and the Windows DNS Client service (Go resolves names
// through it on Windows, so the engine can't reach a hostname-based server
// without it). Loopback is never filtered, so the local HTTP/SOCKS proxy
// keeps working.
//
// It flips the default outbound policy to block instead of adding a
// "block everything" rule: Windows Firewall evaluates block rules before
// allow rules, so a block-all rule would also cut off Kite itself.
//
// Requires administrator privileges.
func EnableKillSwitch() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate running executable: %w", err)
	}

	rules := [][]string{
		{"name=" + killSwitchAllowKite, "program=" + exePath},
		{"name=" + killSwitchAllowTUN, "localip=" + tunSubnet},
		{"name=" + killSwitchAllowDNS, "service=dnscache", "protocol=udp", "remoteport=53"},
	}
	for _, rule := range rules {
		args := append([]string{"advfirewall", "firewall", "add", "rule", "dir=out", "action=allow", "enable=yes"}, rule...)
		if err := netsh(args...); err != nil {
			_ = DisableKillSwitch()
			return fmt.Errorf("add kill switch rule: %w", err)
		}
	}

	if err := netsh("advfirewall", "set", "allprofiles", "firewallpolicy", "blockinbound,blockoutbound"); err != nil {
		_ = DisableKillSwitch()
		return fmt.Errorf("block outbound traffic: %w", err)
	}
	return nil
}

// DisableKillSwitch restores the Windows default outbound policy (allow)
// and removes the rules EnableKillSwitch added. Safe to call even if the
// kill switch was never enabled.
func DisableKillSwitch() error {
	policyErr := netsh("advfirewall", "set", "allprofiles", "firewallpolicy", "blockinbound,allowoutbound")
	for _, name := range []string{killSwitchAllowKite, killSwitchAllowTUN, killSwitchAllowDNS, legacyKillSwitchBlock} {
		_ = removeFirewallRule(name)
	}
	return policyErr
}

// KillSwitchActive reports whether kill switch rules are present, e.g.
// left behind by a crash while connected.
func KillSwitchActive() bool {
	return netsh("advfirewall", "firewall", "show", "rule", "name="+killSwitchAllowKite) == nil
}

func removeFirewallRule(name string) error {
	err := netsh("advfirewall", "firewall", "delete", "rule", "name="+name)
	if err != nil && strings.Contains(err.Error(), "No rules match") {
		return nil
	}
	return err
}

func netsh(args ...string) error {
	out, err := command("netsh", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
