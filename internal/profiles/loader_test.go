package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir_LoadsProfiles(t *testing.T) {
	tmp := t.TempDir()

	p1 := "name: standard\nmatch:\n  vendor: \"\"\n  model: \"\"\nmetrics:\n  - metric: snmp.uptime_seconds\n    oid: 1.3.6.1.2.1.1.3.0\n    type: get\n    unit: s\n"
	if err := os.WriteFile(filepath.Join(tmp, "standard.yml"), []byte(p1), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	p2 := "name: vendor-default\nmatch:\n  vendor: example\n  model: \"\"\nmetrics:\n  - metric: snmp.if.oper_status\n    oid: 1.3.6.1.2.1.2.2.1.8\n    type: walk\n    index: true\n"
	if err := os.WriteFile(filepath.Join(tmp, "vendor.yml"), []byte(p2), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	profiles, err := LoadDir(tmp)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
}
