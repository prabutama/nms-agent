package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	"nms-agent/internal/profiles"
)

type cliFakeProvider struct{ candidates []discovery.Candidate }

func (p cliFakeProvider) Candidates(ctx context.Context, loaded config.Loaded) ([]discovery.Candidate, error) {
	_ = ctx
	_ = loaded
	return p.candidates, nil
}

type cliFakeProber struct {
	byAddress map[string]discovery.Fingerprint
}

func (p cliFakeProber) Probe(ctx context.Context, candidate discovery.Candidate, cfg config.DiscoverySNMP) (discovery.Fingerprint, error) {
	_ = ctx
	_ = cfg
	if fp, ok := p.byAddress[candidate.Address]; ok {
		fp.Candidate = candidate
		return fp, nil
	}
	return discovery.Fingerprint{Candidate: candidate}, nil
}

func TestDiscoveryPreview_DoesNotWriteDevice(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	oldFactory := newDiscoveryService
	defer func() { newDiscoveryService = oldFactory }()
	newDiscoveryService = func() discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			}},
		}
	}
	code := runDiscoveryPreview([]string{"--config", agentYml})
	if code != 0 {
		t.Fatalf("runDiscoveryPreview exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(tmp, "devices.d", "linux-server-a.yml")); err == nil {
		t.Fatalf("preview should not write device file")
	}
}

func TestDiscoveryRun_WritesDevice(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	oldFactory := newDiscoveryService
	defer func() { newDiscoveryService = oldFactory }()
	newDiscoveryService = func() discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			}},
		}
	}
	code := runDiscoveryRun([]string{"--config", agentYml})
	if code != 0 {
		t.Fatalf("runDiscoveryRun exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(tmp, "devices.d", "linux-server-a.yml")); err != nil {
		t.Fatalf("expected promoted file: %v", err)
	}
}

func TestDiscoveryStatus_PrintsConfig(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	code := runDiscoveryStatus([]string{"--config", agentYml})
	if code != 0 {
		t.Fatalf("runDiscoveryStatus exit code=%d", code)
	}
}

func writeDiscoveryAgentFiles(t *testing.T, dir string) string {
	t.Helper()
	devicesDir := filepath.Join(dir, "devices.d")
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "thresholds.yml"), []byte("thresholds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "adapters.yml"), []byte("adapters:\n  active: tui\n  configs: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDiscoveryProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	writeDiscoveryProfile(t, filepath.Join(profilesDir, "linux.yml"), "linux-default", "linux", "")
	agentContent := "agent:\n  poll_interval: 1s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n  profiles_dir: profiles\n  queue_db: queue.db\ndiscovery:\n  enabled: true\n  interval: 10m\n  interface: eth0\n  subnet: 192.168.10.0/24\n  provider: netlink\n  snmp:\n    version: v2c\n    community: public\n    timeout: 2s\n    retries: 1\n    concurrency: 4\n  auto_promote:\n    enabled: true\n    require_snmp_ok: true\n    require_sys_object_id: true\n    require_profile_match: true\n    max_new_devices_per_cycle: 10\n    device_id_template: \"{{vendor}}-{{sys_name}}\"\n    write_to: devices.d\n  exploration:\n    enabled: false\n    run_when: no_profile_match\n    safe_only: true\n    auto_approve_generated_profile: true\n    auto_promote_after_generate: true\n    max_oids_per_device: 100\n    timeout: 3s\n    output_dir: profiles\n"
	agentYml := filepath.Join(dir, "agent.yml")
	if err := os.WriteFile(agentYml, []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}
	return agentYml
}

func writeDiscoveryProfile(t *testing.T, path, name, vendor, model string) {
	t.Helper()
	p := profiles.Profile{
		Name:    name,
		Match:   profiles.Match{Vendor: vendor, Model: model},
		Metrics: []profiles.Metric{{Metric: "snmp.system.name", OID: "1.3.6.1.2.1.1.5.0", Type: "get"}},
	}
	b := []byte("name: " + p.Name + "\nmatch:\n  vendor: \"" + p.Match.Vendor + "\"\n  model: \"" + p.Match.Model + "\"\nmetrics:\n  - metric: snmp.system.name\n    oid: 1.3.6.1.2.1.1.5.0\n    type: get\n")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
