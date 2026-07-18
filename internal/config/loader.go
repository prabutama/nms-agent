package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// LoadFromFile loads the root agent config from path and then loads referenced configs
// (devices directory, thresholds, adapters). It stays platform-agnostic and only
// handles file IO and YAML decoding.
func LoadFromFile(path string) (Loaded, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return Loaded{}, err
	}

	root, err := loadRoot(path)
	if err != nil {
		return Loaded{}, err
	}

	baseDir := filepath.Dir(path)
	resolve := func(p string) string {
		p = expandEnv(p)
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		return filepath.Clean(filepath.Join(baseDir, p))
	}

	devicesDir := resolve(root.Paths.DevicesDir)
	thresholdsFile := resolve(root.Paths.ThresholdsFile)
	adaptersFile := resolve(root.Paths.AdaptersFile)
	profilesDir := root.Paths.ProfilesDir
	if profilesDir == "" {
		profilesDir = filepath.Join(baseDir, "profiles")
	} else {
		profilesDir = resolve(profilesDir)
	}

	devices, err := loadDevicesDir(devicesDir)
	if err != nil {
		return Loaded{}, err
	}

	thresholds, err := loadYAMLFile[ThresholdsConfig](thresholdsFile)
	if err != nil {
		return Loaded{}, err
	}

	adapters, err := loadYAMLFile[AdaptersConfig](adaptersFile)
	if err != nil {
		return Loaded{}, err
	}
	if adapters.Adapters.Configs != nil {
		adapters.Adapters.Configs = expandEnvMap(adapters.Adapters.Configs)
	}

	return Loaded{
		Root:        root,
		Devices:     devices,
		Thresholds:  thresholds,
		Adapters:    adapters,
		ProfilesDir: profilesDir,
	}, nil
}

func loadRoot(path string) (Root, error) {
	root, err := loadYAMLFile[Root](path)
	if err != nil {
		return Root{}, err
	}
	root.Paths.DevicesDir = strings.TrimSpace(root.Paths.DevicesDir)
	root.Paths.ThresholdsFile = strings.TrimSpace(root.Paths.ThresholdsFile)
	root.Paths.AdaptersFile = strings.TrimSpace(root.Paths.AdaptersFile)
	root.Paths.NMSAgentDB = strings.TrimSpace(root.Paths.NMSAgentDB)
	root.Paths.ProfilesDir = strings.TrimSpace(root.Paths.ProfilesDir)
	root.Paths.ViewSocket = strings.TrimSpace(root.Paths.ViewSocket)
	root.Discovery.Interface = strings.TrimSpace(root.Discovery.Interface)
	root.Discovery.Subnet = strings.TrimSpace(root.Discovery.Subnet)
	root.Discovery.Provider = strings.TrimSpace(root.Discovery.Provider)
	root.Discovery.SNMP.Version = strings.TrimSpace(root.Discovery.SNMP.Version)
	root.Discovery.SNMP.Community = strings.TrimSpace(root.Discovery.SNMP.Community)
	root.Discovery.AutoPromote.DeviceIDTemplate = strings.TrimSpace(root.Discovery.AutoPromote.DeviceIDTemplate)
	root.Discovery.AutoPromote.WriteTo = strings.TrimSpace(root.Discovery.AutoPromote.WriteTo)
	root.Discovery.Exploration.RunWhen = strings.TrimSpace(root.Discovery.Exploration.RunWhen)
	root.Discovery.Exploration.OutputDir = strings.TrimSpace(root.Discovery.Exploration.OutputDir)
	return root, nil
}

func loadDevicesDir(dir string) ([]Device, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []Device
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		dev, err := loadYAMLFile[Device](p)
		if err != nil {
			return nil, fmt.Errorf("load device %s: %w", p, err)
		}
		out = append(out, dev)
	}
	return out, nil
}

func loadYAMLFile[T any](path string) (T, error) {
	var zero T

	f, err := os.Open(path)
	if err != nil {
		return zero, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return zero, err
	}

	var v T
	if err := yaml.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// expandEnv performs minimal environment expansion for config paths.
// It does not load .env files yet; it only expands using the current process env.
func expandEnv(s string) string {
	return os.ExpandEnv(strings.TrimSpace(s))
}

func expandEnvMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = expandEnvAny(v)
	}
	return out
}

func expandEnvAny(v any) any {
	switch x := v.(type) {
	case string:
		return expandEnv(x)
	case map[string]any:
		return expandEnvMap(x)
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, expandEnvAny(item))
		}
		return out
	default:
		return v
	}
}
