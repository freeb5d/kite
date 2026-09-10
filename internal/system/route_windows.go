//go:build windows

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// AddExceptionRoute pins a /32 host route to ip via the machine's current
// real default gateway. This must run *before* xray's TUN inbound brings
// up its own default route through the TUN adapter (via
// autoSystemRoutingTable) -- otherwise xray's own outbound connection to
// that same ip (the VPN server itself) would get routed back through the
// TUN interface it's trying to feed, an infinite loop. See the "PLEASE
// READ" section of xray-core's proxy/tun/README.md.
func AddExceptionRoute(ip string) error {
	gateway, err := defaultGateway()
	if err != nil {
		return fmt.Errorf("find default gateway: %w", err)
	}
	cmd := exec.Command("route", "add", ip, "mask", "255.255.255.255", gateway, "metric", "5")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("route add %s via %s: %w (%s)", ip, gateway, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoveExceptionRoute undoes AddExceptionRoute.
func RemoveExceptionRoute(ip string) error {
	return exec.Command("route", "delete", ip).Run()
}

// defaultGateway parses `route print -4` for the current IPv4 default
// (0.0.0.0/0) gateway -- the "real" internet-facing gateway, before any
// TUN adapter route is added.
func defaultGateway() (string, error) {
	out, err := exec.Command("route", "print", "-4").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			return fields[2], nil
		}
	}
	return "", fmt.Errorf("no default route found in `route print` output")
}
