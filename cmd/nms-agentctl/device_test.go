package main

import (
	"os"
	"path/filepath"
	"testing"

	"nms-agent/internal/config"

	yaml "gopkg.in/yaml.v3"
)

func TestDeviceAdd_WritesNewDeviceFile(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	code := runDeviceAdd([]string{
		"--config", agentYml,
		"--id", "dev-2",
		"--address", "127.0.0.2",
		"--vendor", "linux",
		"--model", "proxmox",
		"--snmp=true",
		"--icmp=false",
	})
	if code != 0 {
		t.Fatalf("runDeviceAdd exit code=%d", code)
	}

	// File should exist under devices.d.
	outPath := filepath.Join(tmp, "devices.d", "dev-2.yml")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected device file, got err: %v", err)
	}
}

func TestDeviceAdd_DuplicateIDFails(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	// dev-1 already exists from writeAgentFiles.
	code := runDeviceAdd([]string{
		"--config", agentYml,
		"--id", "dev-1",
		"--address", "127.0.0.1",
		"--vendor", "dummy",
		"--model", "dummy",
	})
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}

func TestDeviceUpdate_UpdatesFields(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	code := runDeviceUpdate([]string{
		"--config", agentYml,
		"--id", "dev-1",
		"--address", "127.0.0.9",
		"--vendor", "linux",
		"--model", "proxmox",
		"--snmp", "true",
		"--icmp", "false",
	})
	if code != 0 {
		t.Fatalf("runDeviceUpdate exit code=%d", code)
	}

	// Find the device file and verify content.
	devicesDir := filepath.Join(tmp, "devices.d")
	path, err := findDeviceFileByID(devicesDir, "dev-1")
	if err != nil {
		t.Fatalf("findDeviceFileByID: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var d config.Device
	if err := yaml.Unmarshal(b, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.Address != "127.0.0.9" || d.Vendor != "linux" || d.Model != "proxmox" {
		t.Fatalf("unexpected fields: %+v", d)
	}
	if !d.SNMP.Enabled || d.ICMP.Enabled {
		t.Fatalf("unexpected flags: snmp=%v icmp=%v", d.SNMP.Enabled, d.ICMP.Enabled)
	}
}

func TestDeviceRemove_DeletesFile(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	// Ensure device exists.
	devicesDir := filepath.Join(tmp, "devices.d")
	path, err := findDeviceFileByID(devicesDir, "dev-1")
	if err != nil {
		t.Fatalf("findDeviceFileByID: %v", err)
	}

	code := runDeviceRemove([]string{"--config", agentYml, "--id", "dev-1"})
	if code != 0 {
		t.Fatalf("runDeviceRemove exit code=%d", code)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected file removed")
	}
}
