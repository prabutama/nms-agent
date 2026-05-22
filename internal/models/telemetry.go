package models

import "time"

// RawSample is unnormalized data as collected from devices.
// It intentionally stays generic and platform-agnostic.
type RawSample struct {
	DeviceID string
	Source   string // e.g. "snmp", "icmp" (collector-defined)
	TS       time.Time
	Fields   map[string]any
}

// Telemetry is the canonical, normalized record persisted to the local queue.
type Telemetry struct {
	DeviceID string
	Metric   string
	TS       time.Time
	Value    float64
	Tags     map[string]string
}
