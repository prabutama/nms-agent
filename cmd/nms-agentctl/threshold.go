package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"nms-agent/internal/config"
	"nms-agent/internal/models"
)

func runThreshold(args []string) int {
	if len(args) < 1 {
		thresholdUsage()
		return 2
	}
	switch args[0] {
	case "list":
		return runThresholdList(args[1:])
	case "set":
		return runThresholdSet(args[1:])
	default:
		thresholdUsage()
		return 2
	}
}

func thresholdUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold list --config configs/agent.yml")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold set --config configs/agent.yml --metric <name> --operator <op> [--warning <val>] [--critical <val>] [--tags k=v,k2=v2]")
}

func runThresholdList(args []string) int {
	fs := flag.NewFlagSet("threshold list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	thPath, err := resolveThresholdsPath(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	rules, err := loadThresholds(thPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	for i, r := range rules {
		var parts []string
		parts = append(parts, fmt.Sprintf("metric=%s", r.Metric))
		parts = append(parts, fmt.Sprintf("operator=%s", r.Operator))
		if r.Warning != nil {
			parts = append(parts, fmt.Sprintf("warning=%v", *r.Warning))
		}
		if r.Critical != nil {
			parts = append(parts, fmt.Sprintf("critical=%v", *r.Critical))
		}
		if len(r.Tags) > 0 {
			tagParts := make([]string, 0, len(r.Tags))
			for k, v := range r.Tags {
				tagParts = append(tagParts, k+"="+v)
			}
			sort.Strings(tagParts)
			parts = append(parts, "tags="+strings.Join(tagParts, ","))
		}
		fmt.Fprintf(os.Stdout, "%d. %s\n", i, strings.Join(parts, " "))
	}
	return 0
}

func runThresholdSet(args []string) int {
	fs := flag.NewFlagSet("threshold set", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/agent.yml", "Path to agent.yml")
	metric := fs.String("metric", "", "Metric name (required)")
	operator := fs.String("operator", "", "Operator: >, >=, <, <=, ==, !=")
	warningRaw := fs.String("warning", "", "Warning threshold (optional, numeric)")
	criticalRaw := fs.String("critical", "", "Critical threshold (optional, numeric)")
	tagsRaw := fs.String("tags", "", "Comma-separated k=v pairs")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *metric == "" {
		fmt.Fprintln(os.Stderr, "--metric is required")
		return 2
	}
	if *operator == "" {
		fmt.Fprintln(os.Stderr, "--operator is required")
		return 2
	}
	if *warningRaw == "" && *criticalRaw == "" {
		fmt.Fprintln(os.Stderr, "at least one of --warning or --critical is required")
		return 2
	}

	var warning *float64
	if *warningRaw != "" {
		v, err := strconv.ParseFloat(*warningRaw, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --warning value %q: %v\n", *warningRaw, err)
			return 2
		}
		warning = &v
	}
	var critical *float64
	if *criticalRaw != "" {
		v, err := strconv.ParseFloat(*criticalRaw, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --critical value %q: %v\n", *criticalRaw, err)
			return 2
		}
		critical = &v
	}

	tags := parseTags(*tagsRaw)

	rule := models.ThresholdRule{
		Metric:   *metric,
		Operator: *operator,
		Warning:  warning,
		Critical: critical,
		Tags:     tags,
	}

	if err := config.ValidateThresholdRules([]models.ThresholdRule{rule}); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	thPath, err := resolveThresholdsPath(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	rules, err := loadThresholds(thPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	rules = upsertRule(rules, rule)

	if err := config.ValidateThresholdRules(rules); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if err := saveThresholds(thPath, rules); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Fprintf(os.Stdout, "threshold set: metric=%s\n", *metric)
	return 0
}

func resolveThresholdsPath(agentCfgPath string) (string, error) {
	abs, err := filepath.Abs(agentCfgPath)
	if err != nil {
		return "", err
	}
	cfg, err := config.LoadFromFile(abs)
	if err != nil {
		return "", err
	}
	baseDir := filepath.Dir(abs)
	return config.ResolvePath(baseDir, cfg.Root.Paths.ThresholdsFile), nil
}

func loadThresholds(path string) ([]models.ThresholdRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config.ThresholdsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Thresholds == nil {
		return []models.ThresholdRule{}, nil
	}
	return cfg.Thresholds, nil
}

func saveThresholds(path string, rules []models.ThresholdRule) error {
	cfg := config.ThresholdsConfig{Thresholds: rules}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "thresholds.*.yml")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}

	_ = os.Remove(path)
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

func upsertRule(rules []models.ThresholdRule, rule models.ThresholdRule) []models.ThresholdRule {
	for i, r := range rules {
		if r.Metric == rule.Metric && tagsEqual(r.Tags, rule.Tags) {
			rules[i].Operator = rule.Operator
			if rule.Warning != nil {
				rules[i].Warning = rule.Warning
			}
			if rule.Critical != nil {
				rules[i].Critical = rule.Critical
			}
			return rules
		}
	}
	return append(rules, rule)
}

func tagsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func parseTags(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	tags := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k != "" && v != "" {
				tags[k] = v
			}
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}
