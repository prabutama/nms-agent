package thingsboard

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"nms-agent/internal/models"
)

type alarmState struct {
	AlarmID   string
	AlarmType string
	Severity  string
	Active    bool
}

type alarmClient interface {
	CreateAlarm(ctx context.Context, alarm AlarmRequest) (*Alarm, error)
	ClearAlarm(ctx context.Context, alarmID string) error
	GetAlarmsByEntity(ctx context.Context, entityType, entityID string) ([]Alarm, error)
	GetDeviceByName(ctx context.Context, name string) (*DeviceInfo, error)
}

type AlarmManager struct {
	client          alarmClient
	site            SiteConfig
	mu              sync.Mutex
	cache           map[string]*alarmState
	originatorCache map[string]EntityRef
}

func NewAlarmManager(client *Client, site SiteConfig) *AlarmManager {
	return &AlarmManager{client: client, site: site, cache: map[string]*alarmState{}, originatorCache: map[string]EntityRef{}}
}

func (m *AlarmManager) ProcessBatch(ctx context.Context, batch []models.Telemetry) error {
	if m == nil || m.client == nil {
		return nil
	}
	for _, t := range batch {
		status := strings.TrimSpace(t.Tags["threshold.status"])
		if status == "" {
			continue
		}
		alarmType := metricToAlarmType(t.Metric)
		key := alarmKey(t.DeviceID, alarmType)

		switch status {
		case "critical", "warning":
			if err := m.upsertAlarm(ctx, key, t, alarmType, status); err != nil {
				return err
			}
		case "normal", "ok":
			if err := m.clearAlarm(ctx, key, t.DeviceID, alarmType); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *AlarmManager) upsertAlarm(ctx context.Context, key string, t models.Telemetry, alarmType, status string) error {
	state, originator, err := m.ensureAlarmState(ctx, key, t.DeviceID, alarmType)
	if err != nil {
		return err
	}
	severity := strings.ToUpper(status)
	if state != nil && state.Active {
		if state.Severity == severity {
			return nil
		}
		if err := m.client.ClearAlarm(ctx, state.AlarmID); err != nil {
			return err
		}
	}
	req := m.buildAlarmRequest(t, originator, alarmType, status, severity)
	alarm, err := m.client.CreateAlarm(ctx, req)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.cache[key] = &alarmState{AlarmID: alarm.ID.ID, AlarmType: alarmType, Severity: severity, Active: true}
	m.mu.Unlock()
	return nil
}

func (m *AlarmManager) buildAlarmRequest(t models.Telemetry, originator *EntityRef, alarmType, status, severity string) AlarmRequest {
	req := AlarmRequest{
		Type:                   alarmType,
		Originator:             *originator,
		Severity:               severity,
		Acknowledged:           false,
		Cleared:                false,
		Propagate:              true,
		PropagateToOwner:       true,
		PropagateToTenant:      true,
		PropagateRelationTypes: []string{"Contains"},
		StartTs:                t.TS.UnixMilli(),
		Name:                   alarmType,
		Details: map[string]any{
			"metric":        t.Metric,
			"device_id":     t.DeviceID,
			"device_name":   t.DeviceID,
			"threshold":     t.Tags["threshold.rule"],
			"status":        status,
			"site_key":      m.site.Key,
			"site_asset_id": m.site.AssetID,
		},
	}
	if t.ValueNumber != nil {
		req.Details["value"] = *t.ValueNumber
	}
	if t.ValueString != nil {
		req.Details["value"] = *t.ValueString
	}
	return req
}

func (m *AlarmManager) clearAlarm(ctx context.Context, key, deviceID, alarmType string) error {
	state, _, err := m.ensureAlarmState(ctx, key, deviceID, alarmType)
	if err != nil {
		return err
	}
	if state == nil || state.AlarmID == "" || !state.Active {
		return nil
	}
	if err := m.client.ClearAlarm(ctx, state.AlarmID); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.cache, key)
	m.mu.Unlock()
	return nil
}

func (m *AlarmManager) lookupOriginator(ctx context.Context, deviceName string) (*EntityRef, error) {
	if deviceName == "" {
		return nil, fmt.Errorf("alarm originator device name is empty")
	}
	m.mu.Lock()
	if ref, ok := m.originatorCache[deviceName]; ok {
		m.mu.Unlock()
		return &ref, nil
	}
	m.mu.Unlock()
	device, err := m.client.GetDeviceByName(ctx, deviceName)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.originatorCache[deviceName] = device.ID
	m.mu.Unlock()
	return &device.ID, nil
}

func (m *AlarmManager) ensureAlarmState(ctx context.Context, key, deviceID, alarmType string) (*alarmState, *EntityRef, error) {
	originator, err := m.lookupOriginator(ctx, deviceID)
	if err != nil {
		return nil, nil, err
	}
	m.mu.Lock()
	if state, ok := m.cache[key]; ok && state != nil && state.AlarmID != "" && state.Active {
		copied := *state
		m.mu.Unlock()
		return &copied, originator, nil
	}
	m.mu.Unlock()
	alarms, err := m.client.GetAlarmsByEntity(ctx, originator.EntityType, originator.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, alarm := range alarms {
		if alarm.Type != alarmType || alarm.Cleared {
			continue
		}
		state := &alarmState{AlarmID: alarm.ID.ID, AlarmType: alarm.Type, Severity: alarm.Severity, Active: true}
		m.mu.Lock()
		m.cache[key] = state
		m.mu.Unlock()
		copied := *state
		return &copied, originator, nil
	}
	return nil, originator, nil
}

func alarmKey(deviceID, alarmType string) string {
	return deviceID + "::" + alarmType
}

func metricToAlarmType(metric string) string {
	switch {
	case metric == "icmp.reachable":
		return "DEVICE_DOWN"
	case metric == "icmp.latency_ms":
		return "HIGH_LATENCY"
	case metric == "icmp.packet_loss_pct":
		return "PACKET_LOSS"
	case metric == "icmp.jitter_ms":
		return "HIGH_JITTER"
	case strings.HasPrefix(metric, "snmp.host.cpu"):
		return "HIGH_CPU"
	case strings.HasPrefix(metric, "snmp.host.memory"):
		return "HIGH_MEMORY"
	case strings.Contains(metric, "rx_utilization"):
		return "HIGH_RX_UTILIZATION"
	case strings.Contains(metric, "tx_utilization"):
		return "HIGH_TX_UTILIZATION"
	case strings.Contains(metric, "oper_status"):
		return "INTERFACE_DOWN"
	default:
		out := strings.ToUpper(strings.ReplaceAll(metric, ".", "_"))
		out = strings.ReplaceAll(out, "-", "_")
		if out == "" {
			out = "ALARM_UNKNOWN"
		}
		return out
	}
}
