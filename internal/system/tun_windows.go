//go:build windows

package system

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// SetupTUNInterface configures the Wintun adapter xray-core created: xray
// only brings the adapter up, so the address, DNS server and routes that
// actually send traffic into it are Kite's job. Two /1 routes override the
// default route without replacing it; interface-bound routes and settings
// disappear on their own when xray closes the adapter.
//
// The server's exception route (AddExceptionRoute) must already be in place,
// or xray's own connection to the server would loop back into the adapter.
func SetupTUNInterface(name, addr, mask, dns string) error {
	var iface *net.Interface
	for i := 0; i < 50; i++ {
		if found, err := net.InterfaceByName(name); err == nil {
			iface = found
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if iface == nil {
		return fmt.Errorf("TUN adapter %q didn't appear", name)
	}
	idx := strconv.Itoa(iface.Index)

	steps := [][]string{
		{"netsh", "interface", "ipv4", "set", "address", "name=" + idx, "source=static", "address=" + addr, "mask=" + mask},
		{"netsh", "interface", "ipv4", "set", "dnsservers", "name=" + idx, "source=static", "address=" + dns, "register=none", "validate=no"},
		// Lowest metric so Windows prefers this adapter's DNS server over
		// the LAN's (which would leak queries and may be poisoned).
		{"netsh", "interface", "ipv4", "set", "interface", "interface=" + idx, "metric=1"},
		{"route", "add", "0.0.0.0", "mask", "128.0.0.0", addr, "metric", "1", "if", idx},
		{"route", "add", "128.0.0.0", "mask", "128.0.0.0", addr, "metric", "1", "if", idx},
	}
	for _, step := range steps {
		if out, err := command(step[0], step[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %w (%s)", step, err, out)
		}
	}
	return nil
}

// RemoveStaleTUNRoutes deletes /1 routes through a gateway starting with
// prefix that a previous run left behind (crash, or an adapter xray never
// released) -- they'd send all traffic into a dead tunnel.
func RemoveStaleTUNRoutes(prefix string) {
	out, err := command("route", "print", "-4").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 && (f[0] == "0.0.0.0" || f[0] == "128.0.0.0") && f[1] == "128.0.0.0" && strings.HasPrefix(f[2], prefix) {
			_ = command("route", "delete", f[0], "mask", f[1], f[2]).Run()
		}
	}
}

// TeardownTUNInterface removes the routes SetupTUNInterface added. xray
// doesn't always release the Wintun adapter on Close, and a lingering
// adapter keeps its routes -- so all traffic (including xray's own dial to
// the server, once its exception route is gone) would keep flowing into a
// dead tunnel. Removing the routes first stops that regardless.
func TeardownTUNInterface(addr string) {
	for _, dst := range []string{"0.0.0.0", "128.0.0.0"} {
		_ = command("route", "delete", dst, "mask", "128.0.0.0", addr).Run()
	}
}
