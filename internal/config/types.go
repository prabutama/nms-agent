package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"nms-agent/internal/models"
)

// Root is loaded from `agent.yml` and references other config files.
type Root struct {
	Agent     Agent     `yaml:"agent"`
	Paths     Paths     `yaml:"paths"`
	Discovery Discovery `yaml:"discovery"`
}

type Agent struct {
	PollInterval time.Duration `yaml:"poll_interval"`
	Delivery     Delivery      `yaml:"delivery"`
	Output       Output        `yaml:"output"`
	Logging      Logging       `yaml:"logging"`
}

// Output configures presentation-only output settings.
// It must not affect core persistence semantics (queue stores canonical telemetry).
type Output struct {
	Timezone string `yaml:"timezone"`
}

// Logging configures structured logging for the agent.
type Logging struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func (l Logging) WithDefaults() Logging {
	out := l
	if out.Level == "" {
		out.Level = "info"
	}
	if out.Format == "" {
		out.Format = "text"
	}
	return out
}

// Delivery configures the queue delivery drain loop (Phase 8).
type Delivery struct {
	MaxBatch           int           `yaml:"max_batch"`
	DrainEnabled       bool          `yaml:"drain_enabled"`
	MaxBatchesPerCycle int           `yaml:"max_batches_per_cycle"`
	StopOnError        bool          `yaml:"stop_on_error"`
	Retry              DeliveryRetry `yaml:"retry"`
}

type DeliveryRetry struct {
	Enabled       bool          `yaml:"enabled"`
	BaseBackoff   time.Duration `yaml:"base_backoff"`
	MaxBackoff    time.Duration `yaml:"max_backoff"`
	MaxRetries    int           `yaml:"max_retries"`
	RetentionDays int           `yaml:"retention_days"`
}

func (r DeliveryRetry) WithDefaults() DeliveryRetry {
	out := r
	if out.BaseBackoff <= 0 {
		out.BaseBackoff = 10 * time.Second
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = 300 * time.Second
	}
	if out.MaxRetries <= 0 {
		out.MaxRetries = 10
	}
	if out.RetentionDays <= 0 {
		out.RetentionDays = 30
	}
	return out
}

// ResolvePath resolves a config path relative to baseDir and expands env vars.
func ResolvePath(baseDir, path string) string {
	p := os.ExpandEnv(strings.TrimSpace(path))
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(baseDir, p))
}

func (d Delivery) WithDefaults() Delivery {
	if d.MaxBatch <= 0 {
		d.MaxBatch = 100
	}
	if d.MaxBatchesPerCycle <= 0 {
		d.MaxBatchesPerCycle = 10
	}
	return d
}

type Paths struct {
	DevicesDir     string `yaml:"devices_dir"`
	ThresholdsFile string `yaml:"thresholds_file"`
	AdaptersFile   string `yaml:"adapters_file"`
	QueueDB        string `yaml:"queue_db"`
	ProfilesDir    string `yaml:"profiles_dir"`
}

type Discovery struct {
	Enabled     bool                 `yaml:"enabled"`
	Interface   string               `yaml:"interface"`
	Subnet      string               `yaml:"subnet"`
	Provider    string               `yaml:"provider"`
	ActiveProbe DiscoveryActiveProbe `yaml:"active_probe"`
	SNMP        DiscoverySNMP        `yaml:"snmp"`
	AutoPromote DiscoveryAutoPromote `yaml:"auto_promote"`
	Exploration DiscoveryExploration `yaml:"exploration"`
}

type DiscoveryActiveProbe struct {
	Timeout     time.Duration `yaml:"timeout"`
	Concurrency int           `yaml:"concurrency"`
}

type DiscoverySNMP struct {
	Version     string        `yaml:"version"`
	Community   string        `yaml:"community"`
	Timeout     time.Duration `yaml:"timeout"`
	Retries     int           `yaml:"retries"`
	Concurrency int           `yaml:"concurrency"`
}

type DiscoveryAutoPromote struct {
	Enabled               bool   `yaml:"enabled"`
	RequireSNMPOK         bool   `yaml:"require_snmp_ok"`
	RequireSysObjectID    bool   `yaml:"require_sys_object_id"`
	RequireProfileMatch   bool   `yaml:"require_profile_match"`
	MaxNewDevicesPerCycle int    `yaml:"max_new_devices_per_cycle"`
	DeviceIDTemplate      string `yaml:"device_id_template"`
	WriteTo               string `yaml:"write_to"`
}

type DiscoveryExploration struct {
	Enabled                     bool          `yaml:"enabled"`
	RunWhen                     string        `yaml:"run_when"`
	SafeOnly                    bool          `yaml:"safe_only"`
	AutoApproveGeneratedProfile bool          `yaml:"auto_approve_generated_profile"`
	AutoPromoteAfterGenerate    bool          `yaml:"auto_promote_after_generate"`
	MaxOIDsPerDevice            int           `yaml:"max_oids_per_device"`
	Timeout                     time.Duration `yaml:"timeout"`
	OutputDir                   string        `yaml:"output_dir"`
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

// ThresholdsConfig contains threshold rules (Phase 7).
type ThresholdsConfig struct {
	Thresholds []models.ThresholdRule `yaml:"thresholds"`
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
	Root        Root
	Devices     []Device
	Thresholds  ThresholdsConfig
	Adapters    AdaptersConfig
	ProfilesDir string
}
