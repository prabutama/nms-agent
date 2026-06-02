package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nms-agent/internal/models"
	"nms-agent/internal/queue"
)

func TestQueueRetry_DeliversAndDeletesPending(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "queue.db")

	q, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	// Seed one pending item.
	tel := []models.Telemetry{{
		TS:          time.Now().UTC(),
		DeviceID:    "dev-1",
		Metric:      "m1",
		ValueType:   "number",
		ValueNumber: floatPtr(1),
		Tags:        map[string]string{"k": "v"},
	}}
	if err := q.EnqueueBatch(context.Background(), tel); err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}

	// Minimal config files for loader/validator.
	devicesDir := filepath.Join(tmp, "devices.d")
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devicesDir, "d1.yml"), []byte("id: dev-1\naddress: 127.0.0.1\nvendor: dummy\nmodel: dummy\n"), 0o644); err != nil {
		t.Fatalf("WriteFile device: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "thresholds.yml"), []byte("thresholds: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile thresholds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "adapters.yml"), []byte("adapters:\n  active: tui\n  configs: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile adapters: %v", err)
	}
	agentYml := filepath.Join(tmp, "agent.yml")
	agentContent := "agent:\n  poll_interval: 1s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n  queue_db: " + filepath.ToSlash(dbPath) + "\n"
	if err := os.WriteFile(agentYml, []byte(agentContent), 0o644); err != nil {
		t.Fatalf("WriteFile agent.yml: %v", err)
	}

	// Run retry.
	code := runQueueRetry([]string{"--config", agentYml, "--limit", "10"})
	if code != 0 {
		t.Fatalf("runQueueRetry exit code=%d", code)
	}

	items, err := q.PendingBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("PendingBatch: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected queue empty, got %d", len(items))
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
