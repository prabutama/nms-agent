package config

import "time"

// Root is loaded from `agent.yml` and references other config files.
type Root struct {
	Agent Agent `yaml:"agent"`
	Paths Paths `yaml:"paths"`
}

type Agent struct {
	PollInterval time.Duration `yaml:"poll_interval"`
}

type Paths struct {
	DevicesDir     string `yaml:"devices_dir"`
	ThresholdsFile string `yaml:"thresholds_file"`
	AdaptersFile   string `yaml:"adapters_file"`
	QueueDB        string `yaml:"queue_db"`
}

// Device is loaded from `devices.d/*.yml`.
type Device struct {
	ID      string `yaml:"id"`
	Address string `yaml:"address"`
	Vendor  string `yaml:"vendor"`
	Model   string `yaml:"model"`

	SNMP DeviceSNMP `yaml:"snmp"`
	ICMP DeviceICMP `yaml:"icmp"`
}

type DeviceSNMP struct {
	Enabled bool `yaml:"enabled"`
}

type DeviceICMP struct {
	Enabled bool `yaml:"enabled"`
}

// ThresholdsConfig is a placeholder shape for Phase 2 (no behavior yet).
type ThresholdsConfig struct {
	Thresholds []any `yaml:"thresholds"`
}

// AdaptersConfig is a placeholder shape for Phase 2 (no adapter runtime logic yet).
type AdaptersConfig struct {
	Adapters AdaptersSection `yaml:"adapters"`
}

type AdaptersSection struct {
	Active  string         `yaml:"active"`
	Configs map[string]any `yaml:"configs"`
}

// Loaded is the fully materialized configuration used by the agent and CLI.
type Loaded struct {
	Root       Root
	Devices    []Device
	Thresholds ThresholdsConfig
	Adapters   AdaptersConfig
}
