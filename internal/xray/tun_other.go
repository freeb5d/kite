//go:build !windows

package xray

func prepareTUN() error {
	return nil
}

func missingTUNDLL() bool {
	return false
}
