package thingsboardmqtt

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

	"nms-agent/internal/adapters/base"
	tbintegration "nms-agent/internal/integrations/thingsboard"
	"nms-agent/internal/models"
	"nms-agent/internal/routes"
)

type thingsboardMQTTConfig struct {
	BrokerURL      string
	ClientID       string
	QoS            byte
	Retain         bool
	AutoReconnect  bool
	StrictQueue    bool
	ConnectTimeout time.Duration
	PublishTimeout time.Duration
	Provisioning   tbintegration.ProvisioningConfig
	Integration    tbintegration.Config
}

const (
	tbRelationSyncMinInterval = time.Minute
	tbTopologySyncMinInterval = 30 * time.Second
)

type relationEnsurer interface {
	EnsureContainsRelations(ctx context.Context, deviceNames []string) error
}

type topologyPublisher interface {
	PublishIfChanged(ctx context.Context, snapshots []routes.RouteSnapshot) error
}

type tokenStore interface {
	GetThingsBoardToken(ctx context.Context, deviceID string) (string, bool, error)
	SaveThingsBoardToken(ctx context.Context, deviceID, token string) error
	MarkThingsBoardTokenUsed(ctx context.Context, deviceID string) error
}

type deviceProvisioner interface {
	ProvisionDevice(ctx context.Context, deviceName string) (string, error)
}

func parseConfig(cfg map[string]any) (thingsboardMQTTConfig, error) {
	c := thingsboardMQTTConfig{
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
	for _, oldKey := range []string{"mode", "topic", "access_token"} {
		if _, ok := cfg[oldKey]; ok {
			return c, fmt.Errorf("thingsboard_mqtt no longer supports config key %q", oldKey)
		}
	}

	if v, ok := cfg["broker"].(string); ok {
		c.BrokerURL = strings.TrimSpace(v)
	}
	if v, ok := cfg["provisioning"].(map[string]any); ok {
		if s, ok := v["base_url"].(string); ok {
			c.Provisioning.BaseURL = strings.TrimSpace(s)
		}
		if s, ok := v["device_key"].(string); ok {
			c.Provisioning.DeviceKey = strings.TrimSpace(s)
		}
		if s, ok := v["device_secret"].(string); ok {
			c.Provisioning.DeviceSecret = strings.TrimSpace(s)
		}
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
		if iv, err := strconvAtoi(strings.TrimSpace(v)); err == nil {
			if iv >= 0 && iv <= 2 {
				c.QoS = byte(iv)
			}
		}
	}

	if c.BrokerURL == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'broker'")
	}
	if c.Provisioning.BaseURL == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'provisioning.base_url'")
	}
	if c.Provisioning.DeviceKey == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'provisioning.device_key'")
	}
	if c.Provisioning.DeviceSecret == "" {
		return c, errors.New("thingsboard_mqtt requires config key 'provisioning.device_secret'")
	}
	if !strings.Contains(c.BrokerURL, "://") {
		c.BrokerURL = "tcp://" + c.BrokerURL
	}
	return c, nil
}

func strconvAtoi(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

type genericMQTTClient interface {
	IsConnected() bool
	IsConnectionOpen() bool
	Connect() mqtt.Token
	Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
	Disconnect(quiesce uint)
}

type ThingsBoardMQTTAdapter struct {
	cfg     thingsboardMQTTConfig
	clients map[string]genericMQTTClient
	obs     base.AdapterObserver
	rest    *tbintegration.Client
	rels    relationEnsurer
	topo    topologyPublisher
	alarms  *tbintegration.AlarmManager
	tokens  tokenStore
	prov    deviceProvisioner

	now                 func() time.Time
	lastRelationSync    time.Time
	lastTopologySync    time.Time
	seenRelationDevices map[string]struct{}
	deviceAddresses     map[string]string
	sentDeviceAddresses map[string]string
	deviceSiteKeys      map[string]string
	sentDeviceSiteKeys  map[string]string
}

func (a *ThingsBoardMQTTAdapter) SetThingsBoardTokenStore(store interface {
	GetThingsBoardToken(context.Context, string) (string, bool, error)
	SaveThingsBoardToken(context.Context, string, string) error
	MarkThingsBoardTokenUsed(context.Context, string) error
}) {
	a.tokens = store
}

func (a *ThingsBoardMQTTAdapter) SetObserver(hub base.AdapterObserver) {
	a.obs = hub
}

func (a *ThingsBoardMQTTAdapter) SetDeviceAddresses(addresses map[string]string) {
	if a == nil {
		return
	}
	cp := map[string]string{}
	for k, v := range addresses {
		if k == "" || v == "" {
			continue
		}
		cp[k] = v
	}
	a.deviceAddresses = cp
	if a.sentDeviceAddresses == nil {
		a.sentDeviceAddresses = map[string]string{}
	}
}

// SetDeviceSiteKeys publishes demo/operations grouping metadata as device attributes.
// Asset relations remain managed in ThingsBoard and are not inferred from this value.
func (a *ThingsBoardMQTTAdapter) SetDeviceSiteKeys(siteKeys map[string]string) {
	if a == nil {
		return
	}
	a.deviceSiteKeys = map[string]string{}
	for deviceID, siteKey := range siteKeys {
		if deviceID != "" && siteKey != "" {
			a.deviceSiteKeys[deviceID] = siteKey
		}
	}
	if a.sentDeviceSiteKeys == nil {
		a.sentDeviceSiteKeys = map[string]string{}
	}
}

func NewAdapter(cfg map[string]any) (*ThingsBoardMQTTAdapter, error) {
	c, err := parseConfig(cfg)
	if err != nil {
		return nil, err
	}
	if c.StrictQueue {
		c.AutoReconnect = false
	}

	adapter := &ThingsBoardMQTTAdapter{cfg: c, clients: map[string]genericMQTTClient{}, prov: tbintegration.NewProvisioningClient(c.Provisioning), now: time.Now, seenRelationDevices: map[string]struct{}{}, deviceAddresses: map[string]string{}, sentDeviceAddresses: map[string]string{}, deviceSiteKeys: map[string]string{}, sentDeviceSiteKeys: map[string]string{}}
	if c.Integration.API.BaseURL != "" && c.Integration.API.APIKey != "" && c.Integration.Site.AssetID != "" {
		rest := tbintegration.NewClient(c.Integration.API)
		adapter.rest = rest
		adapter.rels = tbintegration.NewRelationReconciler(rest, c.Integration.Site)
		adapter.topo = tbintegration.NewTopologyPublisher(rest, c.Integration.Site)
		adapter.alarms = tbintegration.NewAlarmManager(rest, c.Integration.Site)
	}
	return adapter, nil
}

func (a *ThingsBoardMQTTAdapter) clientForDevice(deviceID, token string) genericMQTTClient {
	if a.clients == nil {
		a.clients = map[string]genericMQTTClient{}
	}
	if cli := a.clients[deviceID]; cli != nil {
		return cli
	}
	opts := mqtt.NewClientOptions().AddBroker(a.cfg.BrokerURL)
	clientID := a.cfg.ClientID
	if clientID != "" {
		clientID = clientID + "-" + sanitizeKeyPart(deviceID)
	}
	if clientID != "" {
		opts.SetClientID(clientID)
	}
	opts.SetUsername(token)
	opts.SetPassword("")
	opts.SetConnectTimeout(a.cfg.ConnectTimeout)
	opts.SetAutoReconnect(a.cfg.AutoReconnect)
	opts.SetCleanSession(true)
	cli := mqtt.NewClient(opts)
	a.clients[deviceID] = cli
	return cli
}

func (a *ThingsBoardMQTTAdapter) ensureConnected(ctx context.Context, cli genericMQTTClient) error {
	_ = ctx
	if a == nil || cli == nil {
		return errors.New("thingsboard_mqtt adapter not initialized")
	}
	if cli.IsConnected() && cli.IsConnectionOpen() {
		return nil
	}
	tok := cli.Connect()
	if !tok.WaitTimeout(a.cfg.ConnectTimeout) {
		return errors.New("mqtt connect timeout")
	}
	if err := tok.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	return nil
}

type tbDeviceTelemetry struct {
	TS     int64          `json:"ts"`
	Values map[string]any `json:"values"`
}

const (
	tbDeviceTelemetryTopic  = "v1/devices/me/telemetry"
	tbDeviceAttributesTopic = "v1/devices/me/attributes"
)

var (
	tbKeySeparatorChars = regexp.MustCompile(`[\s/:.]+`)
	tbInvalidKeyChars   = regexp.MustCompile(`[^a-z0-9-]+`)
	tbRepeatedDashes    = regexp.MustCompile(`-+`)
)

func (a *ThingsBoardMQTTAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	if len(batch) == 0 {
		return nil
	}
	telemetryPayload, attrPayload, err := buildPayloads(batch)
	if err != nil {
		return err
	}
	a.mergeDeviceAddressAttributes(attrPayload, batch)
	a.mergeDeviceSiteAttributes(attrPayload, batch)
	for _, deviceID := range uniqueDeviceNames(batch) {
		token, err := a.tokenForDevice(ctx, deviceID)
		if err != nil {
			return err
		}
		cli := a.clientForDevice(deviceID, token)
		if err := a.ensureConnected(ctx, cli); err != nil {
			if a.obs != nil {
				a.obs.UpdateStatus("connect_failed", err.Error())
			}
			return err
		}
		if a.cfg.StrictQueue && !(cli.IsConnected() && cli.IsConnectionOpen()) {
			if a.obs != nil {
				a.obs.UpdateStatus("not_connected", "broker unreachable")
			}
			return errors.New("mqtt not connected")
		}
		if entries := telemetryPayload[deviceID]; len(entries) > 0 {
			if err := a.publishJSON(cli, tbDeviceTelemetryTopic, entries, "thingsboard telemetry"); err != nil {
				return err
			}
		}
		if attrs := attrPayload[deviceID]; len(attrs) > 0 {
			if err := a.publishJSON(cli, tbDeviceAttributesTopic, attrs, "thingsboard attributes"); err != nil {
				return err
			}
			a.markDeviceAddressAttributesSent(deviceID, attrs)
		}
		if a.tokens != nil {
			_ = a.tokens.MarkThingsBoardTokenUsed(ctx, deviceID)
		}
	}
	if a.obs != nil {
		a.obs.Update(batch)
		a.obs.UpdateStatus("published", fmt.Sprintf("count=%d", len(batch)))
	}
	a.runManagementSideEffects(ctx, batch)
	return nil
}

func (a *ThingsBoardMQTTAdapter) publishJSON(cli genericMQTTClient, topic string, payload any, label string) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", label, err)
	}
	tok := cli.Publish(topic, a.cfg.QoS, a.cfg.Retain, b)
	if !tok.WaitTimeout(a.cfg.PublishTimeout) {
		return errors.New("mqtt publish timeout")
	}
	if err := tok.Error(); err != nil {
		return fmt.Errorf("mqtt publish: %w", err)
	}
	return nil
}

func (a *ThingsBoardMQTTAdapter) tokenForDevice(ctx context.Context, deviceID string) (string, error) {
	if a.tokens == nil {
		return "", errors.New("thingsboard token store is not configured")
	}
	if token, ok, err := a.tokens.GetThingsBoardToken(ctx, deviceID); err != nil {
		return "", err
	} else if ok {
		return token, nil
	}
	if a.prov == nil {
		return "", errors.New("thingsboard provisioning client is not configured")
	}
	token, err := a.prov.ProvisionDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	if err := a.tokens.SaveThingsBoardToken(ctx, deviceID, token); err != nil {
		return "", err
	}
	return token, nil
}

func (a *ThingsBoardMQTTAdapter) mergeDeviceAddressAttributes(attrPayload map[string]map[string]any, batch []models.Telemetry) {
	if len(batch) == 0 || len(a.deviceAddresses) == 0 {
		return
	}
	if a.sentDeviceAddresses == nil {
		a.sentDeviceAddresses = map[string]string{}
	}
	for _, deviceID := range uniqueDeviceNames(batch) {
		address := a.deviceAddresses[deviceID]
		if address == "" || a.sentDeviceAddresses[deviceID] == address {
			continue
		}
		deviceAttrs := attrPayload[deviceID]
		if deviceAttrs == nil {
			deviceAttrs = map[string]any{}
		}
		deviceAttrs["ip_address"] = address
		attrPayload[deviceID] = deviceAttrs
	}
}

func (a *ThingsBoardMQTTAdapter) mergeDeviceSiteAttributes(attrPayload map[string]map[string]any, batch []models.Telemetry) {
	if len(batch) == 0 || len(a.deviceSiteKeys) == 0 {
		return
	}
	if a.sentDeviceSiteKeys == nil {
		a.sentDeviceSiteKeys = map[string]string{}
	}
	for _, deviceID := range uniqueDeviceNames(batch) {
		siteKey := a.deviceSiteKeys[deviceID]
		if siteKey == "" || a.sentDeviceSiteKeys[deviceID] == siteKey {
			continue
		}
		deviceAttrs := attrPayload[deviceID]
		if deviceAttrs == nil {
			deviceAttrs = map[string]any{}
		}
		deviceAttrs["site_key"] = siteKey
		attrPayload[deviceID] = deviceAttrs
	}
}

func (a *ThingsBoardMQTTAdapter) markDeviceAddressAttributesSent(deviceID string, attrs map[string]any) {
	if len(attrs) == 0 {
		return
	}
	if a.sentDeviceAddresses == nil {
		a.sentDeviceAddresses = map[string]string{}
	}
	if ip, ok := attrs["ip_address"].(string); ok && ip != "" {
		a.sentDeviceAddresses[deviceID] = ip
	}
	if siteKey, ok := attrs["site_key"].(string); ok && siteKey != "" {
		a.sentDeviceSiteKeys[deviceID] = siteKey
	}
}

func (a *ThingsBoardMQTTAdapter) runManagementSideEffects(ctx context.Context, batch []models.Telemetry) {
	if a.rels == nil && a.topo == nil {
		return
	}
	deviceNames := uniqueDeviceNames(batch)
	fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] management side-effects: devices=%d site=%s\n", len(deviceNames), a.cfg.Integration.Site.Key)
	if a.rels != nil && len(deviceNames) > 0 && a.shouldRunRelationSync(deviceNames) {
		if err := a.rels.EnsureContainsRelations(ctx, deviceNames); err != nil {
			fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] relation error: %v\n", err)
			if a.obs != nil {
				a.obs.UpdateStatus("tb_relation_warning", err.Error())
			}
		} else {
			a.markRelationSync(deviceNames)
		}
	}
	if a.topo != nil {
		snapshots := routeSnapshotsFromBatch(batch)
		if len(snapshots) > 0 && a.shouldRunTopologySync() {
			if err := a.topo.PublishIfChanged(ctx, snapshots); err != nil {
				fmt.Fprintf(os.Stderr, "[thingsboard_mqtt] topology error: %v\n", err)
				if a.obs != nil {
					a.obs.UpdateStatus("tb_topology_warning", err.Error())
				}
			} else {
				a.markTopologySync()
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

func (a *ThingsBoardMQTTAdapter) shouldRunRelationSync(deviceNames []string) bool {
	nowFn := a.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()
	if a.seenRelationDevices == nil {
		a.seenRelationDevices = map[string]struct{}{}
	}
	for _, name := range deviceNames {
		if _, ok := a.seenRelationDevices[name]; !ok {
			return true
		}
	}
	return a.lastRelationSync.IsZero() || now.Sub(a.lastRelationSync) >= tbRelationSyncMinInterval
}

func (a *ThingsBoardMQTTAdapter) markRelationSync(deviceNames []string) {
	nowFn := a.now
	if nowFn == nil {
		nowFn = time.Now
	}
	if a.seenRelationDevices == nil {
		a.seenRelationDevices = map[string]struct{}{}
	}
	for _, name := range deviceNames {
		a.seenRelationDevices[name] = struct{}{}
	}
	a.lastRelationSync = nowFn()
}

func (a *ThingsBoardMQTTAdapter) shouldRunTopologySync() bool {
	nowFn := a.now
	if nowFn == nil {
		nowFn = time.Now
	}
	return a.lastTopologySync.IsZero() || nowFn().Sub(a.lastTopologySync) >= tbTopologySyncMinInterval
}

func (a *ThingsBoardMQTTAdapter) markTopologySync() {
	nowFn := a.now
	if nowFn == nil {
		nowFn = time.Now
	}
	a.lastTopologySync = nowFn()
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

func buildPayloads(batch []models.Telemetry) (map[string][]tbDeviceTelemetry, map[string]map[string]any, error) {
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
		if flatKey, ok := flattenedIndexedKey(t); ok {
			metricKey = flatKey
		}
		if isRouteAttributeMetric(t) {
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
	out := make(map[string][]tbDeviceTelemetry, len(byDevice))
	for deviceID, perTS := range byDevice {
		entries := make([]tbDeviceTelemetry, 0, len(perTS))
		for _, ts := range orderedTS[deviceID] {
			entries = append(entries, tbDeviceTelemetry{TS: ts, Values: perTS[ts]})
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

func flattenedIndexedKey(t models.Telemetry) (string, bool) {
	if strings.HasPrefix(t.Metric, "snmp.if.") {
		return flattenedInterfaceKey(t)
	}
	if strings.HasPrefix(t.Metric, "snmp.host.storage.") {
		return flattenedStorageKey(t)
	}
	return "", false
}

func flattenedInterfaceKey(t models.Telemetry) (string, bool) {
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
	identity = sanitizeKeyPart(identity)
	if identity == "" {
		identity = "idx" + sanitizeKeyPart(ifIndex)
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

func flattenedStorageKey(t models.Telemetry) (string, bool) {
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
	flat = append(flat, "idx"+sanitizeKeyPart(ifIndex))
	flat = append(flat, parts[3:]...)
	return strings.Join(flat, "."), true
}

func sanitizeKeyPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = tbKeySeparatorChars.ReplaceAllString(s, "-")
	s = tbInvalidKeyChars.ReplaceAllString(s, "")
	s = tbRepeatedDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func (a *ThingsBoardMQTTAdapter) HealthCheck(ctx context.Context) error {
	_ = ctx
	if a == nil {
		return errors.New("thingsboard_mqtt adapter not initialized")
	}
	if a.cfg.BrokerURL == "" || a.cfg.Provisioning.BaseURL == "" || a.cfg.Provisioning.DeviceKey == "" || a.cfg.Provisioning.DeviceSecret == "" {
		return errors.New("thingsboard_mqtt config is incomplete")
	}
	return nil
}

func (a *ThingsBoardMQTTAdapter) Close() error {
	if a == nil {
		return nil
	}
	for _, cli := range a.clients {
		if cli != nil {
			cli.Disconnect(250)
		}
	}
	return nil
}
