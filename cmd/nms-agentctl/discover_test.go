package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

type fakeConn struct{ local net.Addr }

func (c fakeConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (c fakeConn) Write(b []byte) (int, error)        { return len(b), nil }
func (c fakeConn) Close() error                       { return nil }
func (c fakeConn) LocalAddr() net.Addr                { return c.local }
func (c fakeConn) RemoteAddr() net.Addr               { return &net.UDPAddr{} }
func (c fakeConn) SetDeadline(t time.Time) error      { return nil }
func (c fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c fakeConn) SetWriteDeadline(t time.Time) error { return nil }

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
	newDiscoveryService = func(_ config.Loaded) discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			}},
		}
	}
	code := runDiscoveryPreview([]string{"--config", agentYml, "--subnet", "192.168.10.0/24", "--interface", "eth0"})
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
	newDiscoveryService = func(_ config.Loaded) discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			}},
		}
	}
	code := runDiscoveryRun([]string{"--config", agentYml, "--subnet", "192.168.10.0/24", "--interface", "eth0"})
	if code != 0 {
		t.Fatalf("runDiscoveryRun exit code=%d", code)
	}
	info, err := os.Stat(filepath.Join(tmp, "devices.d", "linux-server-a.yml"))
	if err != nil {
		t.Fatalf("expected promoted file: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		got := info.Mode().Perm()
		t.Fatalf("promoted file mode=%o, want 600", got)
	}
}

func TestDiscoveryRun_UnknownProfileDoesNotWriteDevice(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	oldFactory := newDiscoveryService
	defer func() { newDiscoveryService = oldFactory }()
	newDiscoveryService = func(_ config.Loaded) discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.20", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.20": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.99999.42", SysName: "unknown-box", SysDescr: "Unknown appliance"},
			}},
		}
	}
	code := runDiscoveryRun([]string{"--config", agentYml, "--subnet", "192.168.10.0/24", "--interface", "eth0"})
	if code != 0 {
		t.Fatalf("runDiscoveryRun exit code=%d", code)
	}
	entries, err := os.ReadDir(filepath.Join(tmp, "devices.d"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no promoted devices, got %d", len(entries))
	}
}

func TestDiscoveryRun_CommunityAliasFallback(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	oldFactory := newDiscoveryService
	defer func() { newDiscoveryService = oldFactory }()
	newDiscoveryService = func(_ config.Loaded) discovery.Service {
		return discovery.Service{
			Provider: cliFakeProvider{candidates: []discovery.Candidate{{Address: "192.168.10.10", Interface: "eth0", Source: "test"}}},
			Prober: cliFakeProber{byAddress: map[string]discovery.Fingerprint{
				"192.168.10.10": {SNMPOK: true, SysObjectID: "1.3.6.1.4.1.8072.3.2.10", SysName: "server-a", SysDescr: "Linux ubuntu"},
			}},
		}
	}
	code := runDiscoveryRun([]string{"--config", agentYml, "--subnet", "192.168.10.0/24", "--interface", "eth0", "--community", "private"})
	if code != 0 {
		t.Fatalf("runDiscoveryRun with --community alias exit code=%d, want 0", code)
	}
}

func TestDiscoveryRun_RequiresSubnet(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	code := runDiscoveryRun([]string{"--config", agentYml})
	if code != 2 {
		t.Fatalf("runDiscoveryRun exit code=%d, want 2", code)
	}
}

func TestDiscoveryStatus_PrintsConfig(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeDiscoveryAgentFiles(t, tmp)
	loaded, err := config.LoadFromFile(agentYml)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDiscoveryCLIOverrides(&loaded, "192.168.10.0/24", "eth0", "active", "public", 0, -1, 0, 10, "", false, true); err != nil {
		t.Fatal(err)
	}
	configContent := "agent:\n  poll_interval: 1s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n  profiles_dir: profiles\n  nms_agent_db: nms-agent.db\ndiscovery:\n  enabled: true\n  interface: eth0\n  subnet: 192.168.10.0/24\n  provider: active\n  snmp:\n    version: v2c\n    community: public\n    timeout: 2s\n    retries: 1\n    concurrency: 4\n  auto_promote:\n    enabled: true\n    require_profile_match: true\n    max_new_devices_per_cycle: 10\n    device_id_template: '{{vendor}}-{{sys_name}}'\n  exploration:\n    enabled: false\n"
	if err := os.WriteFile(agentYml, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := captureStdout(t, func() {
		code := runDiscoveryStatus([]string{"--config", agentYml})
		if code != 0 {
			t.Fatalf("runDiscoveryStatus exit code=%d", code)
		}
	})
	if !strings.Contains(stdout, "configured=true") || !strings.Contains(stdout, "subnet=192.168.10.0/24") {
		t.Fatalf("unexpected status output: %s", stdout)
	}
}

func TestApplyDiscoveryCLIOverrides_MinimalSubnetAppliesSafeDefaults(t *testing.T) {
	tmp := t.TempDir()
	loaded, err := config.LoadFromFile(writeDiscoveryAgentFiles(t, tmp))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DISCOVERY_COMMUNITY", "private")
	if err := applyDiscoveryCLIOverrides(&loaded, "192.168.10.0/24", "eth0", "", "${DISCOVERY_COMMUNITY}", 0, -1, 0, 0, "", false, true); err != nil {
		t.Fatal(err)
	}
	d := loaded.Root.Discovery
	if !d.Enabled || d.Provider != "active" || d.SNMP.Community != "private" {
		t.Fatalf("unexpected discovery defaults: %+v", d)
	}
	if !d.AutoPromote.Enabled || !d.AutoPromote.RequireSNMPOK || !d.AutoPromote.RequireSysObjectID || !d.AutoPromote.RequireProfileMatch {
		t.Fatalf("unsafe promote defaults: %+v", d.AutoPromote)
	}
	if d.AutoPromote.MaxNewDevicesPerCycle != discovery.DefaultMaxNewDevices {
		t.Fatalf("max new devices=%d, want %d", d.AutoPromote.MaxNewDevicesPerCycle, discovery.DefaultMaxNewDevices)
	}
}

func TestApplyDiscoveryCLIOverrides_InvalidProviderFails(t *testing.T) {
	tmp := t.TempDir()
	loaded, err := config.LoadFromFile(writeDiscoveryAgentFiles(t, tmp))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDiscoveryCLIOverrides(&loaded, "192.168.10.0/24", "eth0", "bad", "", 0, -1, 0, 0, "", false, true); err == nil {
		t.Fatalf("expected invalid provider error")
	}
}

func TestApplyDiscoveryCLIOverrides_MaxNewDevicesSemantics(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"default", 0, discovery.DefaultMaxNewDevices},
		{"explicit", 3, 3},
		{"unlimited", -1, discovery.UnlimitedMaxNewDevices},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			loaded, err := config.LoadFromFile(writeDiscoveryAgentFiles(t, tmp))
			if err != nil {
				t.Fatal(err)
			}
			if err := applyDiscoveryCLIOverrides(&loaded, "192.168.10.0/24", "eth0", "active", "", 0, -1, 0, tt.in, "", false, true); err != nil {
				t.Fatal(err)
			}
			if got := loaded.Root.Discovery.AutoPromote.MaxNewDevicesPerCycle; got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAutoDetectInterfaceResolverErrors(t *testing.T) {
	dialErr := interfaceResolver{dial: func(network, address string) (net.Conn, error) {
		return nil, errors.New("dial failed")
	}}
	if _, err := dialErr.autoDetectInterface("192.168.10.0/24"); err == nil || !strings.Contains(err.Error(), "pass --interface") {
		t.Fatalf("expected dial error, got %v", err)
	}
	missingIface := interfaceResolver{
		dial: func(network, address string) (net.Conn, error) {
			return fakeConn{local: &net.UDPAddr{IP: net.ParseIP("192.168.10.2")}}, nil
		},
		interfaces: func() ([]net.Interface, error) { return nil, nil },
	}
	if _, err := missingIface.autoDetectInterface("192.168.10.0/24"); err == nil || !strings.Contains(err.Error(), "pass --interface") {
		t.Fatalf("expected interface error, got %v", err)
	}
}

func TestFirstUsableHost(t *testing.T) {
	host, err := firstUsableHost("192.168.10.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if host != "192.168.10.1" {
		t.Fatalf("host=%s, want 192.168.10.1", host)
	}
	_, err = firstUsableHost("bad")
	if err == nil || !strings.Contains(err.Error(), "valid CIDR") {
		t.Fatalf("expected CIDR error, got %v", err)
	}
	for _, cidr := range []string{"192.168.10.0/31", "192.168.10.1/32"} {
		if _, err := firstUsableHost(cidr); err == nil || !strings.Contains(err.Error(), "no usable host") {
			t.Fatalf("expected no usable host error for %s, got %v", cidr, err)
		}
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
	agentContent := "agent:\n  poll_interval: 1s\npaths:\n  devices_dir: devices.d\n  thresholds_file: thresholds.yml\n  adapters_file: adapters.yml\n  profiles_dir: profiles\n  nms_agent_db: nms-agent.db\n"
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()
	fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
