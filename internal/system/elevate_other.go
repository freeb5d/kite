//go:build !windows

package system

import "fmt"

// IsElevated is only meaningful on Windows (TUN mode's elevation gate);
// treat other platforms as already having whatever privileges they need
// since TUN mode isn't offered there yet.
func IsElevated() bool {
	return true
}

func RelaunchElevated() error {
	return fmt.Errorf("relaunching elevated is only implemented on Windows")
}
