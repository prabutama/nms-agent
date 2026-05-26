package processors

import (
	"context"
	"testing"
	"time"

	"nms-agent/internal/models"
)

func TestPreprocessThresholdProcessor_AddsThresholdTags(t *testing.T) {
	rules := []models.ThresholdRule{{
		Metric:   "icmp.latency_ms",
		Operator: ">",
		Warning:  floatPtr(50),
		Critical: floatPtr(100),
		Tags:     map[string]string{"source": "icmp"},
	}}

	p := PreprocessThresholdProcessor{Rules: rules}
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "icmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric": "icmp.latency_ms",
			"value":  120.0,
			"unit":   "ms",
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(telemetry) != 1 {
		t.Fatalf("expected 1 telemetry, got %d", len(telemetry))
	}
	if telemetry[0].Tags["threshold.status"] != "critical" {
		t.Fatalf("expected threshold.status=critical, got %q", telemetry[0].Tags["threshold.status"])
	}
	if telemetry[0].Tags["threshold.matched"] != "true" {
		t.Fatalf("expected threshold.matched=true")
	}
}

func TestPreprocessThresholdProcessor_TagsWildcardMatch(t *testing.T) {
	rules := []models.ThresholdRule{{
		Metric:   "snmp.if.oper_status",
		Operator: "==",
		Warning:  floatPtr(2),
		Tags:     map[string]string{"ifIndex": "*"},
	}}

	p := PreprocessThresholdProcessor{Rules: rules}
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.oper_status",
			"value":  2.0,
			"tags":   map[string]string{"ifIndex": "5"},
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if telemetry[0].Tags["threshold.status"] != "warning" {
		t.Fatalf("expected threshold.status=warning, got %q", telemetry[0].Tags["threshold.status"])
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
