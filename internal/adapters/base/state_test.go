package base

import (
	"testing"

	"nms-agent/internal/models"
)

func TestStateApplyBatch_TracksMetricCountsPerLatestBatch(t *testing.T) {
	st := NewState()

	v1 := 1.0
	v2 := 2.5
	v3 := 0.0
	st.ApplyBatch([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "icmp.reachable", ValueType: "number", ValueNumber: &v1},
		{DeviceID: "dev-a", Metric: "icmp.latency_ms", ValueType: "number", ValueNumber: &v2},
		{DeviceID: "dev-b", Metric: "icmp.reachable", ValueType: "number", ValueNumber: &v3},
	})

	if got := st.DeviceMetricCount("dev-a"); got != 2 {
		t.Fatalf("dev-a metric count = %d, want 2", got)
	}
	if got := st.DeviceMetricCount("dev-b"); got != 1 {
		t.Fatalf("dev-b metric count = %d, want 1", got)
	}
	if got := st.TotalMetricCount(); got != 3 {
		t.Fatalf("total metric count = %d, want 3", got)
	}

	st.ApplyBatch([]models.Telemetry{
		{DeviceID: "dev-a", Metric: "icmp.packet_loss_pct", ValueType: "number", ValueNumber: &v3},
	})

	if got := st.DeviceMetricCount("dev-a"); got != 1 {
		t.Fatalf("dev-a metric count after reset = %d, want 1", got)
	}
	if got := st.DeviceMetricCount("dev-b"); got != 0 {
		t.Fatalf("dev-b metric count after reset = %d, want 0", got)
	}
	if got := st.TotalMetricCount(); got != 1 {
		t.Fatalf("total metric count after reset = %d, want 1", got)
	}
}
