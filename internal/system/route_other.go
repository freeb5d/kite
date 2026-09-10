//go:build !windows

package system

import "fmt"

func AddExceptionRoute(ip string) error {
	return fmt.Errorf("TUN mode is not implemented on this platform yet")
}

func RemoveExceptionRoute(ip string) error {
	return nil
}
