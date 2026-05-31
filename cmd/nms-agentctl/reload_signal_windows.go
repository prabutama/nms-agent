//go:build windows

package main

import "errors"

func sendReloadSignal(pid int) error {
	_ = pid
	return errors.New("reload is not supported on Windows; run nms-agentctl inside WSL/Linux or pass reload through service manager")
}
