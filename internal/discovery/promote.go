package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nms-agent/internal/config"

	yaml "gopkg.in/yaml.v3"
)

var reUnsafeDeviceID = regexp.MustCompile(`[^a-z0-9._-]+`)

func RenderDeviceID(tmpl string, fp Fingerprint) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = "{{vendor}}-{{sys_name}}"
	}
	name := strings.TrimSpace(fp.SysName)
	if name == "" {
		name = strings.ReplaceAll(fp.Address, ".", "-")
	}
	out := strings.ReplaceAll(tmpl, "{{vendor}}", strings.TrimSpace(fp.Vendor))
	out = strings.ReplaceAll(out, "{{sys_name}}", name)
	out = strings.ToLower(strings.TrimSpace(out))
	out = reUnsafeDeviceID.ReplaceAllString(out, "-")
	out = strings.Trim(out, "-._")
	out = strings.ReplaceAll(out, "--", "-")
	if out == "" {
		out = "device-" + strings.ReplaceAll(fp.Address, ".", "-")
	}
	return out
}

func uniqueDeviceID(base string, used map[string]struct{}) string {
	if _, ok := used[base]; !ok {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func writePromotedDevice(baseDir string, loaded config.Loaded, fp Fingerprint, used map[string]struct{}) (string, error) {
	writeDir := config.ResolvePath(baseDir, loaded.Root.Discovery.AutoPromote.WriteTo)
	if err := os.MkdirAll(writeDir, 0o755); err != nil {
		return "", err
	}
	baseID := RenderDeviceID(loaded.Root.Discovery.AutoPromote.DeviceIDTemplate, fp)
	deviceID := uniqueDeviceID(baseID, used)
	path := filepath.Join(writeDir, deviceID+".yml")
	dev := config.Device{
		ID:      deviceID,
		Address: fp.Address,
		Vendor:  fp.Vendor,
		Model:   fp.Model,
		SNMP:    config.DeviceSNMP{Enabled: true},
		ICMP:    config.DeviceICMP{Enabled: true},
	}
	b, err := yaml.Marshal(dev)
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(writeDir, fmt.Sprintf(".%s.%d.tmp", deviceID, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return path, nil
}
