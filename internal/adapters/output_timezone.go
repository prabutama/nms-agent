package adapters

import (
	"sync"
	"time"
)

var (
	outputLocMu sync.RWMutex
	outputLoc   = time.UTC
)

// SetOutputLocation configures the presentation timezone used by adapters.
// This must be set once at startup from config. Default is UTC.
func SetOutputLocation(loc *time.Location) {
	if loc == nil {
		loc = time.UTC
	}
	outputLocMu.Lock()
	outputLoc = loc
	outputLocMu.Unlock()
}

func getOutputLocation() *time.Location {
	outputLocMu.RLock()
	defer outputLocMu.RUnlock()
	return outputLoc
}
