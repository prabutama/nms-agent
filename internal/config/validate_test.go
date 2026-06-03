package config

import (
	"testing"
	"time"
)

func TestValidate_BasicRequiredFields(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", QueueDB: "q.db"},
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
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", QueueDB: "q.db"},
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
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", QueueDB: "q.db"},
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
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z", QueueDB: "q.db"},
		},
		Devices:  []Device{{ID: "d1", Address: "172.16\x01.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "tui", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error for address with hidden char")
	}
}
