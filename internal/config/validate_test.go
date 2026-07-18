package config

import (
	"testing"
	"time"
)

func TestValidate_BasicRequiredFields(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
		},
		Devices:  []Device{{ID: "d1", Address: "127.0.0.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidate_DuplicateDeviceID(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
		},
		Devices:  []Device{{ID: "d1", Address: "1"}, {ID: "d1", Address: "2"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_DeviceIDWithHiddenChars(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
		},
		Devices:  []Device{{ID: "bad\x01id", Address: "1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error for device id with hidden char")
	}
}

func TestValidate_DeviceAddressWithHiddenChars(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
		},
		Devices:  []Device{{ID: "d1", Address: "172.16\x01.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error for address with hidden char")
	}
}

func TestValidate_IgnoresDiscoveryBlockForDaemonConfig(t *testing.T) {
	community := "public"
	t.Setenv("SNMP_COMMUNITY", community)
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
			Discovery: Discovery{
				Enabled:   true,
				Interface: "eth0",
				Subnet:    "192.168.10.0/24",
				Provider:  "netlink",
				SNMP: DiscoverySNMP{
					Version:     "v2c",
					Community:   "${SNMP_COMMUNITY}",
					Timeout:     2 * time.Second,
					Retries:     1,
					Concurrency: 32,
				},
				AutoPromote: DiscoveryAutoPromote{
					Enabled:               true,
					MaxNewDevicesPerCycle: 10,
					DeviceIDTemplate:      "{{vendor}}-{{sys_name}}",
					WriteTo:               "devices.d",
				},
				Exploration: DiscoveryExploration{
					Enabled:   true,
					RunWhen:   "no_profile_match",
					Timeout:   3 * time.Second,
					OutputDir: "profiles",
				},
			},
		},
		Devices:  []Device{{ID: "d1", Address: "127.0.0.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidate_DiscoveryBlockIsOptionalAndNotRuntimeValidated(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", NMSAgentDB: "nms-agent.db"},
			Discovery: Discovery{
				Enabled:   true,
				Interface: "",
				Subnet:    "bad-subnet",
				Provider:  "other",
				SNMP: DiscoverySNMP{
					Version:     "v3",
					Community:   "",
					Timeout:     0,
					Concurrency: 0,
				},
				AutoPromote: DiscoveryAutoPromote{
					MaxNewDevicesPerCycle: -1,
					WriteTo:               "",
				},
				Exploration: DiscoveryExploration{
					Enabled: true,
					RunWhen: "always",
				},
			},
		},
		Devices:  []Device{{ID: "d1", Address: "127.0.0.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
