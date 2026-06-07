package adapters

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"nms-agent/internal/models"
)

type fakeTBMQTTClient struct {
	connected bool
	open      bool

	connectToken mqtt.Token
	publishToken mqtt.Token
	publishes    []publishCall
}

func (c *fakeTBMQTTClient) IsConnected() bool      { return c.connected }
func (c *fakeTBMQTTClient) IsConnectionOpen() bool { return c.open }
func (c *fakeTBMQTTClient) Connect() mqtt.Token    { return c.connectToken }
func (c *fakeTBMQTTClient) Disconnect(_ uint)      {}
func (c *fakeTBMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.publishes = append(c.publishes, publishCall{topic: topic, qos: qos, retain: retained, payload: payload})
	return c.publishToken
}

func TestParseThingsBoardMQTTConfig_RequiresBrokerAndToken(t *testing.T) {
	if _, err := parseThingsBoardMQTTConfig(nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883"}); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parseThingsBoardMQTTConfig(map[string]any{"access_token": "t"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseThingsBoardMQTTConfig_GatewayModeDoesNotRequireAccessToken(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "mode": "gateway"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Mode != "gateway" {
		t.Fatalf("expected gateway mode, got %q", c.Mode)
	}
	if c.Topic != "nms-agent/thingsboard/telemetry" {
		t.Fatalf("unexpected gateway topic %q", c.Topic)
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_PayloadShape(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 12.3
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "icmp.latency_ms", TS: time.Unix(10, 0).UTC(), ValueType: "number", ValueNumber: &val, Tags: map[string]string{"unit": "ms", "threshold.status": "ok"}}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fcli.publishes) != 1 {
		t.Fatalf("expected 1 publish")
	}
	b, ok := fcli.publishes[0].payload.([]byte)
	if !ok {
		t.Fatalf("expected payload []byte")
	}
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	arr := m["d1"]
	if len(arr) != 1 {
		t.Fatalf("expected 1 telemetry entry")
	}
	if arr[0]["ts"] == nil {
		t.Fatalf("expected ts")
	}
	vals, ok := arr[0]["values"].(map[string]any)
	if !ok {
		t.Fatalf("expected values object")
	}
	if vals["icmp.latency_ms"] == nil {
		t.Fatalf("expected metric key")
	}
	if vals["icmp.latency_ms__tags"] == nil {
		t.Fatalf("expected tags key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_AggregatesMetricsByDeviceAndTimestamp(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}
	ts := time.Unix(10, 0).UTC()
	v1 := 1.0
	v2 := 12.3
	s := "router-a"
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "icmp.reachable", TS: ts, ValueType: "number", ValueNumber: &v1, Tags: map[string]string{"source": "icmp"}},
		{DeviceID: "d1", Metric: "icmp.latency_ms", TS: ts, ValueType: "number", ValueNumber: &v2, Tags: map[string]string{"source": "icmp", "unit": "ms"}},
		{DeviceID: "d1", Metric: "snmp.system.name", TS: ts, ValueType: "string", ValueString: &s, Tags: map[string]string{"source": "snmp"}},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fcli.publishes) != 1 {
		t.Fatalf("expected single publish, got %d", len(fcli.publishes))
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	arr := m["d1"]
	if len(arr) != 1 {
		t.Fatalf("expected 1 aggregated entry, got %d", len(arr))
	}
	vals := arr[0]["values"].(map[string]any)
	if vals["icmp.reachable"] == nil || vals["icmp.latency_ms"] == nil || vals["snmp.system.name"] == nil {
		t.Fatalf("expected aggregated values, got %+v", vals)
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_GroupsDifferentTimestampsSeparately(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}
	v1 := 1.0
	v2 := 2.0
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "icmp.reachable", TS: time.Unix(10, 0).UTC(), ValueType: "number", ValueNumber: &v1},
		{DeviceID: "d1", Metric: "icmp.packet_loss_pct", TS: time.Unix(11, 0).UTC(), ValueType: "number", ValueNumber: &v2},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(m["d1"]) != 2 {
		t.Fatalf("expected 2 timestamp groups, got %d", len(m["d1"]))
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_AddsFlattenedInterfaceKeyUsingIfName(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 13472.05
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.if.rx_bps",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "8", "ifName": "Gi0/1", "source": "snmp", "unit": "bps"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.if.gi0-1.rx_bps"] == nil {
		t.Fatalf("expected flattened ifName key")
	}
	if vals["snmp.if.rx_bps"] != nil {
		t.Fatalf("did not expect generic interface key")
	}
	if vals["snmp.if.gi0-1.rx_bps__tags"] == nil {
		t.Fatalf("expected flattened tags key")
	}
	if vals["snmp.if.gi0-1.rx_bps__value_type"] == nil {
		t.Fatalf("expected flattened value_type key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_SanitizesInterfaceNameToDashedLowercase(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 1.0
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.if.oper_status",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "8", "ifName": " WAN / ISP:1.core ", "source": "snmp"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.if.wan-isp-1-core.oper_status"] == nil {
		t.Fatalf("expected sanitized flattened ifName key")
	}
	if vals["snmp.if.oper_status"] != nil {
		t.Fatalf("did not expect generic interface key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_AddsFlattenedInterfaceKeyUsingIfIndexFallback(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 1.0
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.if.oper_status",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "9", "source": "snmp"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.if.idx9.oper_status"] == nil {
		t.Fatalf("expected flattened ifIndex fallback key")
	}
	if vals["snmp.if.idx9.oper_status__tags"] == nil {
		t.Fatalf("expected flattened fallback tags key")
	}
	if vals["snmp.if.oper_status"] != nil {
		t.Fatalf("did not expect generic interface key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_FallsBackToIfIndexWhenSanitizedNameEmpty(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 100.0
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.if.tx_bps",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "8", "ifName": "@@@###", "source": "snmp"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.if.idx8.tx_bps"] == nil {
		t.Fatalf("expected ifIndex fallback when sanitized name becomes empty")
	}
	if vals["snmp.if.tx_bps"] != nil {
		t.Fatalf("did not expect generic interface key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_FlattensIndexedStorageMetric(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 1024.0
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.host.storage.used_units",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "65536", "source": "snmp"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.host.storage.idx65536.used_units"] == nil {
		t.Fatalf("expected flattened storage key")
	}
	if vals["snmp.host.storage.used_units"] != nil {
		t.Fatalf("did not expect generic storage indexed key")
	}
	if vals["snmp.host.storage.idx65536.used_units__tags"] == nil {
		t.Fatalf("expected flattened storage tags key")
	}
	if vals["snmp.host.storage.idx65536.used_units__value_type"] == nil {
		t.Fatalf("expected flattened storage value_type key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_DoesNotFlattenOtherIndexedMetric(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeTBMQTTClient{connected: true, open: true, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}

	val := 7.0
	batch := []models.Telemetry{{
		DeviceID:    "router-1",
		Metric:      "snmp.other.indexed_metric",
		TS:          time.Unix(10, 0).UTC(),
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        map[string]string{"ifIndex": "42", "source": "snmp"},
	}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := fcli.publishes[0].payload.([]byte)
	var m map[string][]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	vals := m["router-1"][0]["values"].(map[string]any)
	if vals["snmp.other.indexed_metric"] == nil {
		t.Fatalf("expected generic other indexed key")
	}
	if vals["snmp.other.idx42.indexed_metric"] != nil {
		t.Fatalf("did not expect flattened other indexed key")
	}
}

func TestThingsBoardMQTTAdapter_SendBatch_PublishError(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fcli := &fakeTBMQTTClient{connected: true, open: true, publishToken: newFakeToken(errors.New("boom"), true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}
	val := 1.0
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: &val}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}

func TestThingsBoardMQTTAdapter_StrictQueueMode_DisconnectFails(t *testing.T) {
	c, err := parseThingsBoardMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token", "strict_queue_mode": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fcli := &fakeTBMQTTClient{connected: true, open: false, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &ThingsBoardMQTTAdapter{cfg: c, client: fcli}
	val := 1.0
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: &val}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}
