package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// LoadDir loads all profile YAML files from dir.
// Each file is expected to contain a single Profile object.
func LoadDir(dir string) ([]Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []Profile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		prof, err := LoadFile(p)
		if err != nil {
			return nil, fmt.Errorf("load profile %s: %w", p, err)
		}
		out = append(out, prof)
	}
	return out, nil
}

// LoadFile loads a single profile from path.
func LoadFile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	return p, nil
}
