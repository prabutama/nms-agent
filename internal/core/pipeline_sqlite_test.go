package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"nms-agent/internal/models"
	"nms-agent/internal/processors"
	"nms-agent/internal/queue"
)

type testCollector struct{}

func (testCollector) Collect(context.Context) ([]models.RawSample, error) {
	return []models.RawSample{{
		DeviceID: "d1",
		Source:   "dummy",
		TS:       time.Now().UTC(),
		Fields: map[string]any{
			"metric":       "demo.ping",
			"value_type":   "number",
			"value_number": 7.0,
			"unit":         "ms",
		},
	}}, nil
}

type testProcessor struct{}

func (testProcessor) Normalize(context.Context, []models.RawSample) ([]models.Telemetry, error) {
	return []models.Telemetry{{
		DeviceID:    "d1",
		Metric:      "demo.ping",
		TS:          time.Now().UTC(),
		ValueType:   "number",
		ValueNumber: floatPtr(7.0),
		Tags:        map[string]string{"unit": "ms"},
	}}, nil
}

type failingAdapter struct{}

func (failingAdapter) SendBatch(context.Context, []models.Telemetry) error {
	return errors.New("send failed")
}

func TestPipeline_EnqueueBeforeSend_SQLite(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	p := NewPipeline(testCollector{}, testProcessor{}, q, failingAdapter{}, DeliveryConfig{MaxBatch: 10})
	if err := p.RunOnce(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	items, err := q.PendingBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 queued item after send failure, got %d", len(items))
	}
	if items[0].RetryCount != 1 {
		t.Fatalf("expected retry_count=1 after MarkFailed, got %d", items[0].RetryCount)
	}

	// Prove persistence after restart.
	if err := q.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	q2, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite restart: %v", err)
	}
	defer q2.Close()
	items2, err := q2.PendingBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("PendingBatch restart: %v", err)
	}
	if len(items2) != 1 {
		t.Fatalf("expected 1 queued item after restart, got %d", len(items2))
	}
	if items2[0].RetryCount != 1 {
		t.Fatalf("expected retry_count=1 after restart, got %d", items2[0].RetryCount)
	}
}

type capturingAdapter struct {
	sent []models.Telemetry
}

func (a *capturingAdapter) SendBatch(_ context.Context, batch []models.Telemetry) error {
	a.sent = append(a.sent, batch...)
	return nil
}

// realCollector emits a raw sample that triggers normalization + threshold.
type realCollector struct {
	fields map[string]any
}

func (c realCollector) Collect(_ context.Context) ([]models.RawSample, error) {
	return []models.RawSample{{
		DeviceID: "d1",
		Source:   "snmp",
		TS:       time.Now().UTC(),
		Fields:   c.fields,
	}}, nil
}

func TestPipelineE2E_ThresholdAndNormalizationFlow(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	rules := []models.ThresholdRule{
		{
			Metric:   "snmp.if.rx_utilization_pct",
			Operator: ">",
			Warning:  floatPtr(70),
			Critical: floatPtr(95),
			Tags:     map[string]string{"ifIndex": "1"},
		},
		{
			Metric:   "icmp.latency_ms",
			Operator: ">",
			Warning:  floatPtr(50),
			Tags:     map[string]string{"source": "icmp"},
		},
	}

	proc := &processors.PreprocessThresholdProcessor{Rules: rules}
	adapter := &capturingAdapter{}

	// Send a utilization value that exceeds 100% — should be clamped to 100 and trigger critical.
	coll := realCollector{fields: map[string]any{
		"metric":       "snmp.if.rx_utilization_pct",
		"value_type":   "number",
		"value_number": 200.0,
		"tags":         map[string]string{"ifIndex": "1"},
	}}

	p := NewPipeline(coll, proc, q, adapter, DeliveryConfig{MaxBatch: 10})
	if err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(adapter.sent) == 0 {
		t.Fatalf("expected at least 1 item sent to adapter")
	}

	item := adapter.sent[0]
	if item.Metric != "snmp.if.rx_utilization_pct" {
		t.Fatalf("expected rx_utilization_pct, got %s", item.Metric)
	}
	if item.ValueNumber == nil {
		t.Fatalf("expected non-nil value")
	}
	if *item.ValueNumber != 100.0 {
		t.Fatalf("expected clamped 100.0, got %v", *item.ValueNumber)
	}
	if item.Tags["unit"] != "pct" {
		t.Fatalf("expected unit=pct, got %q", item.Tags["unit"])
	}
	if item.Tags["threshold.status"] != "critical" {
		t.Fatalf("expected threshold.status=critical, got %q", item.Tags["threshold.status"])
	}
	if item.Tags["threshold.matched"] != "true" {
		t.Fatalf("expected threshold.matched=true")
	}
}

func TestPipelineE2E_NormalizationDoesNotBreakQueuePersistence(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	proc := &processors.PreprocessThresholdProcessor{}
	adapter := &capturingAdapter{}

	// String metrics should pass through unaffected.
	coll := realCollector{fields: map[string]any{
		"metric":       "snmp.system.description",
		"value_type":   "string",
		"value_string": "RouterOS CHR",
	}}

	p := NewPipeline(coll, proc, q, adapter, DeliveryConfig{MaxBatch: 10})
	if err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(adapter.sent) != 1 {
		t.Fatalf("expected 1 item, got %d", len(adapter.sent))
	}
	if adapter.sent[0].ValueString == nil || *adapter.sent[0].ValueString != "RouterOS CHR" {
		t.Fatalf("string value should be preserved through normalization")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
