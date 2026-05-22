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

	return Loaded{
		Root:       root,
		Devices:    devices,
		Thresholds: thresholds,
		Adapters:   adapters,
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
	root.Paths.QueueDB = strings.TrimSpace(root.Paths.QueueDB)
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
