//go:build linux

package system

import (
	"fmt"
	"os/exec"
)

// SetProxy uses gsettings, which covers GNOME and most GTK-based desktops.
// Other desktop environments (KDE, etc.) will need their own backend later.
func SetProxy(host string, httpPort, socksPort int) error {
	ports := map[string]int{"http": httpPort, "https": httpPort, "socks": socksPort}
	for kind, port := range ports {
		key := "org.gnome.system.proxy." + kind
		if err := exec.Command("gsettings", "set", key, "host", host).Run(); err != nil {
			return err
		}
		if err := exec.Command("gsettings", "set", key, "port", fmt.Sprint(port)).Run(); err != nil {
			return err
		}
	}
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
}

func ClearProxy() error {
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
}

func ClearStaleProxy(host string, port int) error {
	return nil
}
