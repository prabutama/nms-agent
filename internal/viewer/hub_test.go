package viewer

import (
	"testing"

	"nms-agent/internal/models"
)

func makeNum(v float64) *float64 { return &v }
func makeStr(v string) *string   { return &v }

func TestHub_Update_MergesMultipleDevices(t *testing.T) {
	h := NewHub("tui")

	h.Update([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "icmp.reachable", ValueType: "number", ValueNumber: makeNum(1)},
	})

	h.Update([]models.Telemetry{
		{DeviceID: "dev-b", Metric: "icmp.reachable", ValueType: "number", ValueNumber: makeNum(0)},
	})

	snap := h.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 merged entries, got %d", len(snap))
	}
}

func TestHub_Update_ReplaceSameKey(t *testing.T) {
	h := NewHub("tui")

	h.Update([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "icmp.reachable", ValueType: "number", ValueNumber: makeNum(1)},
	})

	h.Update([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "icmp.reachable", ValueType: "number", ValueNumber: makeNum(0)},
	})

	snap := h.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 entry after replace, got %d", len(snap))
	}
	if snap[0].ValueNumber == nil || *snap[0].ValueNumber != 0 {
		t.Fatalf("expected value 0 after replace, got %v", snap[0].ValueNumber)
	}
}

func TestHub_Update_DifferentIfIndex(t *testing.T) {
	h := NewHub("tui")

	h.Update([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "snmp.if.rx_bps", ValueType: "number", ValueNumber: makeNum(100), Tags: map[string]string{"ifIndex": "1"}},
	})

	h.Update([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "snmp.if.rx_bps", ValueType: "number", ValueNumber: makeNum(200), Tags: map[string]string{"ifIndex": "2"}},
	})

	snap := h.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries for different ifIndex, got %d", len(snap))
	}
}
