package adapters

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

func newFakeToken(err error, waitTimeoutOK bool) *fakeToken {
	t := &fakeToken{done: make(chan struct{}), err: err, waitTimeoutOK: waitTimeoutOK}
	close(t.done)
	return t
}

func (t *fakeToken) Wait() bool {
	<-t.done
	return true
}

func (t *fakeToken) WaitTimeout(_ time.Duration) bool {
	<-t.done
	return t.waitTimeoutOK
}

func (t *fakeToken) Done() <-chan struct{} { return t.done }
func (t *fakeToken) Error() error          { return t.err }

type publishCall struct {
	topic   string
	qos     byte
	retain  bool
	payload interface{}
}

type fakeMQTTClient struct {
	connected bool
	open      bool

	connectToken mqtt.Token

	publishToken mqtt.Token
	publishes    []publishCall
}

func (c *fakeMQTTClient) IsConnected() bool { return c.connected }

func (c *fakeMQTTClient) IsConnectionOpen() bool { return c.open }

func (c *fakeMQTTClient) Connect() mqtt.Token {
	// Simulate connect attempt; adapter relies on token result.
	return c.connectToken
}

func (c *fakeMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.publishes = append(c.publishes, publishCall{topic: topic, qos: qos, retain: retained, payload: payload})
	return c.publishToken
}

func (c *fakeMQTTClient) Disconnect(_ uint) {}

func TestParseGenericMQTTConfig_RequiresBrokerAndTopic(t *testing.T) {
	if _, err := parseGenericMQTTConfig(nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parseGenericMQTTConfig(map[string]any{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseGenericMQTTConfig_DefaultsAndPrefix(t *testing.T) {
	c, err := parseGenericMQTTConfig(map[string]any{"broker": "127.0.0.1:1883", "topic": "nms-agent/telemetry"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(c.BrokerURL, "tcp://") {
		t.Fatalf("expected tcp:// prefix, got %q", c.BrokerURL)
	}
	if c.QoS != 1 || c.Retain {
		t.Fatalf("unexpected defaults: qos=%d retain=%v", c.QoS, c.Retain)
	}
}

func TestGenericMQTTAdapter_SendBatch_Success(t *testing.T) {
	cfg := map[string]any{"broker": "tcp://127.0.0.1:1883", "topic": "nms-agent/telemetry"}
	c, err := parseGenericMQTTConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeMQTTClient{
		connected:    false,
		open:         false,
		connectToken: newFakeToken(nil, true),
		publishToken: newFakeToken(nil, true),
	}
	a := &GenericMQTTAdapter{cfg: c, client: fcli}

	batch := []models.Telemetry{{DeviceID: "d1", Metric: "icmp.latency_ms", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(12.3)}}
	if err := a.SendBatch(nil, batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fcli.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(fcli.publishes))
	}
	if fcli.publishes[0].topic != c.Topic {
		t.Fatalf("expected topic %q, got %q", c.Topic, fcli.publishes[0].topic)
	}
	b, ok := fcli.publishes[0].payload.([]byte)
	if !ok {
		t.Fatalf("expected payload []byte, got %T", fcli.publishes[0].payload)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["DeviceID"] != "d1" || m["Metric"] != "icmp.latency_ms" {
		t.Fatalf("unexpected json fields: %+v", m)
	}
}

func TestGenericMQTTAdapter_SendBatch_PublishError(t *testing.T) {
	c, err := parseGenericMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "topic": "t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeMQTTClient{connected: true, open: true, publishToken: newFakeToken(errors.New("boom"), true)}
	a := &GenericMQTTAdapter{cfg: c, client: fcli}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(1)}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenericMQTTAdapter_SendBatch_ConnectTimeout(t *testing.T) {
	c, err := parseGenericMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "topic": "t", "connect_timeout": "10ms"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fcli := &fakeMQTTClient{connected: false, open: false, connectToken: newFakeToken(nil, false)}
	a := &GenericMQTTAdapter{cfg: c, client: fcli}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(1)}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenericMQTTAdapter_StrictQueueMode_RequiresOpenConnection(t *testing.T) {
	c, err := parseGenericMQTTConfig(map[string]any{"broker": "tcp://127.0.0.1:1883", "topic": "t", "strict_queue_mode": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In strict mode we expect send to fail if connection isn't open.
	fcli := &fakeMQTTClient{connected: true, open: false, connectToken: newFakeToken(nil, true), publishToken: newFakeToken(nil, true)}
	a := &GenericMQTTAdapter{cfg: c, client: fcli}
	batch := []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(1)}}
	if err := a.SendBatch(nil, batch); err == nil {
		t.Fatalf("expected error")
	}
}
