//go:build windows

package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// installed reports whether this copy of Kite was set up by the installer
// (which leaves uninstall.exe next to kite.exe) rather than run portable.
func installed(exePath string) bool {
	_, err := os.Stat(filepath.Join(filepath.Dir(exePath), "uninstall.exe"))
	return err == nil
}

// runInstaller launches the downloaded setup silently once Kite has quit,
// then starts the updated Kite. Installing into Program Files needs admin,
// so Windows shows one UAC prompt.
func runInstaller(setupPath, exePath string) error {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "Start-Sleep -Seconds 2; " +
		"Start-Process -FilePath " + q(setupPath) + " -ArgumentList '/S' -Verb RunAs -Wait; " +
		"Remove-Item -LiteralPath " + q(setupPath) + " -ErrorAction SilentlyContinue; " +
		"Start-Process -FilePath " + q(exePath)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd.Start()
}
