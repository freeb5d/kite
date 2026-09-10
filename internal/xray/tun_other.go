//go:build !windows

package xray

func prepareTUN() error {
	return nil
}
