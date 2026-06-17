package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nms-agent/internal/models"

	yaml "gopkg.in/yaml.v3"
)

// Validate checks the loaded config for basic correctness.
// Keep this strict for required fields and conservative for future fields.
func Validate(cfg Loaded) error {
	var errs []string

	if cfg.Root.Agent.PollInterval <= 0 {
		errs = append(errs, "agent.poll_interval must be > 0")
	}
	// Logging config is optional; validate if provided.
	if lvl := strings.TrimSpace(cfg.Root.Agent.Logging.Level); lvl != "" {
		switch lvl {
		case "debug", "info", "warn", "warning", "error":
		default:
			errs = append(errs, "agent.logging.level must be one of: debug, info, warn, error")
		}
	}
	if fmt := strings.TrimSpace(cfg.Root.Agent.Logging.Format); fmt != "" {
		switch fmt {
		case "text", "json":
		default:
			errs = append(errs, "agent.logging.format must be one of: text, json")
		}
	}

	// Output timezone is presentation-only; validate if provided.
	if tz := strings.TrimSpace(cfg.Root.Agent.Output.Timezone); tz != "" {
		if _, err := LoadLocation(tz); err != nil {
			errs = append(errs, "agent.output.timezone: "+err.Error())
		}
	}
	// Paths fields should be present in agent.yml.
	if strings.TrimSpace(cfg.Root.Paths.DevicesDir) == "" {
		errs = append(errs, "paths.devices_dir is required")
	}
	if strings.TrimSpace(cfg.Root.Paths.ThresholdsFile) == "" {
		errs = append(errs, "paths.thresholds_file is required")
	}
	if strings.TrimSpace(cfg.Root.Paths.AdaptersFile) == "" {
		errs = append(errs, "paths.adapters_file is required")
	}
	if strings.TrimSpace(cfg.Root.Paths.QueueDB) == "" {
		errs = append(errs, "paths.queue_db is required")
	}

	// Devices.
	seenIDs := map[string]struct{}{}
	for i, d := range cfg.Devices {
		prefix := fmt.Sprintf("devices[%d]", i)
		if strings.TrimSpace(d.ID) == "" {
			errs = append(errs, prefix+".id is required")
		} else {
			if _, ok := seenIDs[d.ID]; ok {
				errs = append(errs, "duplicate device id: "+d.ID)
			} else {
				seenIDs[d.ID] = struct{}{}
			}
			// Reject device ids with hidden/control characters.
			for _, r := range d.ID {
				if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
					errs = append(errs, prefix+".id contains invalid character: '"+string(r)+"'")
					break
				}
			}
		}
		if strings.TrimSpace(d.Address) == "" {
			errs = append(errs, prefix+".address is required")
		} else {
			// Reject addresses with hidden/control characters.
			for _, r := range d.Address {
				if r < 32 && r != '\t' || (r > 126 && r != '\t') {
					errs = append(errs, prefix+".address contains invalid character: '"+string(r)+"'")
					break
				}
			}
		}
		// Collector enable flags are optional; no strict validation yet.
	}

	// Adapters config: keep minimal but ensure it parses and has an active name.
	if strings.TrimSpace(cfg.Adapters.Adapters.Active) == "" {
		errs = append(errs, "adapters.active is required")
	}
	if cfg.Adapters.Adapters.Configs == nil {
		// If YAML omitted it, treat as empty.
		cfg.Adapters.Adapters.Configs = map[string]any{}
	}

	// Threshold rules validation (Phase 7 MVP).
	for i, r := range cfg.Thresholds.Thresholds {
		prefix := fmt.Sprintf("thresholds[%d]", i)
		if strings.TrimSpace(r.Metric) == "" {
			errs = append(errs, prefix+".metric is required")
		}
		op := strings.TrimSpace(r.Operator)
		switch op {
		case ">", ">=", "<", "<=", "==", "!=":
			// ok
		default:
			errs = append(errs, prefix+".operator must be one of >, >=, <, <=, ==, !=")
		}
		if r.Warning == nil && r.Critical == nil {
			errs = append(errs, prefix+".warning or .critical is required")
		}
		for k, v := range r.Tags {
			if strings.TrimSpace(k) == "" {
				errs = append(errs, prefix+".tags has empty key")
				break
			}
			if strings.TrimSpace(v) == "" {
				errs = append(errs, prefix+".tags["+k+"] is empty")
				break
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid config:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

// ValidateThresholdRules validates an isolated set of threshold rules (no full config required).
func ValidateThresholdRules(rules []models.ThresholdRule) error {
	var errs []string
	for i, r := range rules {
		prefix := fmt.Sprintf("thresholds[%d]", i)
		if strings.TrimSpace(r.Metric) == "" {
			errs = append(errs, prefix+".metric is required")
		}
		op := strings.TrimSpace(r.Operator)
		switch op {
		case ">", ">=", "<", "<=", "==", "!=":
			// ok
		default:
			errs = append(errs, prefix+".operator must be one of >, >=, <, <=, ==, !=")
		}
		if r.Warning == nil && r.Critical == nil {
			errs = append(errs, prefix+".warning or .critical is required")
		}
		for k, v := range r.Tags {
			if strings.TrimSpace(k) == "" {
				errs = append(errs, prefix+".tags has empty key")
				break
			}
			if strings.TrimSpace(v) == "" {
				errs = append(errs, prefix+".tags["+k+"] is empty")
				break
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid threshold rules:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

// ValidateFiles performs validation that requires reading files/dirs from disk.
// This is used by the CLI validate command.
func ValidateFiles(agentConfigPath string) error {
	abs, err := filepath.Abs(agentConfigPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("config file not found: %s: %w", abs, err)
	}

	cfg, err := LoadFromFile(abs)
	if err != nil {
		return err
	}
	if err := Validate(cfg); err != nil {
		return err
	}

	// Additional YAML structure sanity for placeholder files.
	if err := ensureYAMLKeyExists(abs, cfg.Root.Paths.ThresholdsFile, "thresholds"); err != nil {
		return err
	}
	if err := ensureYAMLKeyExists(abs, cfg.Root.Paths.AdaptersFile, "adapters"); err != nil {
		return err
	}

	// Security warnings.
	w := warnings(cfg)
	for _, msg := range w {
		fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
	}

	return nil
}

func warnings(cfg Loaded) []string {
	var out []string

	// Check adapter configs for known insecure patterns.
	adapterName := cfg.Adapters.Adapters.Active
	adapterCfg := cfg.Adapters.Adapters.Configs

	// MQTT over plain TCP without TLS.
	if adapterName == "generic_mqtt" || adapterName == "thingsboard_mqtt" {
		if broker, ok := adapterCfg["broker"].(string); ok {
			if strings.HasPrefix(broker, "tcp://") {
				out = append(out, fmt.Sprintf("adapter %q uses plain TCP broker %q (consider mqtts:// or wss:// in production)", adapterName, broker))
			}
		}
	}

	return out
}

func ensureYAMLKeyExists(agentConfigAbsPath, relativeOrAbsPath, topKey string) error {
	baseDir := filepath.Dir(agentConfigAbsPath)
	path := expandEnv(relativeOrAbsPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	path = filepath.Clean(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(data, &n); err != nil {
		return err
	}
	if !nodeHasTopKey(&n, topKey) {
		return fmt.Errorf("%s must contain top-level key '%s'", path, topKey)
	}
	return nil
}

func nodeHasTopKey(n *yaml.Node, key string) bool {
	// Document -> Mapping
	if n == nil {
		return false
	}
	cur := n
	if cur.Kind == yaml.DocumentNode && len(cur.Content) > 0 {
		cur = cur.Content[0]
	}
	if cur.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(cur.Content); i += 2 {
		k := cur.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return true
		}
	}
	return false
}
