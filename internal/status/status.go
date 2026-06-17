package status

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Runtime holds a point-in-time snapshot of agent runtime state.
type Runtime struct {
	mu *sync.RWMutex

	AdapterActive   string    `json:"adapter_active"`
	PollInterval    string    `json:"poll_interval"`
	DeviceCount     int       `json:"device_count"`
	Uptime          string    `json:"uptime"`
	LastCycleOK     bool      `json:"last_cycle_ok"`
	LastCycleErr    string    `json:"last_cycle_error,omitempty"`
	LastCycleStart  time.Time `json:"last_cycle_start"`
	QueuePending    int       `json:"queue_pending"`
	QueueDeadLetter int       `json:"queue_dead_letter"`
	QueueMaxRetry   int       `json:"queue_max_retry"`
	startedAt       time.Time
}

// NewRuntime creates a Runtime and records the start time.
func NewRuntime() *Runtime {
	return &Runtime{mu: &sync.RWMutex{}, startedAt: time.Now()}
}

// Snapshot returns a copy of the current status without holding the lock.
func (r *Runtime) Snapshot() Runtime {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := Runtime{
		AdapterActive:   r.AdapterActive,
		PollInterval:    r.PollInterval,
		DeviceCount:     r.DeviceCount,
		Uptime:          time.Since(r.startedAt).Truncate(time.Second).String(),
		LastCycleOK:     r.LastCycleOK,
		LastCycleErr:    r.LastCycleErr,
		LastCycleStart:  r.LastCycleStart,
		QueuePending:    r.QueuePending,
		QueueDeadLetter: r.QueueDeadLetter,
		QueueMaxRetry:   r.QueueMaxRetry,
		startedAt:       r.startedAt,
	}
	return s
}

// Update writes a new status snapshot from the current cycle.
func (r *Runtime) Update(adapter string, pollInterval string, deviceCount int, ok bool, cycleErr string, cycleStart time.Time, pending int, deadLetter int, maxRetry int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AdapterActive = adapter
	r.PollInterval = pollInterval
	r.DeviceCount = deviceCount
	r.LastCycleOK = ok
	r.LastCycleErr = cycleErr
	r.LastCycleStart = cycleStart
	r.QueuePending = pending
	r.QueueDeadLetter = deadLetter
	r.QueueMaxRetry = maxRetry
}

// WriteFile atomically writes the status snapshot as JSON to path.
func (r *Runtime) WriteFile(path string) error {
	s := r.Snapshot()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadFile reads a status file written by WriteFile.
func ReadFile(path string) (*Runtime, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Runtime
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
