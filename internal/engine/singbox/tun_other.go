//go:build !windows

package singbox

func prepareTUN() error {
	return nil
}

func missingTUNDLL() bool {
	return false
}
