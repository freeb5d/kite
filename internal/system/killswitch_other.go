//go:build !windows

package system

import "fmt"

// EnableKillSwitch/DisableKillSwitch are Windows-only for now (Windows
// Firewall via netsh); see killswitch_windows.go. The UI hides the Kill
// Switch toggle on other platforms, same as TUN mode.
func EnableKillSwitch() error {
	return fmt.Errorf("kill switch is currently only supported on Windows")
}

func DisableKillSwitch() error {
	return nil
}
