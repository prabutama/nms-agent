package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	tbintegration "nms-agent/internal/integrations/thingsboard"
	"nms-agent/internal/models"
	"nms-agent/internal/routes"
)

type thingsboardMQTTConfig struct {
	BrokerURL      string
	Mode           string
	Topic          string
	AccessToken    string
	Username       string
	Password       string
	ClientID       string
	QoS            byte
	Retain         bool
	AutoReconnect  bool
	StrictQueue    bool
	ConnectTimeout time.Duration
	PublishTimeout time.Duration
	Integration    tbintegration.Config
}

func parseThingsBoardMQTTConfig(cfg map[string]any) (thingsboardMQTTConfig, error) {
	c := thingsboardMQTTConfig{
		Mode:           "direct",
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
	if v, ok := cfg["mode"].(string); ok {
		c.Mode = strings.TrimSpace(strings.ToLower(v))
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
	if v, ok := cfg["username"].(string); ok {
		c.Username = strings.TrimSpace(v)
	}
	if v, ok := cfg["password"].(string); ok {
		c.Password = v
	}
	if v, ok := cfg["thingsboard"].(map[string]any); ok {
		if api, ok := v["api"].(map[string]any); ok {
			if s, ok := api["base_url"].(string); ok {
				c.Integration.API.BaseURL = strings.TrimSpace(s)
			}
			if s, ok := api["api_key"].(string); ok {
				c.Integration.API.APIKey = strings.TrimSpace(s)
			}
		}
		if site, ok := v["site"].(map[string]any); ok {
			if s, ok := site["key"].(string); ok {
				c.Integration.Site.Key = strings.TrimSpace(s)
			}
			if s, ok := site["asset_id"].(string); ok {
				c.Integration.Site.AssetID = strings.TrimSpace(s)
			}
			if s, ok := site["asset_name"].(string); ok {
				c.Integration.Site.AssetName = strings.TrimSpace(s)
			}
			if s, ok := site["customer_id"].(string); ok {
				c.Integration.Site.CustomerID = strings.TrimSpace(s)
			}
		}
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
	switch c.Mode {
	case "", "direct":
		c.Mode = "direct"
		if c.Topic == "" {
			c.Topic = "v1/gateway/telemetry"
		}
		if c.AccessToken == "" {
			return c, errors.New("thingsboard_mqtt requires config key 'access_token'")
		}
	case "gateway":
		if c.Topic == "" {
			c.Topic = "nms-agent/thingsboard/telemetry"
		}
	default:
		return c, errors.New("thingsboard_mqtt requires config key 'mode' to be 'direct' or 'gateway'")
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
	rest   *tbintegration.Client
	rels   *tbintegration.RelationReconciler
	topo   *tbintegration.TopologyPublisher
	alarms *tbintegration.AlarmManager
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
	if c.Mode == "direct" {
		// ThingsBoard MQTT auth: username = access token.
		opts.SetUsername(c.AccessToken)
		opts.SetPassword("")
	} else if c.Username != "" {
		opts.SetUsername(c.Username)
		opts.SetPassword(c.Password)
	}
	opts.SetConnectTimeout(c.ConnectTimeout)
	opts.SetAutoReconnect(c.AutoReconnect)
	opts.SetCleanSession(true)

	cli := mqtt.NewClient(opts)
	adapter := &ThingsBoardMQTTAdapter{cfg: c, client: cli}
	if c.Integration.API.BaseURL != "" && c.Integration.API.APIKey != "" && c.Integration.Site.AssetID != "" {
		rest := tbintegration.NewClient(c.Integration.API)
		adapter.rest = rest
		adapter.rels = tbintegration.NewRelationReconciler(rest, c.Integration.Site)
		adapter.topo = tbintegration.NewTopologyPublisher(rest, c.Integration.Site)
		adapter.alarms = tbintegration.NewAlarmManager(rest, c.Integration.Site)
	}
	return adapter, nil
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

const tbGatewayAttributesTopic = "v1/gateway/attributes"

var (
	tbKeySeparatorChars = regexp.MustCompile(`[\s/:.]+`)
	tbInvalidKeyChars   = regexp.MustCompile(`[^a-z0-9-]+`)
	tbRepeatedDashes    = regexp.MustCompile(`-+`)
)

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

	telemetryPayload, attrPayload, err := buildThingsBoardPayloads(a.cfg.Mode, batch)
	if err != nil {
		return err
	}
	b, err := json.Marshal(telemetryPayload)
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
	if len(attrPayload) > 0 {
		attrBytes, err := json.Marshal(attrPayload)
		if err != nil {
			return fmt.Errorf("marshal thingsboard attributes payload: %w", err)
		}
		attrTok := a.client.Publish(tbGatewayAttributesTopic, a.cfg.QoS, a.cfg.Retain, attrBytes)
		if !attrTok.WaitTimeout(a.cfg.PublishTimeout) {
			return errors.New("mqtt publish timeout")
		}
		if err := attrTok.Error(); err != nil {
			return fmt.Errorf("mqtt publish attributes: %w", err)
		}
	}
	if a.obs != nil {
		a.obs.Update(batch)
		a.obs.UpdateStatus("published", fmt.Sprintf("count=%d", len(batch)))
	}
	a.runManagementSideEffects(ctx, batch)
	return nil
}

func (a *ThingsBoardMQTTAdapter) runManagementSideEffects(ctx context.Context, batch []models.Telemetry) {
	if a.rels == nil && a.topo == nil {
		return
	}
	deviceNames := uniqueDeviceNames(batch)
	fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] management side-effects: devices=%d customer_id=%s\n", len(deviceNames), a.cfg.Integration.Site.CustomerID)
	if a.rels != nil && len(deviceNames) > 0 {
		if err := a.rels.EnsureContainsRelations(ctx, deviceNames); err != nil {
			fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] relation error: %v\n", err)
			if a.obs != nil {
				a.obs.UpdateStatus("tb_relation_warning", err.Error())
			}
		}
	}
	if a.topo != nil {
		snapshots := routeSnapshotsFromBatch(batch)
		if len(snapshots) > 0 {
			if err := a.topo.PublishIfChanged(ctx, snapshots); err != nil {
				fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] topology error: %v\n", err)
				if a.obs != nil {
					a.obs.UpdateStatus("tb_topology_warning", err.Error())
				}
			}
		}
	}
	if a.alarms != nil {
		if err := a.alarms.ProcessBatch(ctx, batch); err != nil {
			fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] alarm error: %v\n", err)
			if a.obs != nil {
				a.obs.UpdateStatus("tb_alarm_warning", err.Error())
			}
		}
	}
}

func uniqueDeviceNames(batch []models.Telemetry) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(batch))
	for _, t := range batch {
		if t.DeviceID == "" || seen[t.DeviceID] {
			continue
		}
		seen[t.DeviceID] = true
		out = append(out, t.DeviceID)
	}
	return out
}

func routeSnapshotsFromBatch(batch []models.Telemetry) []routes.RouteSnapshot {
	out := make([]routes.RouteSnapshot, 0)
	for _, t := range batch {
		if t.Metric != "route.ipv4.snapshot" || t.ValueType != "string" || t.ValueString == nil || *t.ValueString == "" {
			continue
		}
		var snap routes.RouteSnapshot
		if err := json.Unmarshal([]byte(*t.ValueString), &snap); err != nil {
			continue
		}
		out = append(out, snap)
	}
	return out
}

func buildThingsBoardPayloads(mode string, batch []models.Telemetry) (map[string][]tbGatewayTelemetry, map[string]map[string]any, error) {
	byDevice := make(map[string]map[int64]map[string]any)
	orderedTS := make(map[string][]int64)
	attrs := make(map[string]map[string]any)
	for _, t := range batch {
		if strings.TrimSpace(t.DeviceID) == "" {
			return nil, nil, errors.New("telemetry DeviceID is required")
		}
		if strings.TrimSpace(t.Metric) == "" {
			return nil, nil, errors.New("telemetry Metric is required")
		}
		baseValue, err := telemetryBaseValue(t)
		if err != nil {
			return nil, nil, err
		}
		metricKey := t.Metric
		if flatKey, ok := thingsBoardFlattenedIndexedKey(t); ok {
			metricKey = flatKey
		}
		if mode == "direct" && isRouteAttributeMetric(t) {
			deviceAttrs := attrs[t.DeviceID]
			if deviceAttrs == nil {
				deviceAttrs = make(map[string]any)
				attrs[t.DeviceID] = deviceAttrs
			}
			deviceAttrs[metricKey] = baseValue
			deviceAttrs[metricKey+"__value_type"] = t.ValueType
			if t.Tags != nil {
				deviceAttrs[metricKey+"__tags"] = t.Tags
			}
			continue
		}
		ts := t.TS.UnixMilli()
		perTS := byDevice[t.DeviceID]
		if perTS == nil {
			perTS = make(map[int64]map[string]any)
			byDevice[t.DeviceID] = perTS
		}
		values := perTS[ts]
		if values == nil {
			values = make(map[string]any)
			perTS[ts] = values
			orderedTS[t.DeviceID] = append(orderedTS[t.DeviceID], ts)
		}
		values[metricKey] = baseValue
		values[metricKey+"__value_type"] = t.ValueType
		if t.Tags != nil {
			values[metricKey+"__tags"] = t.Tags
		}
	}
	out := make(map[string][]tbGatewayTelemetry, len(byDevice))
	for deviceID, perTS := range byDevice {
		entries := make([]tbGatewayTelemetry, 0, len(perTS))
		for _, ts := range orderedTS[deviceID] {
			entries = append(entries, tbGatewayTelemetry{TS: ts, Values: perTS[ts]})
		}
		out[deviceID] = entries
	}
	return out, attrs, nil
}

func isRouteAttributeMetric(t models.Telemetry) bool {
	return t.ValueType == "string" && strings.HasPrefix(t.Metric, "route.ipv4.")
}

func telemetryBaseValue(t models.Telemetry) (any, error) {
	switch t.ValueType {
	case "number":
		if t.ValueNumber == nil {
			return nil, errors.New("telemetry ValueNumber is nil")
		}
		return *t.ValueNumber, nil
	case "string":
		if t.ValueString == nil {
			return nil, errors.New("telemetry ValueString is nil")
		}
		return *t.ValueString, nil
	default:
		return nil, fmt.Errorf("unsupported ValueType %q", t.ValueType)
	}
}

func thingsBoardFlattenedIndexedKey(t models.Telemetry) (string, bool) {
	if strings.HasPrefix(t.Metric, "snmp.if.") {
		return thingsBoardFlattenedInterfaceKey(t)
	}
	if strings.HasPrefix(t.Metric, "snmp.host.storage.") {
		return thingsBoardFlattenedStorageKey(t)
	}
	return "", false
}

func thingsBoardFlattenedInterfaceKey(t models.Telemetry) (string, bool) {
	if t.Tags == nil {
		return "", false
	}
	ifIndex := strings.TrimSpace(t.Tags["ifIndex"])
	if ifIndex == "" {
		return "", false
	}
	identity := strings.TrimSpace(t.Tags["ifName"])
	if identity == "" {
		identity = "idx" + ifIndex
	}
	identity = sanitizeThingsBoardKeyPart(identity)
	if identity == "" {
		identity = "idx" + sanitizeThingsBoardKeyPart(ifIndex)
	}
	parts := strings.Split(t.Metric, ".")
	if len(parts) < 2 {
		return "", false
	}
	flat := make([]string, 0, len(parts)+1)
	flat = append(flat, parts[:2]...)
	flat = append(flat, identity)
	flat = append(flat, parts[2:]...)
	return strings.Join(flat, "."), true
}

func thingsBoardFlattenedStorageKey(t models.Telemetry) (string, bool) {
	if t.Tags == nil {
		return "", false
	}
	ifIndex := strings.TrimSpace(t.Tags["ifIndex"])
	if ifIndex == "" {
		return "", false
	}
	parts := strings.Split(t.Metric, ".")
	if len(parts) < 4 {
		return "", false
	}
	flat := make([]string, 0, len(parts)+1)
	flat = append(flat, parts[:3]...)
	flat = append(flat, "idx"+sanitizeThingsBoardKeyPart(ifIndex))
	flat = append(flat, parts[3:]...)
	return strings.Join(flat, "."), true
}

func sanitizeThingsBoardKeyPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = tbKeySeparatorChars.ReplaceAllString(s, "-")
	s = tbInvalidKeyChars.ReplaceAllString(s, "")
	s = tbRepeatedDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
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
