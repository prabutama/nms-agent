package thingsboardmqtt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"nms-agent/internal/models"
	"nms-agent/internal/routes"
)

type fakeTBMQTTClient struct {
	connected bool
	open      bool

	connectToken mqtt.Token
	publishToken mqtt.Token
	publishes    []publishCall
}

type publishCall struct {
	topic    string
	qos      byte
	retained bool
	payload  []byte
}

func newFakeTBMQTTClient(connected, open bool) *fakeTBMQTTClient {
	return &fakeTBMQTTClient{
		connected:    connected,
		open:         open,
		connectToken: &fakeTBToken{waitOK: true},
		publishToken: &fakeTBToken{waitOK: true},
	}
}

func (c *fakeTBMQTTClient) IsConnected() bool      { return c.connected }
func (c *fakeTBMQTTClient) IsConnectionOpen() bool { return c.open }
func (c *fakeTBMQTTClient) Connect() mqtt.Token    { return c.connectToken }
func (c *fakeTBMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	raw, _ := payload.([]byte)
	c.publishes = append(c.publishes, publishCall{topic: topic, qos: qos, retained: retained, payload: raw})
	return c.publishToken
}
func (c *fakeTBMQTTClient) Disconnect(quiesce uint) {}

type fakeTBToken struct {
	err    error
	waitOK bool
	done   chan struct{}
}

type fakeTBObserver struct {
	updates  [][]models.Telemetry
	statuses []string
	details  []string
}

type fakeRelationEnsurer struct{ calls int }

func (f *fakeRelationEnsurer) EnsureContainsRelations(_ context.Context, deviceNames []string) error {
	f.calls++
	return nil
}

type fakeTopologyPublisher struct{ calls int }

func (f *fakeTopologyPublisher) PublishIfChanged(_ context.Context, snapshots []routes.RouteSnapshot) error {
	f.calls++
	return nil
}

func (o *fakeTBObserver) Update(batch []models.Telemetry) {
	cp := append([]models.Telemetry(nil), batch...)
	o.updates = append(o.updates, cp)
}

func (o *fakeTBObserver) UpdateStatus(status string, details string) {
	o.statuses = append(o.statuses, status)
	o.details = append(o.details, details)
}

func (t *fakeTBToken) Wait() bool                     { return t.waitOK }
func (t *fakeTBToken) WaitTimeout(time.Duration) bool { return t.waitOK }
func (t *fakeTBToken) Done() <-chan struct{}          { return t.done }
func (t *fakeTBToken) Error() error                   { return t.err }

func TestThingsBoardMQTTAdapter_DirectMode(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client: fake,
	}
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(fake.publishes))
	}
	var payload map[string][]tbGatewayTelemetry
	if err := json.Unmarshal(fake.publishes[0].payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	entries, ok := payload["d1"]
	if !ok {
		t.Fatalf("expected device d1 in payload")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TS <= 0 {
		t.Fatalf("expected valid timestamp")
	}
	v, ok := entries[0].Values["test.metric"]
	if !ok {
		t.Fatalf("expected test.metric in values")
	}
	if v.(float64) != 42 {
		t.Fatalf("expected 42, got %v", v)
	}
}

func TestThingsBoardMQTTAdapter_PublishesIPAddressAttribute(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client:              fake,
		deviceAddresses:     map[string]string{"d1": "192.168.1.1"},
		sentDeviceAddresses: map[string]string{},
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 2 {
		t.Fatalf("expected 2 publishes, got %d", len(fake.publishes))
	}
	if fake.publishes[1].topic != tbGatewayAttributesTopic {
		t.Fatalf("expected attributes topic, got %q", fake.publishes[1].topic)
	}
	var payload map[string]map[string]any
	if err := json.Unmarshal(fake.publishes[1].payload, &payload); err != nil {
		t.Fatalf("decode attr payload: %v", err)
	}
	deviceAttrs := payload["d1"]
	if deviceAttrs == nil {
		t.Fatalf("expected attributes for d1")
	}
	if deviceAttrs["ip_address"] != "192.168.1.1" {
		t.Fatalf("expected ip_address=192.168.1.1, got %v", deviceAttrs["ip_address"])
	}
}

func TestThingsBoardMQTTAdapter_GatewayModeProjectsIPAddressIntoTelemetry(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{Mode: "gateway", Topic: "nms-agent/thingsboard/telemetry"},
		client:              fake,
		deviceAddresses:     map[string]string{"d1": "192.168.1.1"},
		sentDeviceAddresses: map[string]string{},
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 1 {
		t.Fatalf("expected 1 publish in gateway mode, got %d", len(fake.publishes))
	}
	var payload map[string][]tbGatewayTelemetry
	if err := json.Unmarshal(fake.publishes[0].payload, &payload); err != nil {
		t.Fatalf("decode telemetry payload: %v", err)
	}
	entries := payload["d1"]
	if len(entries) != 1 {
		t.Fatalf("expected 1 telemetry entry, got %d", len(entries))
	}
	if entries[0].Values["ip_address"] != "192.168.1.1" {
		t.Fatalf("expected ip_address in telemetry payload, got %v", entries[0].Values["ip_address"])
	}
	if entries[0].Values["ip_address__value_type"] != "string" {
		t.Fatalf("expected ip_address__value_type=string, got %v", entries[0].Values["ip_address__value_type"])
	}
}

func TestThingsBoardMQTTAdapter_DoesNotRepublishSameIPAddressAttribute(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client:              fake,
		deviceAddresses:     map[string]string{"d1": "192.168.1.1"},
		sentDeviceAddresses: map[string]string{},
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("first SendBatch: %v", err)
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("second SendBatch: %v", err)
	}
	attrPublishes := 0
	for _, pub := range fake.publishes {
		if pub.topic == tbGatewayAttributesTopic {
			attrPublishes++
		}
	}
	if attrPublishes != 1 {
		t.Fatalf("expected 1 attributes publish, got %d", attrPublishes)
	}
}

func TestThingsBoardMQTTAdapter_RepublishesIPAddressAttributeOnChange(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client:              fake,
		deviceAddresses:     map[string]string{"d1": "192.168.1.1"},
		sentDeviceAddresses: map[string]string{},
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("first SendBatch: %v", err)
	}
	a.SetDeviceAddresses(map[string]string{"d1": "192.168.1.2"})
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("second SendBatch: %v", err)
	}
	attrPublishes := make([]map[string]map[string]any, 0)
	for _, pub := range fake.publishes {
		if pub.topic != tbGatewayAttributesTopic {
			continue
		}
		var payload map[string]map[string]any
		if err := json.Unmarshal(pub.payload, &payload); err != nil {
			t.Fatalf("decode attr payload: %v", err)
		}
		attrPublishes = append(attrPublishes, payload)
	}
	if len(attrPublishes) != 2 {
		t.Fatalf("expected 2 attributes publishes, got %d", len(attrPublishes))
	}
	if attrPublishes[1]["d1"]["ip_address"] != "192.168.1.2" {
		t.Fatalf("expected updated ip_address=192.168.1.2, got %v", attrPublishes[1]["d1"]["ip_address"])
	}
}

func TestThingsBoardMQTTAdapter_ObserverPublishedStatus(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	obs := &fakeTBObserver{}
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client: fake,
	}
	a.SetObserver(obs)
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(obs.updates) != 1 {
		t.Fatalf("expected 1 observer update, got %d", len(obs.updates))
	}
	if len(obs.statuses) != 1 {
		t.Fatalf("expected 1 observer status, got %d", len(obs.statuses))
	}
	if obs.statuses[0] != "published" {
		t.Fatalf("expected published status, got %q", obs.statuses[0])
	}
}

func TestThingsBoardMQTTAdapter_ManagementSideEffectsGating(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	rels := &fakeRelationEnsurer{}
	topo := &fakeTopologyPublisher{}
	times := []time.Time{
		time.Unix(100, 0),
		time.Unix(100, 0),
		time.Unix(110, 0),
		time.Unix(110, 0),
		time.Unix(161, 0),
		time.Unix(161, 0),
	}
	idx := 0
	nowFn := func() time.Time {
		if idx >= len(times) {
			return times[len(times)-1]
		}
		v := times[idx]
		idx++
		return v
	}
	a := &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client:              fake,
		rels:                rels,
		topo:                topo,
		now:                 nowFn,
		seenRelationDevices: map[string]struct{}{},
	}
	batch := []models.Telemetry{
		{DeviceID: "router-1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()},
		{DeviceID: "router-1", Metric: "route.ipv4.snapshot", ValueType: "string", ValueString: stringPtrTB(routeSnapshotJSON()), TS: time.Now().UTC()},
	}

	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("first SendBatch: %v", err)
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("second SendBatch: %v", err)
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("third SendBatch: %v", err)
	}

	if rels.calls != 2 {
		t.Fatalf("expected relation sync called 2 times, got %d", rels.calls)
	}
	if topo.calls != 2 {
		t.Fatalf("expected topology sync called 2 times, got %d", topo.calls)
	}
}

func TestThingsBoardMQTTAdapter_MultipleDevices(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client: fake,
	}
	now := time.Now().UTC()
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "cpu", ValueType: "number", ValueNumber: floatPtr(50), TS: now},
		{DeviceID: "d2", Metric: "mem", ValueType: "number", ValueNumber: floatPtr(1024), TS: now},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(fake.publishes))
	}
	var payload map[string][]tbGatewayTelemetry
	if err := json.Unmarshal(fake.publishes[0].payload, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(payload))
	}
}

func TestThingsBoardMQTTAdapter_ConnectError(t *testing.T) {
	fake := newFakeTBMQTTClient(false, false)
	fake.connectToken = &fakeTBToken{waitOK: true, err: errors.New("connection refused")}
	obs := &fakeTBObserver{}
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{Mode: "direct", AccessToken: "test", BrokerURL: "tcp://127.0.0.1:1883"},
		client: fake,
	}
	a.SetObserver(obs)
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
	if len(obs.statuses) != 1 {
		t.Fatalf("expected 1 observer status, got %d", len(obs.statuses))
	}
	if obs.statuses[0] != "connect_failed" {
		t.Fatalf("expected connect_failed status, got %q", obs.statuses[0])
	}
}

func TestThingsBoardMQTTAdapter_StrictQueueNotConnected(t *testing.T) {
	fake := newFakeTBMQTTClient(false, false)
	a := &ThingsBoardMQTTAdapter{
		cfg: thingsboardMQTTConfig{
			Mode:        "direct",
			AccessToken: "test",
			BrokerURL:   "tcp://127.0.0.1:1883",
			StrictQueue: true,
		},
		client: fake,
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error in strict mode")
	}
}

func TestThingsBoardMQTTAdapter_HealthCheck(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		a := &ThingsBoardMQTTAdapter{
			cfg:    thingsboardMQTTConfig{},
			client: newFakeTBMQTTClient(true, true),
		}
		if err := a.HealthCheck(nil); err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}
	})
	t.Run("disconnected", func(t *testing.T) {
		a := &ThingsBoardMQTTAdapter{
			cfg:    thingsboardMQTTConfig{},
			client: newFakeTBMQTTClient(false, false),
		}
		if err := a.HealthCheck(nil); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestThingsBoardMQTTAdapter_EmptyBatch(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{Mode: "direct", Topic: "v1/gateway/telemetry"},
		client: fake,
	}
	if err := a.SendBatch(nil, nil); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 0 {
		t.Fatalf("expected 0 publishes for empty batch")
	}
}

func TestThingsBoardMQTTAdapter_Close(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{
		cfg:    thingsboardMQTTConfig{},
		client: fake,
	}
	if err := a.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func floatPtr(v float64) *float64 { return &v }

func stringPtrTB(v string) *string { return &v }

func routeSnapshotJSON() string {
	return `{"device_id":"router-1","address_family":"ipv4","supported":true,"routes":[{"device_id":"router-1","destination":"192.168.1.0/24","route_type":"connected"}]}`
}
