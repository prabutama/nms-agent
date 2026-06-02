package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"nms-agent/internal/models"
)

type thingsboardMQTTConfig struct {
	BrokerURL      string
	Topic          string
	AccessToken    string
	ClientID       string
	QoS            byte
	Retain         bool
	AutoReconnect  bool
	StrictQueue    bool
	ConnectTimeout time.Duration
	PublishTimeout time.Duration
}

func parseThingsBoardMQTTConfig(cfg map[string]any) (thingsboardMQTTConfig, error) {
	c := thingsboardMQTTConfig{
		Topic:          "v1/gateway/telemetry",
		QoS:            1,
		Retain:         false,
		AutoReconnect:  true,
		StrictQueue:    false,
		ConnectTimeout: 5 * time.Second,
		PublishTimeout: 5 * time.Second,
	}
	if cfg == nil {
		return c, errors.New("thingsboard_mqtt config is required")
	}

	if v, ok := cfg["broker"].(string); ok {
		c.BrokerURL = strings.TrimSpace(v)
	}
	if v, ok := cfg["topic"].(string); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			c.Topic = v
		}
	}
	if v, ok := cfg["access_token"].(string); ok {
		c.AccessToken = strings.TrimSpace(v)
	}
	if v, ok := cfg["client_id"].(string); ok {
		c.ClientID = strings.TrimSpace(v)
	}
	if v, ok := cfg["retain"].(bool); ok {
		c.Retain = v
	}
	if v, ok := cfg["auto_reconnect"].(bool); ok {
		c.AutoReconnect = v
	}
	if v, ok := cfg["strict_queue_mode"].(bool); ok {
		c.StrictQueue = v
	}
	if v, ok := cfg["connect_timeout"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.ConnectTimeout = d
		}
	}
	if v, ok := cfg["publish_timeout"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.PublishTimeout = d
		}
	}

	switch v := cfg["qos"].(type) {
	case int:
		if v >= 0 && v <= 2 {
			c.QoS = byte(v)
		}
	case int64:
		if v >= 0 && v <= 2 {
			c.QoS = byte(v)
		}
	case float64:
		iv := int(v)
		if float64(iv) == v && iv >= 0 && iv <= 2 {
			c.QoS = byte(iv)
		}
	case string:
		if iv, err := parseInt(strings.TrimSpace(v)); err == nil {
			if iv >= 0 && iv <= 2 {
				c.QoS = byte(iv)
			}
		}
	}

	if c.BrokerURL == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'broker'")
	}
	if c.AccessToken == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'access_token'")
	}
	if !strings.Contains(c.BrokerURL, "://") {
		c.BrokerURL = "tcp://" + c.BrokerURL
	}
	return c, nil
}

type ThingsBoardMQTTAdapter struct {
	cfg    thingsboardMQTTConfig
	client genericMQTTClient
	obs    AdapterObserver
}

func (a *ThingsBoardMQTTAdapter) SetObserver(hub AdapterObserver) {
	a.obs = hub
}

func NewThingsBoardMQTTAdapter(cfg map[string]any) (*ThingsBoardMQTTAdapter, error) {
	c, err := parseThingsBoardMQTTConfig(cfg)
	if err != nil {
		return nil, err
	}
	if c.StrictQueue {
		c.AutoReconnect = false
	}

	opts := mqtt.NewClientOptions().AddBroker(c.BrokerURL)
	if c.ClientID != "" {
		opts.SetClientID(c.ClientID)
	}
	// ThingsBoard MQTT auth: username = access token.
	opts.SetUsername(c.AccessToken)
	opts.SetPassword("")
	opts.SetConnectTimeout(c.ConnectTimeout)
	opts.SetAutoReconnect(c.AutoReconnect)
	opts.SetCleanSession(true)

	cli := mqtt.NewClient(opts)
	return &ThingsBoardMQTTAdapter{cfg: c, client: cli}, nil
}

func (a *ThingsBoardMQTTAdapter) ensureConnected(ctx context.Context) error {
	_ = ctx
	if a == nil || a.client == nil {
		return errors.New("thingsboard_mqtt adapter not initialized")
	}
	if a.client.IsConnected() && a.client.IsConnectionOpen() {
		return nil
	}
	tok := a.client.Connect()
	if !tok.WaitTimeout(a.cfg.ConnectTimeout) {
		return errors.New("mqtt connect timeout")
	}
	if err := tok.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	return nil
}

type tbGatewayTelemetry struct {
	TS     int64          `json:"ts"`
	Values map[string]any `json:"values"`
}

func (a *ThingsBoardMQTTAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	if len(batch) == 0 {
		return nil
	}
	if err := a.ensureConnected(ctx); err != nil {
		if a.obs != nil {
			a.obs.UpdateStatus("connect_failed", err.Error())
		}
		return err
	}
	if a.cfg.StrictQueue && !(a.client.IsConnected() && a.client.IsConnectionOpen()) {
		if a.obs != nil {
			a.obs.UpdateStatus("not_connected", "broker unreachable")
		}
		return errors.New("mqtt not connected")
	}

	for _, t := range batch {
		if strings.TrimSpace(t.DeviceID) == "" {
			return errors.New("telemetry DeviceID is required")
		}
		if strings.TrimSpace(t.Metric) == "" {
			return errors.New("telemetry Metric is required")
		}

		values := map[string]any{}
		switch t.ValueType {
		case "number":
			if t.ValueNumber == nil {
				return errors.New("telemetry ValueNumber is nil")
			}
			values[t.Metric] = *t.ValueNumber
		case "string":
			if t.ValueString == nil {
				return errors.New("telemetry ValueString is nil")
			}
			values[t.Metric] = *t.ValueString
		default:
			return fmt.Errorf("unsupported ValueType %q", t.ValueType)
		}

		// Carry full canonical metadata as additional telemetry keys.
		values[t.Metric+"__value_type"] = t.ValueType
		if t.Tags != nil {
			values[t.Metric+"__tags"] = t.Tags
		}

		payload := map[string][]tbGatewayTelemetry{
			t.DeviceID: {{TS: t.TS.UnixMilli(), Values: values}},
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal thingsboard payload: %w", err)
		}
		if a.cfg.StrictQueue && !(a.client.IsConnected() && a.client.IsConnectionOpen()) {
			return errors.New("mqtt not connected")
		}
		tok := a.client.Publish(a.cfg.Topic, a.cfg.QoS, a.cfg.Retain, b)
		if !tok.WaitTimeout(a.cfg.PublishTimeout) {
			return errors.New("mqtt publish timeout")
		}
		if err := tok.Error(); err != nil {
			return fmt.Errorf("mqtt publish: %w", err)
		}
	}
	if a.obs != nil {
		a.obs.Update(batch)
		a.obs.UpdateStatus("published", fmt.Sprintf("count=%d", len(batch)))
	}
	return nil
}

func (a *ThingsBoardMQTTAdapter) HealthCheck(ctx context.Context) error {
	if err := a.ensureConnected(ctx); err != nil {
		return err
	}
	if !(a.client.IsConnected() && a.client.IsConnectionOpen()) {
		return errors.New("mqtt not connected")
	}
	return nil
}

func (a *ThingsBoardMQTTAdapter) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	a.client.Disconnect(250)
	return nil
}
