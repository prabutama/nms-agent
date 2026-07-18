package main

import (
	"os"
	"path/filepath"
	"testing"

	"nms-agent/internal/config"
	"nms-agent/internal/models"
)

func writeAgentFiles(t *testing.T, dir, queueDB, thresholdsContent string) string {
	t.Helper()

	devicesDir := filepath.Join(dir, "devices.d")
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll devices.d: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devicesDir, "d1.yml"),
		[]byte("id: dev-1\naddress: 127.0.0.1\nvendor: dummy\nmodel: dummy\n"), 0o644); err != nil {
		t.Fatalf("WriteFile device: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "thresholds.yml"),
		[]byte(thresholdsContent), 0o644); err != nil {
		t.Fatalf("WriteFile thresholds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "adapters.yml"),
		[]byte("adapters:\n  active: terminal\n  configs: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile adapters: %v", err)
	}

	agentYml := filepath.Join(dir, "agent.yml")
	agentContent := "agent:\n  poll_interval: 1s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n  nms_agent_db: " + filepath.ToSlash(queueDB) + "\n"
	if err := os.WriteFile(agentYml, []byte(agentContent), 0o644); err != nil {
		t.Fatalf("WriteFile agent.yml: %v", err)
	}
	return agentYml
}

func TestThresholdSet_AddsNewRule(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	code := runThresholdSet([]string{
		"--config", agentYml,
		"--metric", "test.latency",
		"--operator", ">",
		"--warning", "50",
		"--critical", "100",
		"--tags", "source=ping",
	})
	if code != 0 {
		t.Fatalf("runThresholdSet exit code=%d", code)
	}

	rules, err := loadThresholds(filepath.Join(tmp, "thresholds.yml"))
	if err != nil {
		t.Fatalf("loadThresholds: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Metric != "test.latency" {
		t.Fatalf("metric=%q", rules[0].Metric)
	}
	if rules[0].Warning == nil || *rules[0].Warning != 50 {
		t.Fatalf("warning=%v", rules[0].Warning)
	}
	if rules[0].Critical == nil || *rules[0].Critical != 100 {
		t.Fatalf("critical=%v", rules[0].Critical)
	}
	if rules[0].Tags["source"] != "ping" {
		t.Fatalf("tags=%v", rules[0].Tags)
	}
}

func TestThresholdSet_UpdatesExistingRule(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"),
		"thresholds:\n  - metric: test.latency\n    operator: \">\"\n    warning: 30\n    tags:\n      source: ping\n")

	code := runThresholdSet([]string{
		"--config", agentYml,
		"--metric", "test.latency",
		"--operator", ">",
		"--warning", "80",
		"--critical", "200",
		"--tags", "source=ping",
	})
	if code != 0 {
		t.Fatalf("runThresholdSet exit code=%d", code)
	}

	rules, err := loadThresholds(filepath.Join(tmp, "thresholds.yml"))
	if err != nil {
		t.Fatalf("loadThresholds: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Warning == nil || *rules[0].Warning != 80 {
		t.Fatalf("expected warning=80, got %v", rules[0].Warning)
	}
	if rules[0].Critical == nil || *rules[0].Critical != 200 {
		t.Fatalf("expected critical=200, got %v", rules[0].Critical)
	}
}

func TestThresholdSet_AppendsWhenMetricTagsDiffer(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"),
		"thresholds:\n  - metric: test.latency\n    operator: \">\"\n    warning: 30\n    tags:\n      source: ping\n")

	code := runThresholdSet([]string{
		"--config", agentYml,
		"--metric", "test.latency",
		"--operator", ">",
		"--warning", "50",
		"--tags", "source=icmp",
	})
	if code != 0 {
		t.Fatalf("runThresholdSet exit code=%d", code)
	}

	rules, err := loadThresholds(filepath.Join(tmp, "thresholds.yml"))
	if err != nil {
		t.Fatalf("loadThresholds: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}

func TestThresholdSet_MissingMetricFails(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	code := runThresholdSet([]string{
		"--config", agentYml,
		"--operator", ">",
		"--warning", "50",
	})
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}

func TestThresholdList_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"),
		"thresholds:\n  - metric: test.metric\n    operator: \">\"\n    warning: 50\n    critical: 100\n    tags:\n      source: test\n")

	code := runThresholdList([]string{"--config", agentYml})
	if code != 0 {
		t.Fatalf("runThresholdList exit code=%d", code)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]string
	}{
		{"", nil},
		{"k=v", map[string]string{"k": "v"}},
		{"a=b,c=d", map[string]string{"a": "b", "c": "d"}},
		{"  k=v , x=y ", map[string]string{"k": "v", "x": "y"}},
		{"k=v,=bad", map[string]string{"k": "v"}},
	}
	for _, tt := range tests {
		got := parseTags(tt.input)
		if !tagsEqual(got, tt.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestTagsEqual(t *testing.T) {
	if !tagsEqual(nil, nil) {
		t.Fatal("nil == nil")
	}
	if !tagsEqual(map[string]string{}, map[string]string{}) {
		t.Fatal("empty == empty")
	}
	if !tagsEqual(map[string]string{"a": "b"}, map[string]string{"a": "b"}) {
		t.Fatal("equal maps")
	}
	if tagsEqual(map[string]string{"a": "b"}, map[string]string{"a": "c"}) {
		t.Fatal("different values")
	}
	if tagsEqual(map[string]string{"a": "b"}, map[string]string{"a": "b", "c": "d"}) {
		t.Fatal("different lengths")
	}
}

func TestValidateThresholdRules(t *testing.T) {
	if err := config.ValidateThresholdRules(nil); err != nil {
		t.Fatalf("nil rules should pass: %v", err)
	}
	if err := config.ValidateThresholdRules([]models.ThresholdRule{}); err != nil {
		t.Fatalf("empty rules should pass: %v", err)
	}
	if err := config.ValidateThresholdRules([]models.ThresholdRule{
		{Metric: "m1", Operator: ">", Warning: float64Ptr(50)},
	}); err != nil {
		t.Fatalf("valid rule: %v", err)
	}
	if err := config.ValidateThresholdRules([]models.ThresholdRule{
		{Metric: "", Operator: ">", Warning: float64Ptr(50)},
	}); err == nil {
		t.Fatal("expected error for empty metric")
	}
	if err := config.ValidateThresholdRules([]models.ThresholdRule{
		{Metric: "m1", Operator: "??", Warning: float64Ptr(50)},
	}); err == nil {
		t.Fatal("expected error for invalid operator")
	}
	if err := config.ValidateThresholdRules([]models.ThresholdRule{
		{Metric: "m1", Operator: ">"},
	}); err == nil {
		t.Fatal("expected error for missing warning/critical")
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}
