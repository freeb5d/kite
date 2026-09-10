//go:build linux

package system

import (
	"fmt"
	"os/exec"
)

// SetProxy uses gsettings, which covers GNOME and most GTK-based desktops.
// Other desktop environments (KDE, etc.) will need their own backend later.
func SetProxy(host string, port int) error {
	if err := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run(); err != nil {
		return err
	}
	for _, kind := range []string{"http", "https", "socks"} {
		key := fmt.Sprintf("org.gnome.system.proxy.%s", kind)
		if err := exec.Command("gsettings", "set", key, "host", host).Run(); err != nil {
			return err
		}
		if err := exec.Command("gsettings", "set", key, "port", fmt.Sprintf("%d", port)).Run(); err != nil {
			return err
		}
	}
	return nil
}

func ClearProxy() error {
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
}
