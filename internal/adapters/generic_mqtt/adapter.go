package genericmqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"nms-agent/internal/adapters/base"
	"nms-agent/internal/models"
)

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

type genericMQTTConfig struct {
	BrokerURL      string
	Topic          string
	ClientID       string
	Username       string
	Password       string
	QoS            byte
	Retain         bool
	AutoReconnect  bool
	StrictQueue    bool
	ConnectTimeout time.Duration
	PublishTimeout time.Duration
}

func parseConfig(cfg map[string]any) (genericMQTTConfig, error) {
	c := genericMQTTConfig{
		QoS:            1,
		Retain:         false,
		AutoReconnect:  true,
		StrictQueue:    false,
		ConnectTimeout: 5 * time.Second,
		PublishTimeout: 5 * time.Second,
	}
	if cfg == nil {
		return c, errors.New("generic_mqtt config is required")
	}

	if v, ok := cfg["broker"].(string); ok {
		c.BrokerURL = strings.TrimSpace(v)
	}
	if v, ok := cfg["topic"].(string); ok {
		c.Topic = strings.TrimSpace(v)
	}
	if v, ok := cfg["client_id"].(string); ok {
		c.ClientID = strings.TrimSpace(v)
	}
	if v, ok := cfg["username"].(string); ok {
		c.Username = v
	}
	if v, ok := cfg["password"].(string); ok {
		c.Password = v
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
		return c, errors.New("generic_mqtt requires config key 'broker'")
	}
	if c.Topic == "" {
		return c, errors.New("generic_mqtt requires config key 'topic'")
	}
	if !strings.Contains(c.BrokerURL, "://") {
		c.BrokerURL = "tcp://" + c.BrokerURL
	}
	return c, nil
}

type genericMQTTClient interface {
	IsConnected() bool
	IsConnectionOpen() bool
	Connect() mqtt.Token
	Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
	Disconnect(quiesce uint)
}

type GenericMQTTAdapter struct {
	cfg    genericMQTTConfig
	client genericMQTTClient
	obs    base.AdapterObserver
}

func (a *GenericMQTTAdapter) SetObserver(hub base.AdapterObserver) {
	a.obs = hub
}

func NewAdapter(cfg map[string]any) (*GenericMQTTAdapter, error) {
	c, err := parseConfig(cfg)
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
	if c.Username != "" {
		opts.SetUsername(c.Username)
		opts.SetPassword(c.Password)
	}
	opts.SetConnectTimeout(c.ConnectTimeout)
	opts.SetAutoReconnect(c.AutoReconnect)
	opts.SetCleanSession(true)

	cli := mqtt.NewClient(opts)
	a := &GenericMQTTAdapter{cfg: c, client: cli}
	return a, nil
}

func (a *GenericMQTTAdapter) ensureConnected(ctx context.Context) error {
	_ = ctx
	if a == nil || a.client == nil {
		return errors.New("generic_mqtt adapter not initialized")
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

func (a *GenericMQTTAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
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
		out := t
		out.TS = t.TS.In(base.GetOutputLocation())
		payload, err := json.Marshal(out)
		if err != nil {
			return fmt.Errorf("marshal telemetry: %w", err)
		}
		if a.cfg.StrictQueue && !(a.client.IsConnected() && a.client.IsConnectionOpen()) {
			return errors.New("mqtt not connected")
		}
		tok := a.client.Publish(a.cfg.Topic, a.cfg.QoS, a.cfg.Retain, payload)
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

func (a *GenericMQTTAdapter) HealthCheck(ctx context.Context) error {
	if err := a.ensureConnected(ctx); err != nil {
		return err
	}
	if !(a.client.IsConnected() && a.client.IsConnectionOpen()) {
		return errors.New("mqtt not connected")
	}
	return nil
}

func (a *GenericMQTTAdapter) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	a.client.Disconnect(250)
	return nil
}
