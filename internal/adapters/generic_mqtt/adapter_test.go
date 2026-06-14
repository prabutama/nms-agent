package genericmqtt

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"nms-agent/internal/models"
)

type fakeToken struct {
	done          chan struct{}
	err           error
	waitTimeoutOK bool
}

func newFakeToken(err error, waitOK bool) *fakeToken {
	done := make(chan struct{})
	close(done)
	return &fakeToken{done: done, err: err, waitTimeoutOK: waitOK}
}
func (t *fakeToken) Wait() bool                     { return t.waitTimeoutOK }
func (t *fakeToken) WaitTimeout(time.Duration) bool { <-t.done; return t.waitTimeoutOK }
func (t *fakeToken) Done() <-chan struct{}          { return t.done }
func (t *fakeToken) Error() error                   { return t.err }

type recordedPublish struct {
	topic    string
	qos      byte
	retained bool
	payload  []byte
}

type fakeMQTTClient struct {
	connected  bool
	open       bool
	connectErr error
	publishes  []recordedPublish
	publishErr error
}

func (c *fakeMQTTClient) IsConnected() bool      { return c.connected }
func (c *fakeMQTTClient) IsConnectionOpen() bool { return c.open }
func (c *fakeMQTTClient) Connect() mqtt.Token    { return newFakeToken(c.connectErr, true) }
func (c *fakeMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	raw, _ := payload.([]byte)
	c.publishes = append(c.publishes, recordedPublish{topic: topic, qos: qos, retained: retained, payload: raw})
	return newFakeToken(c.publishErr, true)
}
func (c *fakeMQTTClient) Disconnect(quiesce uint) {}

func TestGenericMQTTAdapter_SendBatch(t *testing.T) {
	fake := &fakeMQTTClient{connected: true, open: true}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{Topic: "test/metrics"},
		client: fake,
	}
	batch := []models.Telemetry{
		{
			DeviceID:    "d1",
			Metric:      "test.metric",
			ValueType:   "number",
			ValueNumber: floatPtr(42),
			TS:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(fake.publishes))
	}
	p := fake.publishes[0]
	if p.topic != "test/metrics" {
		t.Fatalf("topic = %q", p.topic)
	}
	if p.qos != 0 {
		t.Fatalf("qos = %d", p.qos)
	}
	var decoded models.Telemetry
	if err := json.Unmarshal(p.payload, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Metric != "test.metric" {
		t.Fatalf("metric = %q", decoded.Metric)
	}
}

func TestGenericMQTTAdapter_SendBatch_MultipleMetrics(t *testing.T) {
	fake := &fakeMQTTClient{connected: true, open: true}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{Topic: "test/metrics"},
		client: fake,
	}
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	batch := []models.Telemetry{
		{DeviceID: "d1", Metric: "m1", ValueType: "number", ValueNumber: floatPtr(1), TS: now},
		{DeviceID: "d1", Metric: "m2", ValueType: "number", ValueNumber: floatPtr(2), TS: now},
		{DeviceID: "d2", Metric: "m3", ValueType: "string", ValueString: strPtr("hello"), TS: now},
	}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.publishes) != 3 {
		t.Fatalf("expected 3 publishes, got %d", len(fake.publishes))
	}
}

func TestGenericMQTTAdapter_SendBatch_NotConnected(t *testing.T) {
	fake := &fakeMQTTClient{connected: false, open: false, connectErr: errors.New("connect failed")}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{Topic: "t"},
		client: fake,
	}
	a.ensureConnected(nil)

	if err := a.SendBatch(nil, []models.Telemetry{{DeviceID: "d1", Metric: "m1", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenericMQTTAdapter_SendBatch_StrictQueue_NotConnected(t *testing.T) {
	fake := &fakeMQTTClient{connected: false, open: false}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{Topic: "t", StrictQueue: true},
		client: fake,
	}
	if err := a.SendBatch(nil, []models.Telemetry{{DeviceID: "d1", Metric: "m1", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}); err == nil {
		t.Fatalf("expected error in strict mode")
	}
}

func TestGenericMQTTAdapter_HealthCheck(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		a := &GenericMQTTAdapter{
			cfg:    genericMQTTConfig{},
			client: &fakeMQTTClient{connected: true, open: true},
		}
		if err := a.HealthCheck(nil); err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}
	})
	t.Run("disconnected", func(t *testing.T) {
		a := &GenericMQTTAdapter{
			cfg:    genericMQTTConfig{},
			client: &fakeMQTTClient{connected: false, open: false},
		}
		if err := a.HealthCheck(nil); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestGenericMQTTAdapter_Close(t *testing.T) {
	fake := &fakeMQTTClient{}
	a := &GenericMQTTAdapter{client: fake}
	if err := a.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenericMQTTAdapter_SendBatch_Error(t *testing.T) {
	fake := &fakeMQTTClient{connected: true, open: true, publishErr: errors.New("publish failed")}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{Topic: "test"},
		client: fake,
	}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m1", ValueType: "number", ValueNumber: floatPtr(1), TS: time.Now().UTC()}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenericMQTTAdapter_Reconnect(t *testing.T) {
	fake := &fakeMQTTClient{connected: false, open: false}
	a := &GenericMQTTAdapter{
		cfg:    genericMQTTConfig{BrokerURL: "tcp://localhost:1883", Topic: "test"},
		client: fake,
	}
	err := a.ensureConnected(nil)
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "connect timeout") && !strings.Contains(err.Error(), "mqtt connect") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func floatPtr(v float64) *float64 { return &v }
func strPtr(s string) *string     { return &s }
