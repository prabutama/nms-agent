package base

import (
	"sync"
	"time"
)

var (
	outputLocMu sync.RWMutex
	outputLoc   = time.UTC
)

func SetOutputLocation(loc *time.Location) {
	if loc == nil {
		loc = time.UTC
	}
	outputLocMu.Lock()
	outputLoc = loc
	outputLocMu.Unlock()
}

func GetOutputLocation() *time.Location {
	outputLocMu.RLock()
	defer outputLocMu.RUnlock()
	return outputLoc
}
