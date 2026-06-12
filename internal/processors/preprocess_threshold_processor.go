package processors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nms-agent/internal/models"
)

// PreprocessThresholdProcessor applies minimal normalization and threshold tags.
// It keeps the canonical Telemetry contract unchanged.
type PreprocessThresholdProcessor struct {
	Rules []models.ThresholdRule

	lastCounters  map[counterKey]counterSample
	lastSpeed     map[speedKey]float64
	lastHighSpeed map[speedKey]float64
	lastIfaceInfo map[speedKey]ifaceInfo
}

type ifaceInfo struct {
	ifType           int
	ifName           string
	connectorPresent int // 0=unknown, 1=true, 2=false
}

func (p *PreprocessThresholdProcessor) Normalize(ctx context.Context, raw []models.RawSample) ([]models.Telemetry, error) {
	telemetry, err := PassthroughProcessor{}.Normalize(ctx, raw)
	if err != nil {
		return nil, err
	}

	p.cacheIfaceInfo(telemetry)

	derived := p.deriveInterfaceMetrics(telemetry)
	telemetry = append(telemetry, derived...)

	telemetry = p.filterPhysicalInterfaces(telemetry)

	telemetry = p.normalizeMetrics(telemetry)

	for i := range telemetry {
		t := &telemetry[i]
		if t.ValueType != "number" || t.ValueNumber == nil {
			continue
		}
		applyThresholds(t, p.Rules)
	}
	return telemetry, nil
}

type counterKey struct {
	deviceID  string
	ifIndex   string
	direction string // rx|tx
}

type speedKey struct {
	deviceID string
	ifIndex  string
}

type counterSample struct {
	value float64
	ts    time.Time
	tags  map[string]string
}

func (p *PreprocessThresholdProcessor) deriveInterfaceMetrics(telemetry []models.Telemetry) []models.Telemetry {
	if p.lastCounters == nil {
		p.lastCounters = map[counterKey]counterSample{}
	}
	if p.lastSpeed == nil {
		p.lastSpeed = map[speedKey]float64{}
	}
	if p.lastHighSpeed == nil {
		p.lastHighSpeed = map[speedKey]float64{}
	}

	current := map[counterKey]counterSample{}
	priority := map[counterKey]int{}

	for _, t := range telemetry {
		if t.ValueType != "number" || t.ValueNumber == nil {
			continue
		}
		ifIndex := t.Tags["ifIndex"]
		if ifIndex == "" {
			continue
		}
		// Capture speed when present.
		if t.Metric == "snmp.if.speed_bps" && *t.ValueNumber > 0 {
			p.lastSpeed[speedKey{deviceID: t.DeviceID, ifIndex: ifIndex}] = *t.ValueNumber
			continue
		}
		// Capture high-speed as fallback; only if speed_bps hasn't been recorded yet.
		if t.Metric == "snmp.if.high_speed_mbps" && *t.ValueNumber > 0 {
			sk := speedKey{deviceID: t.DeviceID, ifIndex: ifIndex}
			if p.lastSpeed[sk] == 0 {
				p.lastHighSpeed[sk] = *t.ValueNumber
			}
			continue
		}

		direction, prio := counterDirection(t.Metric)
		if direction == "" {
			continue
		}
		key := counterKey{deviceID: t.DeviceID, ifIndex: ifIndex, direction: direction}
		if prio < priority[key] {
			continue
		}
		priority[key] = prio
		current[key] = counterSample{value: *t.ValueNumber, ts: t.TS, tags: t.Tags}
	}

	derived := make([]models.Telemetry, 0, len(current)*2)
	for key, cur := range current {
		prev, ok := p.lastCounters[key]
		if ok {
			delta := cur.value - prev.value
			dt := cur.ts.Sub(prev.ts).Seconds()
			if delta > 0 && dt > 0 {
				bps := (delta * 8) / dt
				metric := "snmp.if.rx_bps"
				utilMetric := "snmp.if.rx_utilization_pct"
				if key.direction == "tx" {
					metric = "snmp.if.tx_bps"
					utilMetric = "snmp.if.tx_utilization_pct"
				}
				baseTags := baseIfaceTags(cur.tags)
				baseTags["unit"] = "bps"
				derived = append(derived, numberTelemetry(key.deviceID, metric, cur.ts, bps, baseTags))

				sk := speedKey{deviceID: key.deviceID, ifIndex: key.ifIndex}
				speed := p.lastSpeed[sk]
				if speed == 0 {
					if highMbps := p.lastHighSpeed[sk]; highMbps > 0 {
						speed = highMbps * 1_000_000
					}
				}
				if speed > 0 {
					util := (bps / speed) * 100
					utilTags := baseIfaceTags(cur.tags)
					utilTags["unit"] = "pct"
					derived = append(derived, numberTelemetry(key.deviceID, utilMetric, cur.ts, util, utilTags))
				}
			}
		}
	}

	for key, cur := range current {
		p.lastCounters[key] = cur
	}
	return derived
}

func (p *PreprocessThresholdProcessor) cacheIfaceInfo(telemetry []models.Telemetry) {
	if p.lastIfaceInfo == nil {
		p.lastIfaceInfo = map[speedKey]ifaceInfo{}
	}
	for _, t := range telemetry {
		ifIndex := t.Tags["ifIndex"]
		if ifIndex == "" {
			continue
		}
		sk := speedKey{deviceID: t.DeviceID, ifIndex: ifIndex}
		info := p.lastIfaceInfo[sk]

		switch t.Metric {
		case "snmp.if.type":
			if t.ValueType == "number" && t.ValueNumber != nil {
				info.ifType = int(*t.ValueNumber)
			}
		case "snmp.if.name":
			if t.ValueType == "string" && t.ValueString != nil {
				info.ifName = *t.ValueString
			}
		case "snmp.if.connector_present":
			if t.ValueType == "number" && t.ValueNumber != nil {
				info.connectorPresent = int(*t.ValueNumber)
			}
		}
		p.lastIfaceInfo[sk] = info
	}
}

func (p *PreprocessThresholdProcessor) filterPhysicalInterfaces(telemetry []models.Telemetry) []models.Telemetry {
	out := make([]models.Telemetry, 0, len(telemetry))
	for _, t := range telemetry {
		ifIndex := t.Tags["ifIndex"]
		if ifIndex != "" {
			info, known := p.lastIfaceInfo[speedKey{deviceID: t.DeviceID, ifIndex: ifIndex}]
			if known && !p.isPhysicalInterface(info) {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

func (p *PreprocessThresholdProcessor) isPhysicalInterface(info ifaceInfo) bool {
	// Rule 1: ifName matches known virtual pattern => drop.
	if info.ifName != "" && isVirtualIfaceName(info.ifName) {
		return false
	}
	// Rule 2: ifConnectorPresent=true => strong physical signal.
	if info.connectorPresent == 1 {
		return true
	}
	// Rule 3: ifConnectorPresent explicitly false => drop.
	if info.connectorPresent == 2 {
		return false
	}
	// Rule 4: ifType allowlist (6=ethernet, 71=wifi).
	if info.ifType == 6 || info.ifType == 71 {
		return true
	}
	// Rule 5: ifType known but not in allowlist => drop.
	if info.ifType != 0 {
		return false
	}
	// Rule 6: totally unknown signals => keep (safe default).
	return true
}

var virtualIfacePrefixes = []string{
	"vmbr", "tap", "veth", "fwbr", "fwpr", "fwln",
	"docker", "br-", "cni", "flannel", "cali", "virbr",
	"lo", "dummy", "tun", "bond", "team", "sit", "gre",
	"wg", "tailscale", "zt", "vnet",
}

func isVirtualIfaceName(name string) bool {
	for _, prefix := range virtualIfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func counterDirection(metric string) (string, int) {
	switch metric {
	case "snmp.if.hc_in_octets":
		return "rx", 2
	case "snmp.if.hc_out_octets":
		return "tx", 2
	case "snmp.if.in_octets":
		return "rx", 1
	case "snmp.if.out_octets":
		return "tx", 1
	default:
		return "", 0
	}
}

func baseIfaceTags(tags map[string]string) map[string]string {
	out := map[string]string{}
	if tags == nil {
		return out
	}
	if v := tags["source"]; v != "" {
		out["source"] = v
	}
	if v := tags["ifIndex"]; v != "" {
		out["ifIndex"] = v
	}
	return out
}

func numberTelemetry(deviceID, metric string, ts time.Time, value float64, tags map[string]string) models.Telemetry {
	val := value
	return models.Telemetry{
		DeviceID:    deviceID,
		Metric:      metric,
		TS:          ts,
		ValueType:   "number",
		ValueNumber: &val,
		Tags:        tags,
	}
}

func (p *PreprocessThresholdProcessor) normalizeMetrics(telemetry []models.Telemetry) []models.Telemetry {
	for i := range telemetry {
		t := &telemetry[i]
		if t.ValueType != "number" || t.ValueNumber == nil {
			continue
		}
		metric := t.Metric
		val := *t.ValueNumber

		if t.Tags == nil {
			t.Tags = map[string]string{}
		}

		switch {
		case strings.HasSuffix(metric, "_pct"):
			if val < 0 {
				val = 0
			} else if val > 100 {
				val = 100
			}
			if t.Tags["unit"] == "" {
				t.Tags["unit"] = "pct"
			}

		case strings.HasSuffix(metric, "_ms"):
			if val < 0 {
				val = 0
			}
			if t.Tags["unit"] == "" {
				t.Tags["unit"] = "ms"
			}

		case strings.HasSuffix(metric, "_seconds"):
			if val < 0 {
				val = 0
			}
			if t.Tags["unit"] == "" {
				t.Tags["unit"] = "s"
			}

		case strings.HasSuffix(metric, "_bps"):
			if val < 0 {
				val = 0
			}
			if t.Tags["unit"] == "" {
				t.Tags["unit"] = "bps"
			}
		}

		// icmp.reachable: normalise to 0 or 1.
		if metric == "icmp.reachable" {
			if *t.ValueNumber > 0 {
				val = 1
			} else {
				val = 0
			}
		}

		t.ValueNumber = &val
	}
	return telemetry
}

func applyThresholds(t *models.Telemetry, rules []models.ThresholdRule) {
	if t == nil || len(rules) == 0 {
		return
	}
	if t.ValueType != "number" || t.ValueNumber == nil {
		return
	}
	if t.Tags == nil {
		t.Tags = map[string]string{}
	}

	matched := false
	status := "normal"
	var ruleName string

	for i, r := range rules {
		if strings.TrimSpace(r.Metric) != t.Metric {
			continue
		}
		if !tagsMatch(t.Tags, r.Tags) {
			continue
		}
		matched = true
		sev := evalRule(*t.ValueNumber, r)
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
	} else if len(rules) > 0 {
		t.Tags["threshold.status"] = status
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
