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
