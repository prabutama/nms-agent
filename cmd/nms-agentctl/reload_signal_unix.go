//go:build !windows

package main

import (
	"fmt"
	"syscall"
)

func sendReloadSignal(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("send SIGHUP to pid %d: %w", pid, err)
	}
	return nil
}
