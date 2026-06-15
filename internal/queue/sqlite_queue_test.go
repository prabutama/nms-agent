package queue

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"nms-agent/internal/models"
)

func TestSQLiteQueue_PersistsAcrossRestart(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q1, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	ctx := context.Background()
	telemetry := []models.Telemetry{{
		DeviceID:    "d1",
		Metric:      "demo.ping",
		TS:          time.Now().UTC(),
		ValueType:   "number",
		ValueNumber: floatPtr(1.23),
		Tags:        map[string]string{"unit": "ms"},
	}}
	if err := q1.EnqueueBatch(ctx, telemetry); err != nil {
		_ = q1.Close()
		t.Fatalf("EnqueueBatch: %v", err)
	}
	if err := q1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	q2, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite restart: %v", err)
	}
	defer q2.Close()

	items, err := q2.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Telemetry.DeviceID != "d1" {
		t.Fatalf("unexpected device id: %s", items[0].Telemetry.DeviceID)
	}

	if err := q2.MarkDelivered(ctx, []string{items[0].ID}); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}
	items, err = q2.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch after delivered: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestSQLiteQueue_MarkFailedIncrementsRetry(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	if err := q.EnqueueBatch(ctx, []models.Telemetry{{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(1)}}); err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}

	items, err := q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].RetryCount != 0 {
		t.Fatalf("expected retry 0, got %d", items[0].RetryCount)
	}

	if err := q.MarkFailed(ctx, []string{items[0].ID}, "boom"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	items2, err := q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items2) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items2))
	}
	if items2[0].RetryCount != 1 {
		t.Fatalf("expected retry 1, got %d", items2[0].RetryCount)
	}
}

func TestSQLiteQueue_MarkDeliveredDeletesMultipleIDs(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	if err := q.EnqueueBatch(ctx, telemetryBatch(3)); err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}

	items, err := q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	if err := q.MarkDelivered(ctx, ids); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}
	items, err = q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch after delivered: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestSQLiteQueue_MarkFailedIncrementsMultipleIDs(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	if err := q.EnqueueBatch(ctx, telemetryBatch(3)); err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}

	items, err := q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	if err := q.MarkFailed(ctx, ids, "boom"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	items, err = q.PendingBatch(ctx, 10)
	if err != nil {
		t.Fatalf("PendingBatch after failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	for _, item := range items {
		if item.RetryCount != 1 {
			t.Fatalf("expected retry 1, got %d", item.RetryCount)
		}
	}
}

func TestSQLiteQueue_EnqueueBatch_GeneratesUniqueIDs(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	batch := make([]models.Telemetry, 0, 200)
	for i := 0; i < 200; i++ {
		batch = append(batch, models.Telemetry{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(float64(i))})
	}
	if err := q.EnqueueBatch(ctx, batch); err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}

	items, err := q.PendingBatch(ctx, 300)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items) != 200 {
		t.Fatalf("expected 200 items, got %d", len(items))
	}
	seen := map[string]struct{}{}
	for _, it := range items {
		if _, ok := seen[it.ID]; ok {
			t.Fatalf("duplicate id: %s", it.ID)
		}
		seen[it.ID] = struct{}{}
	}
}

func telemetryBatch(n int) []models.Telemetry {
	batch := make([]models.Telemetry, 0, n)
	for i := 0; i < n; i++ {
		batch = append(batch, models.Telemetry{DeviceID: "d1", Metric: "m", TS: time.Now().UTC(), ValueType: "number", ValueNumber: floatPtr(float64(i))})
	}
	return batch
}

func floatPtr(v float64) *float64 {
	return &v
}
