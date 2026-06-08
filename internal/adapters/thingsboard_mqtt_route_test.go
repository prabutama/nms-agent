package adapters

import (
	"encoding/json"
	"testing"
	"time"

	"nms-agent/internal/models"
)

func TestThingsBoardMQTTAdapter_DirectModeProjectsRouteStringsToAttributes(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "mode": "direct", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}
	ts := time.Unix(10, 0).UTC()
	v := 1.0
	s := "10.10.10.1"
	batch := []models.Telemetry{
		{DeviceID: "r1", Metric: "route.ipv4.route_count", TS: ts, ValueType: "number", ValueNumber: &v},
		{DeviceID: "r1", Metric: "route.ipv4.default.next_hop", TS: ts, ValueType: "string", ValueString: &s},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatal(err)
	}
	if len(fcli.publishes) != 2 {
		t.Fatalf("expected telemetry and attributes publish, got %d", len(fcli.publishes))
	}
	if fcli.publishes[1].topic != tbGatewayAttributesTopic {
		t.Fatalf("expected attributes topic, got %s", fcli.publishes[1].topic)
	}
	telemetryPayload := fcli.publishes[0].payload.([]byte)
	var telemetry map[string][]map[string]any
	if err := json.Unmarshal(telemetryPayload, &telemetry); err != nil {
		t.Fatal(err)
	}
	vals := telemetry["r1"][0]["values"].(map[string]any)
	if vals["route.ipv4.default.next_hop"] != nil {
		t.Fatalf("route string should not stay in direct telemetry")
	}
	attrPayload := fcli.publishes[1].payload.([]byte)
	var attrs map[string]map[string]any
	if err := json.Unmarshal(attrPayload, &attrs); err != nil {
		t.Fatal(err)
	}
	if attrs["r1"]["route.ipv4.default.next_hop"] != s {
		t.Fatalf("expected route string in attributes")
	}
}
