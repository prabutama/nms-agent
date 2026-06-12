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

type AlarmManager struct {
	client *Client
	site   SiteConfig
	mu     sync.Mutex
	cache  map[string]*alarmState
}

func NewAlarmManager(client *Client, site SiteConfig) *AlarmManager {
	return &AlarmManager{client: client, site: site, cache: map[string]*alarmState{}}
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
			if err := m.clearAlarm(ctx, key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *AlarmManager) upsertAlarm(ctx context.Context, key string, t models.Telemetry, alarmType, status string) error {
	originator, err := m.lookupOriginator(ctx, t.DeviceID)
	if err != nil {
		return err
	}
	severity := strings.ToUpper(status)
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
	alarm, err := m.client.CreateAlarm(ctx, req)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.cache[key] = &alarmState{AlarmID: alarm.ID.ID, AlarmType: alarmType, Severity: severity, Active: true}
	m.mu.Unlock()
	return nil
}

func (m *AlarmManager) clearAlarm(ctx context.Context, key string) error {
	m.mu.Lock()
	state, ok := m.cache[key]
	m.mu.Unlock()
	if !ok || state == nil || state.AlarmID == "" || !state.Active {
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
	device, err := m.client.GetDeviceByName(ctx, deviceName)
	if err != nil {
		return nil, err
	}
	return &device.ID, nil
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
