//go:build darwin

package system

import (
	"fmt"
	"os/exec"
)

// networkService is the default macOS interface service name; a real
// implementation should detect the active service via `networksetup -listallnetworkservices`.
const networkService = "Wi-Fi"

func SetProxy(host string, port int) error {
	portStr := fmt.Sprintf("%d", port)
	if err := exec.Command("networksetup", "-setwebproxy", networkService, host, portStr).Run(); err != nil {
		return err
	}
	return exec.Command("networksetup", "-setsocksfirewallproxy", networkService, host, portStr).Run()
}

func ClearProxy() error {
	if err := exec.Command("networksetup", "-setwebproxystate", networkService, "off").Run(); err != nil {
		return err
	}
	return exec.Command("networksetup", "-setsocksfirewallproxystate", networkService, "off").Run()
}
