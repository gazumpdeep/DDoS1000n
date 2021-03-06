//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package ota

import (
	"fmt"
	"runtime"
)

func Restart(extraArgs ...string) error {
	return fmt.Errorf("restart on the %s system is not available", runtime.GOOS)
}
