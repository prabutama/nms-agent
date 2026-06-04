package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nms-agent/internal/config"
	"nms-agent/internal/profiles"
)

type fakeProvider struct{ candidates []Candidate }

func (p fakeProvider) Candidates(ctx context.Context, loaded config.Loaded) ([]Candidate, error) {
	_ = ctx
	_ = loaded
	return p.candidates, nil
}

type fakeProber struct{ byAddress map[string]Fingerprint }

func (p fakeProber) Probe(ctx context.Context, candidate Candidate, cfg config.DiscoverySNMP) (Fingerprint, error) {
	_ = ctx
	_ = cfg
	if fp, ok := p.byAddress[candidate.Address]; ok {
		fp.Candidate = candidate
		return fp, nil
	}
	return Fingerprint{Candidate: candidate}, nil
}

type fakeExplorer struct{ byAddress map[string]ExplorationResult }

func (e fakeExplorer) Explore(ctx context.Context, configPath string, loaded config.Loaded, fp Fingerprint) (ExplorationResult, error) {
	_ = ctx
	_ = configPath
	_ = loaded
	if res, ok := e.byAddress[fp.Address]; ok {
		return res, nil
	}
	return ExplorationResult{}, nil
}

func TestServiceRunOnce_PromotesKnownProfile(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	devicesDir := filepath.Join(tmp, "devices.d")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(devicesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	writeProfile(t, filepath.Join(profilesDir, "linux.yml"), "linux-default", "linux", "")

	loaded := config.Loaded{
		Root: config.Root{
			Agent: config.Agent{PollInterval: time.Second},
			Paths: config.Paths{DevicesDir: "devices.d", ThresholdsFile: "thresholds.yml", AdaptersFile: "adapters.yml", QueueDB: "queue.db"},
			Discovery: config.Discovery{
				Enabled:     true,
				Interval:    time.Minute,
				Interface:   "eth0",
				Subnet:      "192.168.10.0/24",
				Provider:    "netlink",
				SNMP:        config.DiscoverySNMP{Version: "v2c", Community: "public", Timeout: time.Second, Retries: 1, Concurrency: 4},
				AutoPromote: config.DiscoveryAutoPromote{Enabled: true, RequireSNMPOK: true, RequireSysObjectID: true, RequireProfileMatch: true, MaxNewDevicesPerCycle: 10, DeviceIDTemplate: "{{vendor}}-{{sys_name}}", WriteTo: "devices.d"},
			},
		},
		ProfilesDir: profilesDir,
	}
	service := Service{
		Provider: fakeProvider{candidates: []Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
		Prober: fakeProber{byAddress: map[string]Fingerprint{
			"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
		}},
	}
	configPath := filepath.Join(tmp, "agent.yml")
	res, err := service.RunOnce(context.Background(), configPath, loaded)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if !res.Changed || res.Promoted != 1 || res.ProfileMatched != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(devicesDir, "linux-server-a.yml")); err != nil {
		t.Fatalf("expected promoted file: %v", err)
	}
}

func TestServiceRunOnce_AppendsCollisionSuffix(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	writeProfile(t, filepath.Join(profilesDir, "linux.yml"), "linux-default", "linux", "")
	loaded := config.Loaded{
		Root: config.Root{
			Agent: config.Agent{PollInterval: time.Second},
			Paths: config.Paths{DevicesDir: "devices.d", ThresholdsFile: "thresholds.yml", AdaptersFile: "adapters.yml", QueueDB: "queue.db"},
			Discovery: config.Discovery{
				Enabled:     true,
				Interval:    time.Minute,
				Interface:   "eth0",
				Subnet:      "192.168.10.0/24",
				Provider:    "netlink",
				SNMP:        config.DiscoverySNMP{Version: "v2c", Community: "public", Timeout: time.Second, Retries: 1, Concurrency: 2},
				AutoPromote: config.DiscoveryAutoPromote{Enabled: true, RequireSNMPOK: true, RequireSysObjectID: true, RequireProfileMatch: true, MaxNewDevicesPerCycle: 10, DeviceIDTemplate: "{{vendor}}-{{sys_name}}", WriteTo: "devices.d"},
			},
		},
		Devices:     []config.Device{{ID: "linux-server-a", Address: "192.168.10.5", Vendor: "linux", Model: ""}},
		ProfilesDir: profilesDir,
	}
	service := Service{
		Provider: fakeProvider{candidates: []Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
		Prober: fakeProber{byAddress: map[string]Fingerprint{
			"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
		}},
	}
	configPath := filepath.Join(tmp, "agent.yml")
	res, err := service.RunOnce(context.Background(), configPath, loaded)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if res.Promoted != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(tmp, "devices.d", "linux-server-a-2.yml")); err != nil {
		t.Fatalf("expected collision suffixed file: %v", err)
	}
}

func TestServiceRunOnce_RespectsPromotionLimit(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	writeProfile(t, filepath.Join(profilesDir, "linux.yml"), "linux-default", "linux", "")
	loaded := config.Loaded{
		Root: config.Root{
			Agent: config.Agent{PollInterval: time.Second},
			Paths: config.Paths{DevicesDir: "devices.d", ThresholdsFile: "thresholds.yml", AdaptersFile: "adapters.yml", QueueDB: "queue.db"},
			Discovery: config.Discovery{
				Enabled:     true,
				Interval:    time.Minute,
				Interface:   "eth0",
				Subnet:      "192.168.10.0/24",
				Provider:    "netlink",
				SNMP:        config.DiscoverySNMP{Version: "v2c", Community: "public", Timeout: time.Second, Retries: 1, Concurrency: 2},
				AutoPromote: config.DiscoveryAutoPromote{Enabled: true, RequireSNMPOK: true, RequireSysObjectID: true, RequireProfileMatch: true, MaxNewDevicesPerCycle: 1, DeviceIDTemplate: "{{vendor}}-{{sys_name}}", WriteTo: "devices.d"},
			},
		},
		ProfilesDir: profilesDir,
	}
	service := Service{
		Provider: fakeProvider{candidates: []Candidate{{Address: "192.168.10.10"}, {Address: "192.168.10.11"}}},
		Prober: fakeProber{byAddress: map[string]Fingerprint{
			"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			"192.168.10.11": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-b", SysDescr: "Linux ubuntu"},
		}},
	}
	res, err := service.RunOnce(context.Background(), filepath.Join(tmp, "agent.yml"), loaded)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if res.Promoted != 1 {
		t.Fatalf("expected 1 promoted, got %+v", res)
	}
}

func TestServiceRunOnce_DoesNotTreatStandardAsProfileMatch(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	loaded := config.Loaded{
		Root: config.Root{
			Agent: config.Agent{PollInterval: time.Second},
			Paths: config.Paths{DevicesDir: "devices.d", ThresholdsFile: "thresholds.yml", AdaptersFile: "adapters.yml", QueueDB: "queue.db"},
			Discovery: config.Discovery{
				Enabled:     true,
				Interval:    time.Minute,
				Interface:   "eth0",
				Subnet:      "192.168.10.0/24",
				Provider:    "netlink",
				SNMP:        config.DiscoverySNMP{Version: "v2c", Community: "public", Timeout: time.Second, Retries: 1, Concurrency: 1},
				AutoPromote: config.DiscoveryAutoPromote{Enabled: true, RequireSNMPOK: true, RequireSysObjectID: true, RequireProfileMatch: true, MaxNewDevicesPerCycle: 10, DeviceIDTemplate: "{{vendor}}-{{sys_name}}", WriteTo: "devices.d"},
			},
		},
		ProfilesDir: profilesDir,
	}
	service := Service{
		Provider: fakeProvider{candidates: []Candidate{{Address: "192.168.10.10"}}},
		Prober: fakeProber{byAddress: map[string]Fingerprint{
			"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.99999.1", SysName: "mystery", SysDescr: "Unknown appliance"},
		}},
	}
	res, err := service.RunOnce(context.Background(), filepath.Join(tmp, "agent.yml"), loaded)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if res.Promoted != 0 || res.ProfileMatched != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestServiceRunOnce_ExplorationGeneratesProfileAndPromotes(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, filepath.Join(profilesDir, "standard.yml"), "standard", "", "")
	loaded := config.Loaded{
		Root: config.Root{
			Agent: config.Agent{PollInterval: time.Second},
			Paths: config.Paths{DevicesDir: "devices.d", ThresholdsFile: "thresholds.yml", AdaptersFile: "adapters.yml", QueueDB: "queue.db"},
			Discovery: config.Discovery{
				Enabled:     true,
				Interval:    time.Minute,
				Interface:   "eth0",
				Subnet:      "192.168.10.0/24",
				Provider:    "netlink",
				SNMP:        config.DiscoverySNMP{Version: "v2c", Community: "public", Timeout: time.Second, Retries: 1, Concurrency: 1},
				AutoPromote: config.DiscoveryAutoPromote{Enabled: true, RequireSNMPOK: true, RequireSysObjectID: true, RequireProfileMatch: true, MaxNewDevicesPerCycle: 10, DeviceIDTemplate: "{{vendor}}-{{sys_name}}", WriteTo: "devices.d"},
				Exploration: config.DiscoveryExploration{Enabled: true, RunWhen: "no_profile_match", AutoApproveGeneratedProfile: true, AutoPromoteAfterGenerate: true, OutputDir: "profiles", Timeout: time.Second, MaxOIDsPerDevice: 10},
			},
		},
		ProfilesDir: profilesDir,
	}
	service := Service{
		Provider: fakeProvider{candidates: []Candidate{{Address: "192.168.10.20"}}},
		Prober: fakeProber{byAddress: map[string]Fingerprint{
			"192.168.10.20": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.99999.42", SysName: "mystery-a", SysDescr: "Unknown appliance"},
		}},
		Explorer: fakeExplorer{byAddress: map[string]ExplorationResult{
			"192.168.10.20": {
				Generated: true,
				Vendor:    "discovered",
				Model:     "sysobj-1-3-6-1-4-1-99999-42",
				Profile: profiles.Profile{
					Name:    "generated-sysobj-1-3-6-1-4-1-99999-42",
					Match:   profiles.Match{Vendor: "discovered", Model: "sysobj-1-3-6-1-4-1-99999-42"},
					Metrics: []profiles.Metric{{Metric: "snmp.system.name", OID: "1.3.6.1.2.1.1.5.0", Type: "get"}},
				},
			},
		}},
	}
	genProfilePath := filepath.Join(profilesDir, "generated-sysobj-1-3-6-1-4-1-99999-42.yml")
	writeGeneratedProfileFixture(t, genProfilePath, "generated-sysobj-1-3-6-1-4-1-99999-42", "discovered", "sysobj-1-3-6-1-4-1-99999-42")
	res, err := service.RunOnce(context.Background(), filepath.Join(tmp, "agent.yml"), loaded)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if res.Promoted != 1 || res.ProfileMatched != 1 || !res.Changed {
		t.Fatalf("unexpected result: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(tmp, "devices.d", "discovered-mystery-a.yml")); err != nil {
		t.Fatalf("expected promoted file: %v", err)
	}
}

func writeProfile(t *testing.T, path, name, vendor, model string) {
	t.Helper()
	content := "name: " + name + "\nmatch:\n  vendor: \"" + vendor + "\"\n  model: \"" + model + "\"\nmetrics:\n  - metric: snmp.system.name\n    oid: 1.3.6.1.2.1.1.5.0\n    type: get\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeGeneratedProfileFixture(t *testing.T, path, name, vendor, model string) {
	t.Helper()
	content := "name: " + name + "\nmatch:\n  vendor: \"" + vendor + "\"\n  model: \"" + model + "\"\nmetrics:\n  - metric: snmp.system.name\n    oid: 1.3.6.1.2.1.1.5.0\n    type: get\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
