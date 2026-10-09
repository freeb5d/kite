//go:build !windows

package update

import "errors"

func installed(string) bool { return false }

func runInstaller(string, string) error { return errors.New("installer updates are Windows-only") }
