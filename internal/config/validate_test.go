package config

import (
	"testing"
	"time"
)

func TestValidate_BasicRequiredFields(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z"},
		},
		Devices:  []Device{{ID: "d1", Address: "127.0.0.1"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "terminal", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidate_DuplicateDeviceID(t *testing.T) {
	cfg := Loaded{
		Root: Root{
			Agent: Agent{PollInterval: time.Second},
			Paths: Paths{DevicesDir: "x", ThresholdsFile: "y", AdaptersFile: "z"},
		},
		Devices:  []Device{{ID: "d1", Address: "1"}, {ID: "d1", Address: "2"}},
		Adapters: AdaptersConfig{Adapters: AdaptersSection{Active: "terminal", Configs: map[string]any{}}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected error")
	}
}
