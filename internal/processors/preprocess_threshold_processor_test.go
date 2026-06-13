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

func TestPreprocessThresholdProcessor_DerivesMemoryUsage(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.host.memory.size_kb",
			"value_type":   "number",
			"value_number": 524288.0,
			"tags":         map[string]string{"source": "snmp"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.host.memory.free_kb",
			"value_type":   "number",
			"value_number": 340272.0,
			"tags":         map[string]string{"source": "snmp"},
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry, "snmp.host.memory.used_kb") {
		t.Fatalf("expected used_kb derived metric")
	}
	if !hasMetric(telemetry, "snmp.host.memory.used_pct") {
		t.Fatalf("expected used_pct derived metric")
	}
	if got, ok := metricValue(telemetry, "snmp.host.memory.used_kb"); !ok || got != 184016 {
		t.Fatalf("expected used_kb=184016, got %v ok=%v", got, ok)
	}
	if got, ok := metricValue(telemetry, "snmp.host.memory.used_pct"); !ok || got != 35.1 {
		t.Fatalf("expected used_pct=35.1, got %v ok=%v", got, ok)
	}
}

func TestPreprocessThresholdProcessor_RoundsMillisecondsToTwoDecimals(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "icmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "icmp.latency_ms",
			"value_type":   "number",
			"value_number": 0.3175,
			"unit":         "ms",
		},
	}, {
		DeviceID: "d1",
		Source:   "icmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "icmp.jitter_ms",
			"value_type":   "number",
			"value_number": 0.07699999999999996,
			"unit":         "ms",
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got, ok := metricValue(telemetry, "icmp.latency_ms"); !ok || got != 0.32 {
		t.Fatalf("expected latency_ms=0.32, got %v ok=%v", got, ok)
	}
	if got, ok := metricValue(telemetry, "icmp.jitter_ms"); !ok || got != 0.08 {
		t.Fatalf("expected jitter_ms=0.08, got %v ok=%v", got, ok)
	}
}

func TestPreprocessThresholdProcessor_DerivesStorageMetrics(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	ts := time.Unix(10, 0).UTC()
	raw := []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       ts,
		Fields: map[string]any{
			"metric":       "snmp.host.storage.allocation_units",
			"value_type":   "number",
			"value_number": 1024.0,
			"tags":         map[string]string{"ifIndex": "65536", "source": "snmp"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       ts,
		Fields: map[string]any{
			"metric":       "snmp.host.storage.size_units",
			"value_type":   "number",
			"value_number": 524288.0,
			"tags":         map[string]string{"ifIndex": "65536", "source": "snmp"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       ts,
		Fields: map[string]any{
			"metric":       "snmp.host.storage.used_units",
			"value_type":   "number",
			"value_number": 222144.0,
			"tags":         map[string]string{"ifIndex": "65536", "source": "snmp"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       ts,
		Fields: map[string]any{
			"metric":       "snmp.host.storage.description",
			"value_type":   "string",
			"value_string": "/",
			"tags":         map[string]string{"ifIndex": "65536", "source": "snmp"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       ts,
		Fields: map[string]any{
			"metric":       "snmp.host.storage.type",
			"value_type":   "string",
			"value_string": "1.3.6.1.2.1.25.2.1.4",
			"tags":         map[string]string{"ifIndex": "65536", "source": "snmp"},
		},
	}}

	telemetry, err := p.Normalize(context.Background(), raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got, ok := metricValue(telemetry, "snmp.host.storage.total_bytes"); !ok || got != 536870912 {
		t.Fatalf("expected total_bytes=536870912, got %v ok=%v", got, ok)
	}
	if got, ok := metricValue(telemetry, "snmp.host.storage.used_bytes"); !ok || got != 227475456 {
		t.Fatalf("expected used_bytes=227475456, got %v ok=%v", got, ok)
	}
	if got, ok := metricValue(telemetry, "snmp.host.storage.free_bytes"); !ok || got != 309395456 {
		t.Fatalf("expected free_bytes=309395456, got %v ok=%v", got, ok)
	}
	if got, ok := metricValue(telemetry, "snmp.host.storage.used_pct"); !ok || got != 42.37 {
		t.Fatalf("expected used_pct=42.37, got %v ok=%v", got, ok)
	}
	item := findMetric(telemetry, "snmp.host.storage.used_pct")
	if item == nil {
		t.Fatalf("expected used_pct metric item")
	}
	if item.Tags["storage_description"] != "/" {
		t.Fatalf("expected storage_description=/, got %q", item.Tags["storage_description"])
	}
	if item.Tags["storage_type"] != "1.3.6.1.2.1.25.2.1.4" {
		t.Fatalf("expected storage_type tag, got %q", item.Tags["storage_type"])
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
	if len(telemetry) == 0 {
		t.Fatalf("expected telemetry items")
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

func TestPreprocessThresholdProcessor_PrefersHighCapacityCounters(t *testing.T) {
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
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.hc_in_octets",
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
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1500.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.hc_in_octets",
			"value_type":   "number",
			"value_number": 2600.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "snmp.if.rx_bps")
	if !ok {
		t.Fatalf("expected rx_bps from high-capacity counters")
	}
	if val != 480.0 {
		t.Fatalf("expected rx_bps=480, got %v", val)
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

func TestPreprocessThresholdProcessor_UsesHighSpeedMbpsFallback(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// First sample: establish baseline.
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

	// Provide high_speed_mbps (but no speed_bps).
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.high_speed_mbps",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	// Counter sample at ts=30 vs baseline at ts=10: dt=20s, delta=2000 octets.
	// bps = (2000*8)/20 = 800 bps; speed = 1000 Mbps * 1e6 = 1e9 bps;
	// utilization = (800/1e9)*100 = 0.00008%
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(30, 0).UTC(),
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
	if !hasMetric(telemetry, "snmp.if.rx_utilization_pct") {
		t.Fatalf("expected utilization from high_speed_mbps fallback")
	}
	val, ok := metricValue(telemetry, "snmp.if.rx_utilization_pct")
	if !ok {
		t.Fatalf("expected numeric utilization value")
	}
	if val != 0 {
		t.Fatalf("expected rounded utilization 0, got %v", val)
	}
}

func TestPreprocessThresholdProcessor_PrefersSpeedBpsOverHighSpeedMbps(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// First sample: establish baseline.
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

	// Provide both speed_bps (100 Mbps) and high_speed_mbps (1000 Mbps) in the same cycle.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.speed_bps",
			"value_type":   "number",
			"value_number": 100_000_000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}, {
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.high_speed_mbps",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})

	// Second sample: utilization must use 100 Mbps from speed_bps, not 1000 Mbps from high_speed_mbps.
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(30, 0).UTC(),
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
	if !hasMetric(telemetry, "snmp.if.rx_utilization_pct") {
		t.Fatalf("expected utilization")
	}
	// baseline ts=10, sample ts=30 => dt=20s; delta=2000 octets => bps=800; speed=100_000_000 => util=0.0008%
	val, ok := metricValue(telemetry, "snmp.if.rx_utilization_pct")
	if !ok {
		t.Fatalf("expected numeric utilization value")
	}
	if val != 0 {
		t.Fatalf("expected rounded utilization 0, got %v", val)
	}
}

func TestPreprocessThresholdProcessor_FiltersNonPhysicalInterfaces(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// First, register ifType=1 (other) for ifIndex=1.
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.type",
			"value_type":   "number",
			"value_number": 1.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	// Now send an interface metric for ifIndex=1. It should be filtered out.
	telemetry, err = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.oper_status",
			"value_type":   "number",
			"value_number": 1.0,
			"tags":         map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected non-physical interface metric to be filtered out")
	}
}

func TestPreprocessThresholdProcessor_KeepsPhysicalInterfaces(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// Register ifType=6 (ethernet) for ifIndex=2.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.type",
			"value_type":   "number",
			"value_number": 6.0,
			"tags":         map[string]string{"ifIndex": "2"},
		},
	}})

	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.oper_status",
			"value_type":   "number",
			"value_number": 1.0,
			"tags":         map[string]string{"ifIndex": "2"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected physical interface metric to be kept")
	}
}

func TestPreprocessThresholdProcessor_KeepsWhenIfTypeUnknown(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// No ifType known for ifIndex=3 — should be kept by default.
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.oper_status",
			"value_type":   "number",
			"value_number": 1.0,
			"tags":         map[string]string{"ifIndex": "3"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected interface metric with unknown ifType to be kept")
	}
}

func TestPreprocessThresholdProcessor_FiltersDerivedMetricsForNonPhysical(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// Register ifType=1 (non-physical) for ifIndex=5.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.type",
			"value_type":   "number",
			"value_number": 1.0,
			"tags":         map[string]string{"ifIndex": "5"},
		},
	}})
	// Baseline counter.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(10, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 1000.0,
			"tags":         map[string]string{"ifIndex": "5"},
		},
	}})
	// Second sample; derived metrics should NOT appear for non-physical ifIndex.
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Unix(20, 0).UTC(),
		Fields: map[string]any{
			"metric":       "snmp.if.in_octets",
			"value_type":   "number",
			"value_number": 2000.0,
			"tags":         map[string]string{"ifIndex": "5"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.rx_bps") {
		t.Fatalf("did not expect rx_bps for non-physical interface")
	}
	if hasMetric(telemetry, "snmp.if.rx_utilization_pct") {
		t.Fatalf("did not expect rx_utilization_pct for non-physical interface")
	}
}

func TestPreprocessThresholdProcessor_FiltersVirtualNamePatterns_Proxmox(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// Register ifName=vmbr0 for ifIndex=10 via cache.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.name", "value_type": "string", "value_string": "vmbr0",
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected vmbr* interface to be filtered out")
	}

	// tap, veth, fwbr, fwpr, fwln patterns.
	for _, name := range []string{"tap0", "vethabc", "fwbr123", "fwpr456", "fwln789"} {
		_, _ = p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d2", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.name", "value_type": "string", "value_string": name,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		telemetry, _ = p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d2", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		if hasMetric(telemetry, "snmp.if.oper_status") {
			t.Fatalf("expected %s interface to be filtered out", name)
		}
	}
}

func TestPreprocessThresholdProcessor_FiltersVirtualNamePatterns_DockerK8s(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	for _, name := range []string{"docker0", "br-abc123", "cni0", "flannel.1", "cali123"} {
		_, _ = p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.name", "value_type": "string", "value_string": name,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		telemetry, _ := p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		if hasMetric(telemetry, "snmp.if.oper_status") {
			t.Fatalf("expected %s interface to be filtered out", name)
		}
	}
}

func TestPreprocessThresholdProcessor_KeepsPhysicalNames(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	for _, name := range []string{"eno1", "eth0", "ens18", "enp3s0", "wlp58s0", "enx001122334455"} {
		_, _ = p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.name", "value_type": "string", "value_string": name,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		telemetry, _ := p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
				"tags": map[string]string{"ifIndex": "1"},
			},
		}})
		if !hasMetric(telemetry, "snmp.if.oper_status") {
			t.Fatalf("expected %s interface to be kept", name)
		}
	}
}

func TestPreprocessThresholdProcessor_ConnectorPresentOverrides(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// Register ifType=1, ifName unknown, but connectorPresent=true => keep.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.type", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}, {
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.connector_present", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected interface with connectorPresent=true to be kept")
	}
}

func TestPreprocessThresholdProcessor_ConnectorFalseDrops(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// ifType=6 but connectorPresent=false => drop.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.type", "value_type": "number", "value_number": 6.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}, {
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.connector_present", "value_type": "number", "value_number": 2.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "10"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected interface with connectorPresent=false to be dropped")
	}
}

func TestPreprocessThresholdProcessor_KeepsWifiIfType(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	// ifType=71 (wifi) should be kept.
	_, _ = p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.type", "value_type": "number", "value_number": 71.0,
			"tags": map[string]string{"ifIndex": "3"},
		},
	}})
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.oper_status", "value_type": "number", "value_number": 1.0,
			"tags": map[string]string{"ifIndex": "3"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !hasMetric(telemetry, "snmp.if.oper_status") {
		t.Fatalf("expected wifi interface (ifType=71) to be kept")
	}
}

func TestNormalizeMetrics_ClampsPercentTo100(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.rx_utilization_pct", "value_type": "number", "value_number": 150.0,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "snmp.if.rx_utilization_pct")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 100.0 {
		t.Fatalf("expected 100, got %v", val)
	}
	if telemetry[0].Tags["unit"] != "pct" {
		t.Fatalf("expected unit=pct, got %q", telemetry[0].Tags["unit"])
	}
}

func TestNormalizeMetrics_ClampsPercentAbove0(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "some_metric_pct", "value_type": "number", "value_number": -5.0,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "some_metric_pct")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %v", val)
	}
}

func TestNormalizeMetrics_ClampsMsToNonNegative(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "icmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "icmp.latency_ms", "value_type": "number", "value_number": -10.0,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "icmp.latency_ms")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %v", val)
	}
	if telemetry[0].Tags["unit"] != "ms" {
		t.Fatalf("expected unit=ms, got %q", telemetry[0].Tags["unit"])
	}
}

func TestNormalizeMetrics_ClampsSecondsToNonNegative(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.uptime_seconds", "value_type": "number", "value_number": -1.0,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "snmp.uptime_seconds")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %v", val)
	}
	if telemetry[0].Tags["unit"] != "s" {
		t.Fatalf("expected unit=s, got %q", telemetry[0].Tags["unit"])
	}
}

func TestNormalizeMetrics_ClampsBpsToNonNegative(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.rx_bps", "value_type": "number", "value_number": -500.0,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "snmp.if.rx_bps")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %v", val)
	}
	if telemetry[0].Tags["unit"] != "bps" {
		t.Fatalf("expected unit=bps, got %q", telemetry[0].Tags["unit"])
	}
}

func TestNormalizeMetrics_ReachableNormalisesTo0Or1(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 0}, {0.0, 0}, {-1, 0}, {1, 1}, {2.5, 1},
	}
	for _, tt := range tests {
		telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
			DeviceID: "d1", Source: "icmp", TS: time.Now().UTC(),
			Fields: map[string]any{
				"metric": "icmp.reachable", "value_type": "number", "value_number": tt.input,
			},
		}})
		if err != nil {
			t.Fatalf("Normalize(%v): %v", tt.input, err)
		}
		val, ok := metricValue(telemetry, "icmp.reachable")
		if !ok {
			t.Fatalf("expected metric for input %v", tt.input)
		}
		if val != tt.want {
			t.Fatalf("input %v: expected %v, got %v", tt.input, tt.want, val)
		}
	}
}

func TestNormalizeMetrics_PreservesExistingUnitTag(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.tx_utilization_pct", "value_type": "number", "value_number": 50.0,
			"unit": "pct",
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if telemetry[0].Tags["unit"] != "pct" {
		t.Fatalf("expected unit=pct preserved, got %q", telemetry[0].Tags["unit"])
	}
}

func TestNormalizeMetrics_DoesNotAffectNonNumber(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.system.description", "value_type": "string", "value_string": "RouterOS",
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(telemetry) != 1 {
		t.Fatalf("expected 1 telemetry, got %d", len(telemetry))
	}
	if telemetry[0].ValueString == nil || *telemetry[0].ValueString != "RouterOS" {
		t.Fatalf("string value should be unchanged")
	}
}

func TestNormalizeMetrics_ThresholdStillEvaluatedAfterNormalization(t *testing.T) {
	p := PreprocessThresholdProcessor{Rules: []models.ThresholdRule{{
		Metric:   "snmp.if.rx_utilization_pct",
		Operator: ">",
		Warning:  floatPtr(80),
		Critical: floatPtr(95),
		Tags:     map[string]string{"ifIndex": "1"},
	}}}
	// Clamped 98% should trigger critical.
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "snmp.if.rx_utilization_pct", "value_type": "number", "value_number": 98.0,
			"tags": map[string]string{"ifIndex": "1"},
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if telemetry[0].Tags["threshold.status"] != "critical" {
		t.Fatalf("expected critical, got %q", telemetry[0].Tags["threshold.status"])
	}
}

func TestNormalizeMetrics_PassthroughDoesNotClampPercentageInRange(t *testing.T) {
	p := PreprocessThresholdProcessor{}
	telemetry, err := p.Normalize(context.Background(), []models.RawSample{{
		DeviceID: "d1", Source: "snmp", TS: time.Now().UTC(),
		Fields: map[string]any{
			"metric": "icmp.packet_loss_pct", "value_type": "number", "value_number": 3.5,
		},
	}})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	val, ok := metricValue(telemetry, "icmp.packet_loss_pct")
	if !ok {
		t.Fatalf("expected metric")
	}
	if val != 3.5 {
		t.Fatalf("expected 3.5 unchanged, got %v", val)
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

func metricValue(items []models.Telemetry, metric string) (float64, bool) {
	for _, t := range items {
		if t.Metric != metric {
			continue
		}
		if t.ValueType != "number" || t.ValueNumber == nil {
			return 0, false
		}
		return *t.ValueNumber, true
	}
	return 0, false
}

func findMetric(items []models.Telemetry, metric string) *models.Telemetry {
	for i := range items {
		if items[i].Metric == metric {
			return &items[i]
		}
	}
	return nil
}

func floatPtr(v float64) *float64 {
	return &v
}
