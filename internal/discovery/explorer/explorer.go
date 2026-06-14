package explorer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
	yaml "gopkg.in/yaml.v3"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	"nms-agent/internal/profiles"
)

type metricCandidate struct {
	Metric string
	OID    string
	Type   string
	Unit   string
	Index  bool
}

var safeCatalog = []metricCandidate{
	{Metric: "snmp.system.description", OID: "1.3.6.1.2.1.1.1.0", Type: "get"},
	{Metric: "snmp.system.name", OID: "1.3.6.1.2.1.1.5.0", Type: "get"},
	{Metric: "snmp.uptime_seconds", OID: "1.3.6.1.2.1.1.3.0", Type: "get", Unit: "s"},
	{Metric: "snmp.if.name", OID: "1.3.6.1.2.1.31.1.1.1.1", Type: "walk", Index: true},
	{Metric: "snmp.if.type", OID: "1.3.6.1.2.1.2.2.1.3", Type: "walk", Index: true},
	{Metric: "snmp.if.oper_status", OID: "1.3.6.1.2.1.2.2.1.8", Type: "walk", Index: true},
	{Metric: "snmp.if.hc_in_octets", OID: "1.3.6.1.2.1.31.1.1.1.6", Type: "walk", Unit: "octets", Index: true},
	{Metric: "snmp.if.hc_out_octets", OID: "1.3.6.1.2.1.31.1.1.1.10", Type: "walk", Unit: "octets", Index: true},
	{Metric: "snmp.if.speed_bps", OID: "1.3.6.1.2.1.2.2.1.5", Type: "walk", Unit: "bps", Index: true},
	{Metric: "snmp.if.high_speed_mbps", OID: "1.3.6.1.2.1.31.1.1.1.15", Type: "walk", Unit: "Mbps", Index: true},
	{Metric: "snmp.host.cpu.load_pct", OID: "1.3.6.1.2.1.25.3.3.1.2", Type: "walk", Unit: "pct", Index: true},
	{Metric: "snmp.host.memory.size_kb", OID: "1.3.6.1.2.1.25.2.2.0", Type: "get", Unit: "kB"},
	{Metric: "snmp.host.storage.type", OID: "1.3.6.1.2.1.25.2.3.1.2", Type: "walk", Index: true},
	{Metric: "snmp.host.storage.description", OID: "1.3.6.1.2.1.25.2.3.1.3", Type: "walk", Index: true},
	{Metric: "snmp.host.storage.allocation_units", OID: "1.3.6.1.2.1.25.2.3.1.4", Type: "walk", Unit: "bytes", Index: true},
	{Metric: "snmp.host.storage.size_units", OID: "1.3.6.1.2.1.25.2.3.1.5", Type: "walk", Index: true},
	{Metric: "snmp.host.storage.used_units", OID: "1.3.6.1.2.1.25.2.3.1.6", Type: "walk", Index: true},
}

var reUnsafe = regexp.MustCompile(`[^a-z0-9._-]+`)

type Explorer struct{}

func (e Explorer) Explore(ctx context.Context, configPath string, loaded config.Loaded, fp discovery.Fingerprint) (discovery.ExplorationResult, error) {
	var zero discovery.ExplorationResult
	if !loaded.Root.Discovery.Exploration.Enabled {
		return zero, nil
	}
	if strings.TrimSpace(loaded.Root.Discovery.Exploration.RunWhen) != "no_profile_match" {
		return zero, nil
	}
	prof, vendor, model, err := exploreProfile(ctx, loaded, fp)
	if err != nil {
		return zero, err
	}
	if err := profiles.ValidateProfile(prof); err != nil {
		return zero, err
	}
	if !loaded.Root.Discovery.Exploration.AutoApproveGeneratedProfile {
		return discovery.ExplorationResult{Profile: prof, Vendor: vendor, Model: model}, nil
	}
	path, err := writeGeneratedProfile(configPath, loaded, prof)
	if err != nil {
		return zero, err
	}
	return discovery.ExplorationResult{
		Generated:   true,
		ProfilePath: path,
		Profile:     prof,
		Vendor:      vendor,
		Model:       model,
	}, nil
}

func exploreProfile(ctx context.Context, loaded config.Loaded, fp discovery.Fingerprint) (profiles.Profile, string, string, error) {
	vendor, model := generatedMatch(fp)
	metrics, err := probeCatalog(ctx, loaded.Root.Discovery, fp.Address)
	if err != nil {
		return profiles.Profile{}, "", "", err
	}
	if len(metrics) == 0 {
		return profiles.Profile{}, "", "", fmt.Errorf("no safe metrics discovered")
	}
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].Metric < metrics[j].Metric })
	name := fmt.Sprintf("generated-%s", normalizeName(model))
	if name == "generated-" {
		name = "generated-profile"
	}
	return profiles.Profile{
		Name: name,
		Match: profiles.Match{
			Vendor: vendor,
			Model:  model,
		},
		Metrics: metrics,
	}, vendor, model, nil
}

func generatedMatch(fp discovery.Fingerprint) (string, string) {
	vendor := strings.TrimSpace(fp.Vendor)
	if vendor == "" {
		vendor = "discovered"
	}
	model := strings.TrimSpace(fp.Model)
	if model == "" {
		if strings.TrimSpace(fp.SysObjectID) != "" {
			model = "sysobj-" + strings.ReplaceAll(strings.TrimSpace(fp.SysObjectID), ".", "-")
		} else if strings.TrimSpace(fp.SysName) != "" {
			model = strings.TrimSpace(fp.SysName)
		} else {
			model = strings.ReplaceAll(fp.Address, ".", "-")
		}
	}
	return normalizeName(vendor), normalizeName(model)
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = reUnsafe.ReplaceAllString(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-._")
}

func probeCatalog(ctx context.Context, cfg config.Discovery, address string) ([]profiles.Metric, error) {
	timeout := cfg.Exploration.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 && d < timeout {
			timeout = d
		}
	}
	client := &g.GoSNMP{
		Target:    address,
		Port:      161,
		Community: expandCommunity(cfg.SNMP.Community),
		Version:   g.Version2c,
		Timeout:   timeout,
		Retries:   cfg.SNMP.Retries,
	}
	if client.Retries <= 0 {
		client.Retries = 1
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	defer func() {
		if client.Conn != nil {
			_ = client.Conn.Close()
		}
	}()
	max := cfg.Exploration.MaxOIDsPerDevice
	if max <= 0 {
		max = 300
	}
	metrics := make([]profiles.Metric, 0, len(safeCatalog))
	count := 0
	for _, cand := range safeCatalog {
		if count >= max {
			break
		}
		if cand.Type == "get" {
			pkt, err := client.Get([]string{cand.OID})
			if err != nil || pkt == nil || len(pkt.Variables) == 0 {
				continue
			}
			metrics = append(metrics, profiles.Metric{Metric: cand.Metric, OID: cand.OID, Type: cand.Type, Unit: cand.Unit, Index: cand.Index})
			count++
			continue
		}
		hit := false
		_ = client.Walk(cand.OID, func(pdu g.SnmpPDU) error {
			hit = true
			return fmt.Errorf("stop")
		})
		if hit {
			metrics = append(metrics, profiles.Metric{Metric: cand.Metric, OID: cand.OID, Type: cand.Type, Unit: cand.Unit, Index: cand.Index})
			count++
		}
	}
	return metrics, nil
}

func writeGeneratedProfile(configPath string, loaded config.Loaded, prof profiles.Profile) (string, error) {
	baseDir := filepath.Dir(configPath)
	outDir := config.ResolvePath(baseDir, loaded.Root.Discovery.Exploration.OutputDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	filename := normalizeName(prof.Name)
	if filename == "" {
		filename = "generated-profile"
	}
	path := filepath.Join(outDir, filename+".yml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	b, err := yaml.Marshal(prof)
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(outDir, fmt.Sprintf(".%s.%d.tmp", filename, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return "", err
	}
	if err := discovery.ChownGeneratedArtifact(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return path, nil
}

func expandCommunity(s string) string {
	s = strings.TrimSpace(os.ExpandEnv(s))
	if s == "" {
		return "public"
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", ""), "\t", ""))
}
