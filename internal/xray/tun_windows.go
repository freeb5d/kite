//go:build windows

package xray

import (
	"os"
	"path/filepath"

	"github.com/freeb5d/kite/internal/system/wintun"
)

// prepareTUN writes the embedded wintun.dll next to the running
// executable if it isn't there yet -- sing-box's TUN inbound looks for
// it in that exact location on Windows (same as xray-core did).
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

// missingTUNDLL reports whether wintun.dll is (no longer) present next
// to the running executable -- used to give a clearer error than
// whatever generic OS message the underlying LoadLibrary call surfaces
// when it can't find it.
func missingTUNDLL() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(filepath.Dir(exePath), "wintun.dll"))
	return os.IsNotExist(err)
}
