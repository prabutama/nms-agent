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
	out := make([]models.RawSample, 0, len(deviceIDs)*18)
	for _, deviceID := range deviceIDs {
		seed := float64(dummySeed(deviceID))
		phase := float64(now.Unix()%300) / 300 * 2 * math.Pi
		baseLoad := 25.0
		if strings.Contains(deviceID, "surabaya") {
			baseLoad = 46
		} else if strings.Contains(deviceID, "makassar") {
			baseLoad = 58
		}
		role := demoDeviceRole(deviceID)
		roleLoadBias := map[string]float64{"router": 5, "firewall": 7, "switch": 4, "server": 10, "ap": 6}[role]
		load := baseLoad + roleLoadBias + (seed * 9) + 12*math.Sin(phase+seed)
		if strings.Contains(deviceID, "makassar") && (role == "router" || role == "switch") {
			load += 14
		}
		if strings.Contains(deviceID, "surabaya") && (role == "server" || role == "firewall") {
			load += 8
		}
		load = clamp(load, 5, 96)
		latency := clamp(8+(seed*3)+4*math.Sin(phase), 2, 80)
		if strings.Contains(deviceID, "surabaya") {
			latency = clamp(latency+28+10*math.Abs(math.Sin(phase+seed)), 2, 120)
		} else if strings.Contains(deviceID, "makassar") {
			latency = clamp(latency+42+18*math.Abs(math.Sin(phase+seed)), 2, 160)
		}
		loss := 0.0
		if strings.Contains(deviceID, "makassar") {
			loss = 2.5 + 3.5*math.Sin(phase+seed)
			if role == "router" || role == "switch" {
				loss += 2.5
			}
			loss = clamp(loss, 0, 11)
		} else if strings.Contains(deviceID, "surabaya") && (role == "firewall" || role == "switch") {
			loss = clamp(0.8+1.4*math.Abs(math.Sin(phase+seed)), 0, 4)
		}
		ifaceName := "eth0"
		if role == "switch" {
			ifaceName = "uplink-1"
		} else if role == "ap" {
			ifaceName = "wlan-uplink"
		} else if role == "firewall" {
			ifaceName = "wan"
		}
		throughputBias := map[string]float64{"router": 1.8, "firewall": 1.5, "switch": 2.4, "server": 1.2, "ap": 0.9}[role]
		rxBps := clamp((900000+450000*math.Abs(math.Sin(phase+seed)))*throughputBias, 100000, 9000000)
		txBps := clamp((650000+320000*math.Abs(math.Cos(phase+seed/2)))*throughputBias, 80000, 7000000)
		ifUtil := clamp(load*0.72+loss*2+seed*4, 1, 98)
		storageUsed := clamp(42+load*0.38, 8, 96)
		if role == "server" {
			storageUsed = clamp(storageUsed+12, 8, 98)
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
					"metric": "snmp.if.idx1.name", "value_type": "string", "value_string": ifaceName,
					"tags": map[string]string{"ifIndex": "1"},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.rx_bps", "value_type": "number", "value_number": rxBps,
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.tx_bps", "value_type": "number", "value_number": txBps,
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.utilization_pct", "value_type": "number", "value_number": ifUtil,
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.speed_bps", "value_type": "number", "value_number": 1000000000.0,
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.oper_status", "value_type": "number", "value_number": 1.0,
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.in_errors", "value_type": "number", "value_number": clamp(loss*12+seed*2, 0, 220),
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
				},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{
					"metric": "snmp.if.idx1.out_errors", "value_type": "number", "value_number": clamp(loss*8+seed, 0, 180),
					"tags": map[string]string{"ifIndex": "1", "ifName": ifaceName},
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
				Fields: map[string]any{"metric": "snmp.host.storage.idx36.description", "value_type": "string", "value_string": "/", "tags": map[string]string{"ifIndex": "36"}},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{"metric": "snmp.host.storage.idx36.used_pct", "value_type": "number", "value_number": storageUsed, "unit": "pct", "tags": map[string]string{"ifIndex": "36"}},
			},
			{
				DeviceID: deviceID, Source: "dummy", TS: now,
				Fields: map[string]any{"metric": "snmp.host.uptime_seconds", "value_type": "number", "value_number": 86400.0*(7+seed*3) + float64(now.Unix()%86400)},
			},
		}...)
	}
	return out, nil
}

func demoDeviceRole(deviceID string) string {
	switch {
	case strings.Contains(deviceID, "router"):
		return "router"
	case strings.Contains(deviceID, "firewall"):
		return "firewall"
	case strings.Contains(deviceID, "switch"):
		return "switch"
	case strings.Contains(deviceID, "server"):
		return "server"
	case strings.Contains(deviceID, "ap-"):
		return "ap"
	default:
		return "device"
	}
}

func dummySeed(deviceID string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(deviceID))
	return float64(h.Sum32()%17) / 10
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}
