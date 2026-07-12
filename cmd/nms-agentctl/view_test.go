package main

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/models"
)

func TestRenderSummaryFromState_ShowsMetricCounts(t *testing.T) {
	st := adapters.NewStateFromTelemetry([]models.Telemetry{
		numberTelemetry("router-1", "icmp.reachable", 1),
		numberTelemetry("router-1", "icmp.latency_ms", 2.5),
		numberTelemetry("switch-1", "icmp.reachable", 1),
	})
	st.LastSeen = time.Date(2026, 7, 12, 10, 15, 20, 0, time.UTC)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	lines := renderSummaryFromState("thingsboard", st, time.UTC, false, 0)

	_ = w.Close()
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	output := string(out)

	if lines != strings.Count(output, "\n") {
		t.Fatalf("line count = %d, want %d", lines, strings.Count(output, "\n"))
	}
	for _, want := range []string{
		"Metrics: total=3",
		"METRICS",
		"router-1",
		"switch-1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q\n%s", want, output)
		}
	}
	if !strings.Contains(output, "router-1") || !strings.Contains(output, "2.5 ms") || !strings.Contains(output, "        2        0") {
		t.Fatalf("output missing router metric count\n%s", output)
	}
	if !strings.Contains(output, "switch-1") || !strings.Contains(output, "        1        0") {
		t.Fatalf("output missing switch metric count\n%s", output)
	}
}

func numberTelemetry(deviceID, metric string, value float64) models.Telemetry {
	v := value
	return models.Telemetry{
		DeviceID:    deviceID,
		Metric:      metric,
		TS:          time.Date(2026, 7, 12, 10, 15, 20, 0, time.UTC),
		ValueType:   "number",
		ValueNumber: &v,
	}
}
