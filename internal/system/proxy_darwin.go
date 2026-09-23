//go:build darwin

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// networkServices lists every enabled network service (Wi-Fi, Ethernet,
// USB tethering, ...) -- the proxy is per-service on macOS, so setting it
// on "Wi-Fi" alone leaves wired connections unproxied.
func networkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	var services []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// First line is an explanatory header; "*" marks disabled services.
		if i == 0 || line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func SetProxy(host string, httpPort, socksPort int) error {
	services, err := networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		for _, args := range [][]string{
			{"-setwebproxy", svc, host, fmt.Sprint(httpPort)},
			{"-setsecurewebproxy", svc, host, fmt.Sprint(httpPort)},
			{"-setsocksfirewallproxy", svc, host, fmt.Sprint(socksPort)},
		} {
			if err := exec.Command("networksetup", args...).Run(); err != nil {
				return fmt.Errorf("networksetup %s %s: %w", args[0], svc, err)
			}
		}
	}
	return nil
}

func ClearProxy() error {
	services, err := networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		for _, flag := range []string{"-setwebproxystate", "-setsecurewebproxystate", "-setsocksfirewallproxystate"} {
			_ = exec.Command("networksetup", flag, svc, "off").Run()
		}
	}
	return nil
}

func ClearStaleProxy(host string, port int) error {
	return nil
}
