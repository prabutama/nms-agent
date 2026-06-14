package thingsboardmqtt

import (
	"testing"
	"time"

	"nms-agent/internal/models"
)

func TestThingsBoardMQTTAdapter_DirectModeProjectsRouteStringsToAttributes(t *testing.T) {
	c, err := parseConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "mode": "direct", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Mode != "direct" {
		t.Fatalf("expected mode=direct, got %q", c.Mode)
	}
	if c.Topic != "v1/gateway/telemetry" {
		t.Fatalf("expected topic v1/gateway/telemetry, got %q", c.Topic)
	}

	snap := `{"device_id":"r1","ipv4_default":[{"destination":"0.0.0.0/0","next_hop":"10.0.0.1","interface_id":"2","protocol":"kernel","route_type":"local"}],"ipv4_connected":[{"destination":"10.0.0.0/24","next_hop":"0.0.0.0","interface_id":"2","protocol":"kernel","route_type":"local"}],"ipv4_remote":[{"destination":"192.168.1.0/24","next_hop":"10.0.0.2","interface_id":"2","protocol":"ospf","route_type":"remote"}]}`
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	batch := []models.Telemetry{
		{DeviceID: "r1", Metric: "test", ValueType: "number", ValueNumber: floatPtr(1), TS: ts},
		{DeviceID: "r1", Metric: "route.ipv4.snapshot", ValueType: "string", ValueString: strPtr(snap), TS: ts},
	}

	telemetryPayload, attrPayload, err := buildPayloads("direct", batch)
	if err != nil {
		t.Fatalf("buildPayloads: %v", err)
	}

	if val, ok := telemetryPayload["r1"]; ok {
		if len(val) != 1 {
			t.Fatalf("expected 1 timestamp entry, got %d", len(val))
		}
		if _, hasTest := val[0].Values["test"]; !hasTest {
			t.Fatalf("expected test metric in telemetry")
		}
		if _, hasSnap := val[0].Values["route.ipv4.snapshot"]; hasSnap {
			t.Fatalf("route snapshot should NOT be in telemetry values in direct mode")
		}
	} else {
		t.Fatalf("expected device r1 in telemetry payload")
	}

	if attrPayload == nil || len(attrPayload) == 0 {
		t.Fatalf("expected attributes payload for route string metrics")
	}
	r1Attrs, ok := attrPayload["r1"]
	if !ok {
		t.Fatalf("expected r1 in attributes payload")
	}

	attrSnap, ok := r1Attrs["route.ipv4.snapshot"]
	if !ok {
		t.Fatalf("expected route.ipv4.snapshot in attributes")
	}
	if attrSnap.(string) != snap {
		t.Fatalf("snapshot content mismatch")
	}
}

func TestThingsBoardMQTTAdapter_GatewayModeKeepsAllInTelemetry(t *testing.T) {
	c, err := parseConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "mode": "gateway"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Mode != "gateway" {
		t.Fatalf("expected mode=gateway")
	}
	if c.Topic != "nms-agent/thingsboard/telemetry" {
		t.Fatalf("expected topic nms-agent/thingsboard/telemetry, got %q", c.Topic)
	}

	snap := `{"device_id":"r1","ipv4_default":[],"ipv4_connected":[],"ipv4_remote":[]}`
	batch := []models.Telemetry{
		{DeviceID: "r1", Metric: "route.ipv4.snapshot", ValueType: "string", ValueString: strPtr(snap), TS: time.Now().UTC()},
	}

	telemetryPayload, attrPayload, err := buildPayloads("gateway", batch)
	if err != nil {
		t.Fatalf("buildPayloads: %v", err)
	}
	if attrPayload != nil && len(attrPayload) > 0 {
		t.Fatalf("expected no attributes payload in gateway mode")
	}
	if _, ok := telemetryPayload["r1"]; !ok {
		t.Fatalf("expected device r1 in telemetry payload in gateway mode")
	}
}

func TestTBFlattenedInterfaceKey(t *testing.T) {
	tests := []struct {
		name   string
		metric string
		tags   map[string]string
		want   string
		wantOK bool
	}{
		{"no tags", "snmp.if.rx_bps", nil, "", false},
		{"no ifIndex", "snmp.if.rx_bps", map[string]string{"ifName": "eth0"}, "", false},
		{"with ifName", "snmp.if.rx_bps", map[string]string{"ifIndex": "2", "ifName": "eth0"}, "snmp.if.eth0.rx_bps", true},
		{"no ifName", "snmp.if.rx_utilization_pct", map[string]string{"ifIndex": "3"}, "snmp.if.idx3.rx_utilization_pct", true},
		{"non-if metric", "test.metric", map[string]string{"ifIndex": "1"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel := models.Telemetry{Metric: tt.metric, Tags: tt.tags}
			got, ok := flattenedIndexedKey(tel)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("got=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestTBFlattenedInterfaceKey_Sanitization(t *testing.T) {
	got, ok := flattenedIndexedKey(models.Telemetry{Metric: "snmp.if.rx_bps", Tags: map[string]string{"ifIndex": "4", "ifName": "eth 0/1"}})
	if !ok || got != "snmp.if.eth-0-1.rx_bps" {
		t.Fatalf("got=%q, want %q", got, "snmp.if.eth-0-1.rx_bps")
	}
}

func TestTBFlattenedStorageKey(t *testing.T) {
	tests := []struct {
		name   string
		metric string
		tags   map[string]string
		want   string
		wantOK bool
	}{
		{"no tags", "snmp.host.storage.used_bytes", nil, "", false},
		{"no ifIndex", "snmp.host.storage.used_bytes", map[string]string{}, "", false},
		{"valid", "snmp.host.storage.used_bytes", map[string]string{"ifIndex": "5"}, "snmp.host.storage.idx5.used_bytes", true},
		{"non-storage prefix", "snmp.if.rx_bps", map[string]string{"ifIndex": "5"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel := models.Telemetry{Metric: tt.metric, Tags: tt.tags}
			got, ok := flattenedStorageKey(tel)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("got=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestTBBuildThingsBoardPayloads_ValidationErrors(t *testing.T) {
	t.Run("empty device id", func(t *testing.T) {
		_, _, err := buildPayloads("direct", []models.Telemetry{
			{DeviceID: "", Metric: "m", TS: time.Now().UTC()},
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
	t.Run("empty metric", func(t *testing.T) {
		_, _, err := buildPayloads("direct", []models.Telemetry{
			{DeviceID: "d1", Metric: "", TS: time.Now().UTC()},
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestTBBuildThingsBoardPayloads_GroupsByTimestamp(t *testing.T) {
	ts1 := time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC)
	ts2 := time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC)
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "m1", ValueType: "number", ValueNumber: floatPtr(1), TS: ts1},
		{DeviceID: "d1", Metric: "m2", ValueType: "number", ValueNumber: floatPtr(2), TS: ts1},
		{DeviceID: "d1", Metric: "m3", ValueType: "number", ValueNumber: floatPtr(3), TS: ts2},
	}
	payload, _, err := buildPayloads("direct", batch)
	if err != nil {
		t.Fatalf("buildPayloads: %v", err)
	}
	entries, ok := payload["d1"]
	if !ok {
		t.Fatalf("expected d1")
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (2 distinct timestamps), got %d", len(entries))
	}
}

func TestTBRouteSnapshotsFromBatch(t *testing.T) {
	snap := `{"device_id":"r1","ipv4_default":[],"ipv4_connected":[],"ipv4_remote":[]}`
	ts := time.Now().UTC()
	batch := []models.Telemetry{
		{DeviceID: "r1", Metric: "route.ipv4.snapshot", ValueType: "string", ValueString: &snap, TS: ts},
		{DeviceID: "r1", Metric: "temp", ValueType: "number", ValueNumber: floatPtr(1), TS: ts},
	}

	result := routeSnapshotsFromBatch(batch)
	if len(result) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(result))
	}
	if result[0].DeviceID != "r1" {
		t.Fatalf("expected device r1, got %s", result[0].DeviceID)
	}
}

func TestTBSanitizeKeyPart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"eth0", "eth0"},
		{"eth 0/1", "eth-0-1"},
		{"  spaces  ", "spaces"},
		{"UPPER", "upper"},
	}
	for _, tt := range tests {
		got := sanitizeKeyPart(tt.input)
		if got != tt.want {
			t.Fatalf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func strPtr(s string) *string { return &s }
