package adapters

import (
	"context"
	"testing"
	"time"

	"nms-agent/internal/models"
)

func TestTUIAdapter_SendBatch(t *testing.T) {
	a, err := NewTUIAdapter(map[string]any{"alt_screen": false, "discard_output": true, "disable_renderer": true})
	if err != nil {
		t.Fatalf("NewTUIAdapter: %v", err)
	}
	defer a.Close()

	time.Sleep(10 * time.Millisecond)

	batch := []models.Telemetry{
		{
			DeviceID:    "d1",
			Metric:      "icmp.reachable",
			ValueType:   "number",
			ValueNumber: floatPtr(1),
			TS:          time.Now().UTC(),
		},
		{
			DeviceID:    "d1",
			Metric:      "snmp.if.rx_utilization_pct",
			ValueType:   "number",
			ValueNumber: floatPtr(95),
			TS:          time.Now().UTC(),
			Tags:        map[string]string{"ifIndex": "1", "threshold.status": "critical"},
		},
		{
			DeviceID:    "d1",
			Metric:      "snmp.if.rx_bps",
			ValueType:   "number",
			ValueNumber: floatPtr(1_500_000),
			TS:          time.Now().UTC(),
			Tags:        map[string]string{"ifIndex": "1"},
		},
		{
			DeviceID:    "d1",
			Metric:      "snmp.if.tx_bps",
			ValueType:   "number",
			ValueNumber: floatPtr(800_000),
			TS:          time.Now().UTC(),
			Tags:        map[string]string{"ifIndex": "1"},
		},
	}

	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("SendBatch: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
}

func TestTUIAdapter_CloseWithoutSend(t *testing.T) {
	a, err := NewTUIAdapter(map[string]any{"alt_screen": false, "discard_output": true, "disable_renderer": true})
	if err != nil {
		t.Fatalf("NewTUIAdapter: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	a.Close()
}

func TestTUIAdapter_MultipleBatches(t *testing.T) {
	a, err := NewTUIAdapter(map[string]any{"alt_screen": false, "discard_output": true, "disable_renderer": true})
	if err != nil {
		t.Fatalf("NewTUIAdapter: %v", err)
	}
	defer a.Close()

	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 3; i++ {
		batch := []models.Telemetry{
			{
				DeviceID:    "d1",
				Metric:      "icmp.reachable",
				ValueType:   "number",
				ValueNumber: floatPtr(1),
				TS:          time.Now().UTC(),
			},
		}
		if err := a.SendBatch(context.Background(), batch); err != nil {
			t.Fatalf("SendBatch[%d]: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTUIModel_MemoryFreeLikeKB(t *testing.T) {
	m := newTUIModel(100 * time.Millisecond)
	// Simulate a device snapshot with UCD memory metrics.
	batch := []models.Telemetry{
		{DeviceID: "p1", Metric: "snmp.host.memory.size_kb", ValueType: "number", ValueNumber: floatPtr(19 * 1024 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.memory.free_kb", ValueType: "number", ValueNumber: floatPtr(484 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.memory.available_kb", ValueType: "number", ValueNumber: floatPtr(708 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.memory.shared_kb", ValueType: "number", ValueNumber: floatPtr(22 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.memory.buffer_kb", ValueType: "number", ValueNumber: floatPtr(200 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.memory.cached_kb", ValueType: "number", ValueNumber: floatPtr(394 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.swap.total_kb", ValueType: "number", ValueNumber: floatPtr(8 * 1024 * 1024), TS: time.Now().UTC()},
		{DeviceID: "p1", Metric: "snmp.host.swap.free_kb", ValueType: "number", ValueNumber: floatPtr(2.8 * 1024 * 1024), TS: time.Now().UTC()},
	}
	m.applyBatch(batch)

	totalKB, usedKB, freeKB, sharedKB, buffCacheKB, availKB, swapTotalKB, swapUsedKB, swapFreeKB, ok := m.memoryFreeLikeKB("p1")
	if !ok {
		t.Fatalf("expected ok")
	}
	if totalKB <= 0 || usedKB < 0 {
		t.Fatalf("unexpected totals: total=%v used=%v", totalKB, usedKB)
	}
	if freeKB <= 0 || availKB <= 0 {
		t.Fatalf("unexpected free/avail: free=%v avail=%v", freeKB, availKB)
	}
	if buffCacheKB <= 0 {
		t.Fatalf("expected buff/cache > 0")
	}
	if sharedKB <= 0 {
		t.Fatalf("expected shared > 0")
	}
	if swapTotalKB <= 0 || swapFreeKB <= 0 || swapUsedKB <= 0 {
		t.Fatalf("unexpected swap: total=%v used=%v free=%v", swapTotalKB, swapUsedKB, swapFreeKB)
	}
}

func TestTUIModel_ICMPHealthMetrics(t *testing.T) {
	m := newTUIModel(100 * time.Millisecond)
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "icmp.reachable", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()},
		{DeviceID: "d1", Metric: "icmp.latency_ms", ValueType: "number", ValueNumber: floatPtr(12.3), TS: time.Now().UTC()},
		{DeviceID: "d1", Metric: "icmp.jitter_ms", ValueType: "number", ValueNumber: floatPtr(1.7), TS: time.Now().UTC()},
		{DeviceID: "d1", Metric: "icmp.packet_loss_pct", ValueType: "number", ValueNumber: floatPtr(0), TS: time.Now().UTC()},
	}
	m.applyBatch(batch)

	ds, ok := m.devices["d1"]
	if !ok {
		t.Fatalf("expected device state")
	}
	if ds.Reachable == nil || !*ds.Reachable {
		t.Fatalf("expected reachable true")
	}
	if ds.LatencyMS == nil || *ds.LatencyMS <= 0 {
		t.Fatalf("expected latency set")
	}
	if ds.JitterMS == nil {
		t.Fatalf("expected jitter set")
	}
	if ds.LossPct == nil {
		t.Fatalf("expected loss set")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
