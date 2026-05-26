package processors

import (
	"context"
	"fmt"
	"strings"

	"nms-agent/internal/models"
)

// PreprocessThresholdProcessor applies minimal normalization and threshold tags.
// It keeps the canonical Telemetry contract unchanged.
type PreprocessThresholdProcessor struct {
	Rules []models.ThresholdRule
}

func (p PreprocessThresholdProcessor) Normalize(ctx context.Context, raw []models.RawSample) ([]models.Telemetry, error) {
	telemetry, err := PassthroughProcessor{}.Normalize(ctx, raw)
	if err != nil {
		return nil, err
	}

	for i := range telemetry {
		t := &telemetry[i]
		applyThresholds(t, p.Rules)
	}
	return telemetry, nil
}

func applyThresholds(t *models.Telemetry, rules []models.ThresholdRule) {
	if t == nil || len(rules) == 0 {
		return
	}
	if t.Tags == nil {
		t.Tags = map[string]string{}
	}

	matched := false
	status := "ok"
	var ruleName string

	for i, r := range rules {
		if strings.TrimSpace(r.Metric) != t.Metric {
			continue
		}
		if !tagsMatch(t.Tags, r.Tags) {
			continue
		}
		matched = true
		sev := evalRule(t.Value, r)
		if sev == "critical" {
			status = "critical"
			ruleName = fmt.Sprintf("%s#%d", r.Metric, i)
			break
		}
		if sev == "warning" {
			status = "warning"
			ruleName = fmt.Sprintf("%s#%d", r.Metric, i)
		}
	}

	if matched {
		t.Tags["threshold.status"] = status
		t.Tags["threshold.matched"] = "true"
		if ruleName != "" {
			t.Tags["threshold.rule"] = ruleName
		}
	}
}

func evalRule(value float64, r models.ThresholdRule) string {
	// Evaluate critical first to allow higher severity to win.
	if r.Critical != nil && compare(value, *r.Critical, r.Operator) {
		return "critical"
	}
	if r.Warning != nil && compare(value, *r.Warning, r.Operator) {
		return "warning"
	}
	return "ok"
}

func compare(value, target float64, op string) bool {
	switch op {
	case ">":
		return value > target
	case ">=":
		return value >= target
	case "<":
		return value < target
	case "<=":
		return value <= target
	case "==":
		return value == target
	case "!=":
		return value != target
	default:
		return false
	}
}

func tagsMatch(tags map[string]string, ruleTags map[string]string) bool {
	if len(ruleTags) == 0 {
		return true
	}
	for k, v := range ruleTags {
		if v == "*" {
			if _, ok := tags[k]; !ok {
				return false
			}
			continue
		}
		if tags[k] != v {
			return false
		}
	}
	return true
}
