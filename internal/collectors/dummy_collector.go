package collectors

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"time"

	"nms-agent/internal/models"
)

// DummyCollector generates deterministic samples for demo/testing.
// It performs no network calls.
type DummyCollector struct {
	DeviceID  string
	DeviceIDs []string
}

func (c DummyCollector) Collect(context.Context) ([]models.RawSample, error) {
	deviceIDs := c.DeviceIDs
	if len(deviceIDs) == 0 && c.DeviceID != "" {
		deviceIDs = []string{c.DeviceID}
	}
	if len(deviceIDs) == 0 {
		deviceIDs = []string{"dummy-1"}
	}

	now := time.Now().UTC()
	out := make([]models.RawSample, 0, len(deviceIDs)*10)
	for _, deviceID := range deviceIDs {
		seed := float64(dummySeed(deviceID))
		phase := float64(now.Unix()%300) / 300 * 2 * math.Pi
		baseLoad := 25.0
		if strings.Contains(deviceID, "surabaya") {
			baseLoad = 58
		} else if strings.Contains(deviceID, "makassar") {
			baseLoad = 76
		}
		load := clamp(baseLoad+(seed*17)+18*math.Sin(phase+seed), 5, 98)
		latency := clamp(8+(seed*3)+4*math.Sin(phase), 2, 80)
		loss := 0.0
		if strings.Contains(deviceID, "makassar") {
			loss = clamp(5+4*math.Sin(phase+seed), 0, 15)
		}
		out = append(out, []models.RawSample{
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
					"metric": "icmp.latency_ms", "value_type": "number", "value_number": latency,
					"unit": "ms",
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "icmp.jitter_ms", "value_type": "number", "value_number": clamp(1+2*math.Abs(math.Sin(phase+seed)), 0, 10),
					"unit": "ms",
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "icmp.packet_loss_pct", "value_type": "number", "value_number": loss,
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
					"metric": "snmp.if.rx_bps", "value_type": "number", "value_number": 1250000 + 400000*math.Sin(phase+seed),
					"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.tx_bps", "value_type": "number", "value_number": 850000 + 250000*math.Sin(phase+seed/2),
					"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.utilization_pct", "value_type": "number", "value_number": clamp(load, 0, 100),
					"tags": map[string]string{"ifIndex": "1", "ifName": "eth0"},
				},
			},
			// Host resources
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.host.cpu.load_pct", "value_type": "number", "value_number": load,
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.host.memory.size_kb", "value_type": "number", "value_number": 8234567.0,
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{"metric": "snmp.host.memory.used_pct", "value_type": "number", "value_number": clamp(load*0.82, 0, 100), "unit": "pct"},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{"metric": "snmp.host.storage.idx36.used_pct", "value_type": "number", "value_number": clamp(48+load*0.35, 0, 98), "unit": "pct", "tags": map[string]string{"ifIndex": "36"}},
			},
		}...)
	}
	return out, nil
}

func dummySeed(deviceID string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(deviceID))
	return float64(h.Sum32()%17) / 10
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}
