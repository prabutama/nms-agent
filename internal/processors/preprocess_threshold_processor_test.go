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
			"metric":       "icmp.latency_ms",
			"value_type":   "number",
			"value_number": 120.0,
			"unit":         "ms",
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
			"metric":       "snmp.if.oper_status",
			"value_type":   "number",
			"value_number": 2.0,
			"tags":         map[string]string{"ifIndex": "5"},
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

func TestPreprocessThresholdProcessor_BpsRequiresTwoSamples(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	first := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}}
	telemetry, err := p.Normalize(context.Background(), first)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	for _, item := range telemetry {
		if item.Metric == "snmp.if.rx_bps" {
			t.Fatalf("did not expect rx_bps on first sample")
		}
	}

	second := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}}
	telemetry2, err := p.Normalize(context.Background(), second)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry2, "snmp.if.rx_bps") {
		t.Fatalf("expected rx_bps on second sample")
	}
}

func TestPreprocessThresholdProcessor_SkipsNegativeDelta(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.out_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.out_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.tx_bps") {
		t.Fatalf("did not expect tx_bps on negative delta")
	}
}

func TestPreprocessThresholdProcessor_SkipsNonPositiveDeltaSeconds(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.rx_bps") {
		t.Fatalf("did not expect rx_bps when delta_seconds <= 0")
	}
}

func TestPreprocessThresholdProcessor_UtilizationRequiresSpeed(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.rx_utilization_pct") {
		t.Fatalf("did not expect utilization without speed")
	}

	// Provide speed and a new counter sample.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(30, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.speed_bps",
			"value_type":   "number",
			"value_number": 100000000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	telemetry2, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(40, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 3000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry2, "snmp.if.rx_utilization_pct") {
		t.Fatalf("expected utilization when speed exists")
	}
}

func TestPreprocessThresholdProcessor_DerivedMetricsEvaluatedByThresholds(t *testing.T) {
	p := PreprocessThresholdProcessor{Rules: []models.ThresholdRule{{
		Metric:   "snmp.if.rx_utilization_pct",
		Operator: ">",
		Warning:  floatPtr(1),
		Tags:     map[string]string{"ifIndex": "1"},
	}}}

	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.speed_bps",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(30, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetricWithTag(telemetry, "snmp.if.rx_utilization_pct", "threshold.status", "warning") {
		t.Fatalf("expected warning threshold on derived utilization")
	}
}

func TestPreprocessThresholdProcessor_SkipsThresholdForStringValue(t *testing.T) {
	rules := []models.ThresholdRule{{
		Metric:   "snmp.system.description",
		Operator: "==",
		Warning:  floatPtr(1),
	}}

	p := PreprocessThresholdProcessor{Rules: rules}
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.system.description",
			"value_type":   "string",
			"value_string": "RouterOS CHR",
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(telemetry) != 1 {
		t.Fatalf("expected 1 telemetry, got %d", len(telemetry))
	}
	if _, ok := telemetry[0].Tags["threshold.matched"]; ok {
		t.Fatalf("did not expect threshold tags for string value")
	}
}

func hasMetric(items []models.Telemetry, metric string) bool {
	for _, t := range items {
		if t.Metric == metric {
			return true
		}
	}
	return false
}

func hasMetricWithTag(items []models.Telemetry, metric, tagKey, tagVal string) bool {
	for _, t := range items {
		if t.Metric == metric && t.Tags[tagKey] == tagVal {
			return true
		}
	}
	return false
}

func floatPtr(v float64) *float64 {
	return &v
}
