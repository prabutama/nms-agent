package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile_ResolvesRelativePaths(t *testing.T) {
	tmp := t.TempDir()

	configsDir := filepath.Join(tmp, "configs")
	devicesDir := filepath.Join(configsDir, "devices.d")
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(configsDir, "thresholds.yml"), []byte("thresholds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configsDir, "adapters.yml"), []byte("adapters:\n  active: terminal\n  configs: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devicesDir, "dev1.yml"), []byte("id: d1\naddress: 127.0.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentYML := []byte("agent:\n  poll_interval: 60s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n")
	if err := os.WriteFile(filepath.Join(configsDir, "agent.yml"), agentYML, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(filepath.Join(configsDir, "agent.yml"))
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	if got := len(cfg.Devices); got != 1 {
		t.Fatalf("expected 1 device, got %d", got)
	}
}

func TestLoadFromFile_ExpandsAdapterEnvVars(t *testing.T) {
	t.Setenv("TB_URL", "https://example.com")
	t.Setenv("TB_API_KEY_BR_B", "secret-key")
	tmp := t.TempDir()
	configsDir := filepath.Join(tmp, "configs")
	devicesDir := filepath.Join(configsDir, "devices.d")
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configsDir, "thresholds.yml"), []byte("thresholds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	adaptersYML := []byte("adapters:\n  active: thingsboard_mqtt\n  configs:\n    broker: tcp://127.0.0.1:1883\n    thingsboard:\n      api:\n        base_url: ${TB_URL}\n        api_key: ${TB_API_KEY_BR_B}\n")
	if err := os.WriteFile(filepath.Join(configsDir, "adapters.yml"), adaptersYML, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devicesDir, "dev1.yml"), []byte("id: d1\naddress: 127.0.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentYML := []byte("agent:\n  poll_interval: 60s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n")
	if err := os.WriteFile(filepath.Join(configsDir, "agent.yml"), agentYML, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromFile(filepath.Join(configsDir, "agent.yml"))
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	tb := cfg.Adapters.Adapters.Configs["thingsboard"].(map[string]any)
	api := tb["api"].(map[string]any)
	if api["base_url"] != "https://example.com" || api["api_key"] != "secret-key" {
		t.Fatalf("unexpected expanded adapter env vars: %+v", api)
	}
}
