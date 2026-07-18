package thingsboardmqtt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	tbintegration "nms-agent/internal/integrations/thingsboard"
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
	return &fakeTBMQTTClient{connected: connected, open: open, connectToken: &fakeTBToken{waitOK: true}, publishToken: &fakeTBToken{waitOK: true}}
}

func (c *fakeTBMQTTClient) IsConnected() bool      { return c.connected }
func (c *fakeTBMQTTClient) IsConnectionOpen() bool { return c.open }
func (c *fakeTBMQTTClient) Connect() mqtt.Token {
	c.connected = true
	c.open = true
	return c.connectToken
}
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

func (t *fakeTBToken) Wait() bool                     { return t.waitOK }
func (t *fakeTBToken) WaitTimeout(time.Duration) bool { return t.waitOK }
func (t *fakeTBToken) Done() <-chan struct{}          { return t.done }
func (t *fakeTBToken) Error() error                   { return t.err }

type fakeTBObserver struct {
	updates  [][]models.Telemetry
	statuses []string
	details  []string
}

func (o *fakeTBObserver) Update(batch []models.Telemetry) {
	o.updates = append(o.updates, append([]models.Telemetry(nil), batch...))
}

func (o *fakeTBObserver) UpdateStatus(status string, details string) {
	o.statuses = append(o.statuses, status)
	o.details = append(o.details, details)
}

type fakeTokenStore struct {
	tokens map[string]string
	saves  int
	used   int
}

func (s *fakeTokenStore) GetThingsBoardToken(_ context.Context, deviceID string) (string, bool, error) {
	token, ok := s.tokens[deviceID]
	return token, ok, nil
}
func (s *fakeTokenStore) SaveThingsBoardToken(_ context.Context, deviceID, token string) error {
	if s.tokens == nil {
		s.tokens = map[string]string{}
	}
	s.tokens[deviceID] = token
	s.saves++
	return nil
}
func (s *fakeTokenStore) MarkThingsBoardTokenUsed(_ context.Context, deviceID string) error {
	s.used++
	return nil
}

type fakeProvisioner struct {
	token string
	calls int
	err   error
}

func (p *fakeProvisioner) ProvisionDevice(_ context.Context, deviceName string) (string, error) {
	p.calls++
	if p.err != nil {
		return "", p.err
	}
	return p.token, nil
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

func TestParseConfigProvisioningOnly(t *testing.T) {
	c, err := parseConfig(validProvisioningConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BrokerURL != "tcp://127.0.0.1:1883" {
		t.Fatalf("broker=%q", c.BrokerURL)
	}
	if c.Provisioning.BaseURL != "http://tb:8080" || c.Provisioning.DeviceKey != "key" || c.Provisioning.DeviceSecret != "secret" {
		t.Fatalf("unexpected provisioning config: %+v", c.Provisioning)
	}
}

func TestParseConfigRejectsOldKeys(t *testing.T) {
	for _, key := range []string{"mode", "topic", "access_token"} {
		t.Run(key, func(t *testing.T) {
			cfg := validProvisioningConfig()
			cfg[key] = "old"
			if _, err := parseConfig(cfg); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestParseConfigRequiresProvisioning(t *testing.T) {
	for _, key := range []string{"base_url", "device_key", "device_secret"} {
		t.Run(key, func(t *testing.T) {
			cfg := validProvisioningConfig()
			delete(cfg["provisioning"].(map[string]any), key)
			if _, err := parseConfig(cfg); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestThingsBoardMQTTAdapterPublishesDeviceTelemetry(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := testAdapter(fake, &fakeTokenStore{tokens: map[string]string{"d1": "token"}}, &fakeProvisioner{token: "new"})
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "test.metric", ValueType: "number", ValueNumber: floatPtr(42), TS: time.Now().UTC()}}
	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(fake.publishes))
	}
	if fake.publishes[0].topic != tbDeviceTelemetryTopic {
		t.Fatalf("topic=%q", fake.publishes[0].topic)
	}
	var payload []tbDeviceTelemetry
	if err := json.Unmarshal(fake.publishes[0].payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got := payload[0].Values["test.metric"]; got.(float64) != 42 {
		t.Fatalf("value=%v", got)
	}
}

func TestThingsBoardMQTTAdapterProvisionsMissingTokenOnce(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	store := &fakeTokenStore{tokens: map[string]string{}}
	prov := &fakeProvisioner{token: "new-token"}
	a := testAdapter(fake, store, prov)
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}
	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("first SendBatch: %v", err)
	}
	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("second SendBatch: %v", err)
	}
	if prov.calls != 1 || store.saves != 1 {
		t.Fatalf("provision calls=%d saves=%d", prov.calls, store.saves)
	}
}

func TestThingsBoardMQTTAdapterPublishesAttributes(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := testAdapter(fake, &fakeTokenStore{tokens: map[string]string{"d1": "token"}}, &fakeProvisioner{})
	a.deviceAddresses = map[string]string{"d1": "192.168.1.1"}
	a.sentDeviceAddresses = map[string]string{}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}
	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("SendBatch: %v", err)
	}
	if len(fake.publishes) != 2 {
		t.Fatalf("expected 2 publishes, got %d", len(fake.publishes))
	}
	if fake.publishes[1].topic != tbDeviceAttributesTopic {
		t.Fatalf("topic=%q", fake.publishes[1].topic)
	}
	var attrs map[string]any
	if err := json.Unmarshal(fake.publishes[1].payload, &attrs); err != nil {
		t.Fatalf("decode attrs: %v", err)
	}
	if attrs["ip_address"] != "192.168.1.1" {
		t.Fatalf("ip_address=%v", attrs["ip_address"])
	}
}

func TestThingsBoardMQTTAdapterProvisionFailure(t *testing.T) {
	a := testAdapter(newFakeTBMQTTClient(true, true), &fakeTokenStore{tokens: map[string]string{}}, &fakeProvisioner{err: errors.New("nope")})
	err := a.SendBatch(context.Background(), []models.Telemetry{{DeviceID: "d1", Metric: "m", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestThingsBoardMQTTAdapterManagementSideEffects(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	rels := &fakeRelationEnsurer{}
	topo := &fakeTopologyPublisher{}
	a := testAdapter(fake, &fakeTokenStore{tokens: map[string]string{"r1": "token"}}, &fakeProvisioner{})
	a.rels = rels
	a.topo = topo
	a.now = func() time.Time { return time.Unix(100, 0) }
	snap := routeSnapshotJSON()
	batch := []models.Telemetry{{DeviceID: "r1", Metric: "route.ipv4.snapshot", ValueType: "string", ValueString: stringPtrTB(snap), TS: time.Now().UTC()}}
	if err := a.SendBatch(context.Background(), batch); err != nil {
		t.Fatalf("SendBatch: %v", err)
	}
	if rels.calls != 1 || topo.calls != 1 {
		t.Fatalf("relations=%d topology=%d", rels.calls, topo.calls)
	}
}

func TestThingsBoardMQTTAdapterHealthCheck(t *testing.T) {
	a := &ThingsBoardMQTTAdapter{cfg: thingsboardMQTTConfig{BrokerURL: "tcp://127.0.0.1:1883", Provisioning: tbintegration.ProvisioningConfig{BaseURL: "http://tb:8080", DeviceKey: "key", DeviceSecret: "secret"}}}
	if err := a.HealthCheck(context.Background()); err != nil {
		t.Fatalf("expected ok, got: %v", err)
	}
}

func TestThingsBoardMQTTAdapterClose(t *testing.T) {
	fake := newFakeTBMQTTClient(true, true)
	a := &ThingsBoardMQTTAdapter{clients: map[string]genericMQTTClient{"d1": fake}}
	if err := a.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testAdapter(cli genericMQTTClient, store *fakeTokenStore, prov *fakeProvisioner) *ThingsBoardMQTTAdapter {
	return &ThingsBoardMQTTAdapter{
		cfg:                 thingsboardMQTTConfig{BrokerURL: "tcp://127.0.0.1:1883", QoS: 1, ConnectTimeout: time.Second, PublishTimeout: time.Second, Provisioning: tbintegration.ProvisioningConfig{BaseURL: "http://tb:8080", DeviceKey: "key", DeviceSecret: "secret"}},
		clients:             map[string]genericMQTTClient{"d1": cli, "r1": cli},
		tokens:              store,
		prov:                prov,
		now:                 time.Now,
		seenRelationDevices: map[string]struct{}{},
		deviceAddresses:     map[string]string{},
		sentDeviceAddresses: map[string]string{},
	}
}

func validProvisioningConfig() map[string]any {
	return map[string]any{
		"broker": "tcp://127.0.0.1:1883",
		"provisioning": map[string]any{
			"base_url":      "http://tb:8080",
			"device_key":    "key",
			"device_secret": "secret",
		},
	}
}

func floatPtr(v float64) *float64 { return &v }

func stringPtrTB(v string) *string { return &v }

func routeSnapshotJSON() string {
	return `{"device_id":"router-1","address_family":"ipv4","supported":true,"routes":[{"device_id":"router-1","destination":"192.168.1.0/24","route_type":"connected"}]}`
}
