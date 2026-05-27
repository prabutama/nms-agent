package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"nms-agent/internal/models"
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

	p := NewPipeline(testCollector{}, testProcessor{}, q, failingAdapter{})
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

func floatPtr(v float64) *float64 {
	return &v
}
