package collectors

import (
	"context"
	"time"

	"nms-agent/internal/models"
)

// DummyCollector generates deterministic samples for demo/testing.
// It performs no network calls.
type DummyCollector struct {
	DeviceID string
}

func (c DummyCollector) Collect(context.Context) ([]models.RawSample, error) {
	deviceID := c.DeviceID
	if deviceID == "" {
		deviceID = "dummy-1"
	}

	now := time.Now().UTC()
	return []models.RawSample{
		// Device health
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "icmp.reachable", "value_type": "number", "value_number": 1.0,
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "icmp.latency_ms", "value_type": "number", "value_number": 12.3,
				"unit": "ms",
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "icmp.jitter_ms", "value_type": "number", "value_number": 1.7,
				"unit": "ms",
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "icmp.packet_loss_pct", "value_type": "number", "value_number": 0.0,
				"unit": "pct",
			},
		},
		// Interface metrics
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.if.name", "value_type": "string", "value_string": "eth0",
				"tags": map[string]string{"ifIndex": "1"},
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.if.rx_bps", "value_type": "number", "value_number": 1250000.0,
				"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.if.tx_bps", "value_type": "number", "value_number": 850000.0,
				"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.if.utilization_pct", "value_type": "number", "value_number": 45.5,
				"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
			},
		},
		// Host resources
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.host.cpu.load_pct", "value_type": "number", "value_number": 23.4,
			},
		},
		{
			DeviceID: deviceID, Source: "dummy", TS: now,
			Fields: map[string]any{
				"metric": "snmp.host.memory.size_kb", "value_type": "number", "value_number": 8234567.0,
			},
		},
	}, nil
}
