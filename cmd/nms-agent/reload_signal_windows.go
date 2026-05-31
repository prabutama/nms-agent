//go:build windows

package main

import "os"

func reloadSignals() []os.Signal {
	// Windows doesn't support SIGHUP in the same way.
	return nil
}
