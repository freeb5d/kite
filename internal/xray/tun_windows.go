//go:build windows

package xray

import (
	"os"
	"path/filepath"

	"github.com/freeb5d/kite/internal/system/wintun"
)

// prepareTUN writes the embedded wintun.dll next to the running
// executable if it isn't there yet -- xray-core's TUN inbound looks for
// it in that exact location on Windows.
func prepareTUN() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	dllPath := filepath.Join(filepath.Dir(exePath), "wintun.dll")
	if _, err := os.Stat(dllPath); err == nil {
		return nil
	}
	return os.WriteFile(dllPath, wintun.DLL, 0o644)
}
