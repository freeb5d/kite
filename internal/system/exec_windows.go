//go:build windows

package system

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// command is exec.Command without the console window Windows otherwise
// flashes up for every console program a GUI (windowsgui) process spawns.
func command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd
}
