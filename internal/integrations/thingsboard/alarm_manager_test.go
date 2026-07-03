package thingsboard

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"nms-agent/internal/models"
)

type fakeAlarmClient struct {
	deviceByName       map[string]*DeviceInfo
	alarmsByEntity     map[string][]Alarm
	createCalls        []AlarmRequest
	clearCalls         []string
	getDeviceByNameHit int
	createErr          error
	clearErr           error
	getAlarmsErr       error
	nextAlarmID        int
}

func newFakeAlarmClient() *fakeAlarmClient {
	return &fakeAlarmClient{
		deviceByName:   map[string]*DeviceInfo{},
		alarmsByEntity: map[string][]Alarm{},
		nextAlarmID:    1,
	}
}

func (f *fakeAlarmClient) CreateAlarm(_ context.Context, alarm AlarmRequest) (*Alarm, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	id := fmt.Sprintf("alarm-%d", f.nextAlarmID)
	f.nextAlarmID++
	f.createCalls = append(f.createCalls, alarm)
	created := Alarm{
		ID:         EntityRef{EntityType: "ALARM", ID: id},
		Type:       alarm.Type,
		Severity:   alarm.Severity,
		Originator: alarm.Originator,
	}
	entityKey := alarm.Originator.EntityType + ":" + alarm.Originator.ID
	f.alarmsByEntity[entityKey] = append(f.alarmsByEntity[entityKey], created)
	return &created, nil
}

func (f *fakeAlarmClient) ClearAlarm(_ context.Context, alarmID string) error {
	if f.clearErr != nil {
		return f.clearErr
	}
	f.clearCalls = append(f.clearCalls, alarmID)
	for entityKey, alarms := range f.alarmsByEntity {
		for i := range alarms {
			if alarms[i].ID.ID == alarmID {
				alarms[i].Cleared = true
			}
		}
		f.alarmsByEntity[entityKey] = alarms
	}
	return nil
}

func (f *fakeAlarmClient) GetAlarmsByEntity(_ context.Context, entityType, entityID string) ([]Alarm, error) {
	if f.getAlarmsErr != nil {
		return nil, f.getAlarmsErr
	}
	entityKey := entityType + ":" + entityID
	alarms := f.alarmsByEntity[entityKey]
	out := make([]Alarm, len(alarms))
	copy(out, alarms)
	return out, nil
}

func (f *fakeAlarmClient) GetDeviceByName(_ context.Context, name string) (*DeviceInfo, error) {
	f.getDeviceByNameHit++
	device, ok := f.deviceByName[name]
	if !ok {
		return nil, errors.New("device not found")
	}
	return device, nil
}

func TestAlarmManager_DedupSameSeverity(t *testing.T) {
	cli := newFakeAlarmClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	m := &AlarmManager{client: cli, cache: map[string]*alarmState{}, originatorCache: map[string]EntityRef{}}
	tel := thresholdTelemetry("router-1", "icmp.latency_ms", "warning")

	if err := m.ProcessBatch(context.Background(), []models.Telemetry{tel}); err != nil {
		t.Fatalf("first ProcessBatch: %v", err)
	}
	if err := m.ProcessBatch(context.Background(), []models.Telemetry{tel}); err != nil {
		t.Fatalf("second ProcessBatch: %v", err)
	}

	if got := len(cli.createCalls); got != 1 {
		t.Fatalf("expected 1 create call, got %d", got)
	}
	if got := len(cli.clearCalls); got != 0 {
		t.Fatalf("expected 0 clear calls, got %d", got)
	}
	if cli.getDeviceByNameHit != 1 {
		t.Fatalf("expected originator cache to limit lookup to 1, got %d", cli.getDeviceByNameHit)
	}
}

func TestAlarmManager_SeverityChangeClearsThenCreates(t *testing.T) {
	cli := newFakeAlarmClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	m := &AlarmManager{client: cli, cache: map[string]*alarmState{}, originatorCache: map[string]EntityRef{}}

	if err := m.ProcessBatch(context.Background(), []models.Telemetry{thresholdTelemetry("router-1", "icmp.latency_ms", "warning")}); err != nil {
		t.Fatalf("warning ProcessBatch: %v", err)
	}
	if err := m.ProcessBatch(context.Background(), []models.Telemetry{thresholdTelemetry("router-1", "icmp.latency_ms", "critical")}); err != nil {
		t.Fatalf("critical ProcessBatch: %v", err)
	}

	if got := len(cli.createCalls); got != 2 {
		t.Fatalf("expected 2 create calls, got %d", got)
	}
	if got := len(cli.clearCalls); got != 1 {
		t.Fatalf("expected 1 clear call, got %d", got)
	}
	if cli.clearCalls[0] != "alarm-1" {
		t.Fatalf("expected first alarm cleared, got %q", cli.clearCalls[0])
	}
}

func TestAlarmManager_ClearOnCacheMissUsesRemoteAlarm(t *testing.T) {
	cli := newFakeAlarmClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	cli.alarmsByEntity["DEVICE:dev-1"] = []Alarm{{
		ID:         EntityRef{EntityType: "ALARM", ID: "alarm-remote"},
		Type:       "HIGH_LATENCY",
		Severity:   "WARNING",
		Cleared:    false,
		Originator: EntityRef{EntityType: "DEVICE", ID: "dev-1"},
	}}
	m := &AlarmManager{client: cli, cache: map[string]*alarmState{}, originatorCache: map[string]EntityRef{}}

	if err := m.ProcessBatch(context.Background(), []models.Telemetry{thresholdTelemetry("router-1", "icmp.latency_ms", "ok")}); err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}

	if got := len(cli.clearCalls); got != 1 {
		t.Fatalf("expected 1 clear call, got %d", got)
	}
	if cli.clearCalls[0] != "alarm-remote" {
		t.Fatalf("expected remote alarm cleared, got %q", cli.clearCalls[0])
	}
}

func TestAlarmManager_ReusesRemoteActiveAlarmOnCacheMiss(t *testing.T) {
	cli := newFakeAlarmClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	cli.alarmsByEntity["DEVICE:dev-1"] = []Alarm{{
		ID:         EntityRef{EntityType: "ALARM", ID: "alarm-remote"},
		Type:       "HIGH_LATENCY",
		Severity:   "WARNING",
		Cleared:    false,
		Originator: EntityRef{EntityType: "DEVICE", ID: "dev-1"},
	}}
	m := &AlarmManager{client: cli, cache: map[string]*alarmState{}, originatorCache: map[string]EntityRef{}}

	if err := m.ProcessBatch(context.Background(), []models.Telemetry{thresholdTelemetry("router-1", "icmp.latency_ms", "warning")}); err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}

	if got := len(cli.createCalls); got != 0 {
		t.Fatalf("expected 0 create calls, got %d", got)
	}
	if got := len(cli.clearCalls); got != 0 {
		t.Fatalf("expected 0 clear calls, got %d", got)
	}
}

func thresholdTelemetry(deviceID, metric, status string) models.Telemetry {
	v := 123.0
	return models.Telemetry{
		DeviceID:    deviceID,
		Metric:      metric,
		TS:          time.Now().UTC(),
		ValueType:   "number",
		ValueNumber: &v,
		Tags: map[string]string{
			"threshold.status":  status,
			"threshold.rule":    metric + "#0",
			"threshold.matched": "true",
		},
	}
}
