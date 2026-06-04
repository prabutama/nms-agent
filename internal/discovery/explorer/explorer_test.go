package explorer

import (
	"os"
	"path/filepath"
	"testing"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	"nms-agent/internal/profiles"
)

func TestGeneratedMatch_UsesSysObjectIDFallback(t *testing.T) {
	vendor, model := generatedMatch(discovery.Fingerprint{Candidate: discovery.Candidate{Address: "192.168.10.20"}, SysObjectID: "1.3.6.1.4.1.99999.42"})
	if vendor != "discovered" {
		t.Fatalf("unexpected vendor: %s", vendor)
	}
	if model != "sysobj-1-3-6-1-4-1-99999-42" {
		t.Fatalf("unexpected model: %s", model)
	}
}

func TestWriteGeneratedProfile_WritesFile(t *testing.T) {
	tmp := t.TempDir()
	loaded := config.Loaded{Root: config.Root{Discovery: config.Discovery{Exploration: config.DiscoveryExploration{OutputDir: "profiles"}}}}
	prof := profiles.Profile{
		Name:    "generated-x",
		Match:   profiles.Match{Vendor: "discovered", Model: "sysobj-x"},
		Metrics: []profiles.Metric{{Metric: "snmp.system.name", OID: "1.3.6.1.2.1.1.5.0", Type: "get"}},
	}
	path, err := writeGeneratedProfile(filepath.Join(tmp, "agent.yml"), loaded, prof)
	if err != nil {
		t.Fatalf("writeGeneratedProfile error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected generated profile file: %v", err)
	}
}
