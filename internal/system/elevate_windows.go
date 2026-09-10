//go:build windows

package system

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsElevated reports whether the current process is running with
// administrator privileges. TUN mode needs this: creating a system-level
// virtual network adapter (and adding routes for it) isn't allowed for a
// standard user token.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RelaunchElevated re-launches the current executable with a UAC prompt
// and returns immediately -- it does not wait for or track the new
// process. The caller is expected to exit shortly after a successful
// call. Returns an error if the relaunch itself couldn't be started,
// which includes the user dismissing the UAC prompt.
func RelaunchElevated() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exePath)
	dir, _ := syscall.UTF16PtrFromString(cwd)

	const swNormal = 1
	ret, _, _ := shellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, uintptr(unsafe.Pointer(dir)), swNormal)
	// ShellExecute returns a value > 32 on success; anything else
	// (including the user cancelling the UAC prompt) is <= 32.
	if ret <= 32 {
		return fmt.Errorf("could not start an elevated instance (code %d) -- the UAC prompt may have been cancelled", ret)
	}
	return nil
}
