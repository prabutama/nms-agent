package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nms-agent/internal/config"
	"nms-agent/internal/queue"

	yaml "gopkg.in/yaml.v3"
)

func TestDeviceListShowsMaskedThingsBoardToken(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "nms-agent.db")
	agentYml := writeAgentFiles(t, tmp, dbPath, "thresholds: []\n")
	q, err := queue.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	if err := q.SaveThingsBoardToken(context.Background(), "dev-1", "abcd1234wxyz"); err != nil {
		t.Fatalf("SaveThingsBoardToken: %v", err)
	}
	q.Close()

	out := captureStdout(t, func() {
		code := runDeviceList([]string{"--config", agentYml})
		if code != 0 {
			t.Fatalf("runDeviceList exit code=%d", code)
		}
	})
	if !strings.Contains(out, "tb_token") {
		t.Fatalf("expected tb_token header, got %q", out)
	}
	if !strings.Contains(out, "abcd...wxyz") {
		t.Fatalf("expected masked token, got %q", out)
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "-"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"  abcdefghijkl  ", "abcd...ijkl"},
	}
	for _, tt := range tests {
		if got := maskToken(tt.in); got != tt.want {
			t.Fatalf("maskToken(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

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

func TestSanitizeInput_CleansHiddenChars(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"trailing CR", "172.16.30.1\r", "172.16.30.1"},
		{"leading space", "  172.16.30.1  ", "172.16.30.1"},
		{"tab in middle", "172.16\t30.1", "172.16 30.1"},
		{"control char", "172.16\x01.1", "172.16.1"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeInput(tt.input)
			if got != tt.expect {
				t.Fatalf("sanitizeInput(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestValidateDeviceID(t *testing.T) {
	if err := validateDeviceID("valid-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateDeviceID(""); err == nil {
		t.Fatal("expected error for empty id")
	}
	if err := validateDeviceID("bad id"); err == nil {
		t.Fatal("expected error for id with space")
	}
	if err := validateDeviceID("bad@id"); err == nil {
		t.Fatal("expected error for id with special char")
	}
}

func TestValidateAddress(t *testing.T) {
	if err := validateAddress("127.0.0.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateAddress("localhost"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateAddress(""); err == nil {
		t.Fatal("expected error for empty address")
	}
	if err := validateAddress("bad\x01address"); err == nil {
		t.Fatal("expected error for address with control char")
	}
}

func TestValidateVendorModel(t *testing.T) {
	if err := validateVendorModel("linux", "ubuntu"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateVendorModel("", "ubuntu"); err == nil {
		t.Fatal("expected error for empty vendor")
	}
	if err := validateVendorModel("linux", ""); err == nil {
		t.Fatal("expected error for empty model")
	}
	if err := validateVendorModel("linux\x01", "ubuntu"); err == nil {
		t.Fatal("expected error for vendor with control char")
	}
}
