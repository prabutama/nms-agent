package adapters

import (
	"sort"
	"strings"
	"time"

	"nms-agent/internal/models"
)

// State holds the aggregated monitoring state derived from telemetry batches.
// It is shared between the Bubble Tea TUI adapter and the CLI summary renderer.
type State struct {
	Devices            map[string]deviceState
	Ifaces             map[string]ifaceState
	DeviceAlerts       map[string][]alertState
	DeviceResourcesMap map[string]deviceResources
	Storage            map[string]map[string]storageState
	Alerts             map[string]alertState
	Cycle              int
	LastUpdate         time.Time
	LastSeen           time.Time
}

// NewState creates an empty State.
func NewState() *State {
	return &State{
		Devices:            map[string]deviceState{},
		Ifaces:             map[string]ifaceState{},
		DeviceAlerts:       map[string][]alertState{},
		DeviceResourcesMap: map[string]deviceResources{},
		Storage:            map[string]map[string]storageState{},
		Alerts:             map[string]alertState{},
		LastUpdate:         time.Now().UTC(),
	}
}

// ApplyBatch reduces a telemetry batch into the state.
func (s *State) ApplyBatch(batch []models.Telemetry) {
	if len(batch) == 0 {
		return
	}
	s.LastSeen = time.Now().UTC()
	for _, t := range batch {
		ds := s.Devices[t.DeviceID]
		ds.LastSeen = time.Now().UTC()

		if t.Metric == "icmp.reachable" && t.ValueType == "number" && t.ValueNumber != nil {
			r := *t.ValueNumber > 0
			ds.Reachable = &r
		}
		if t.ValueType == "number" && t.ValueNumber != nil {
			switch t.Metric {
			case "icmp.latency_ms":
				v := *t.ValueNumber
				ds.LatencyMS = &v
			case "icmp.jitter_ms":
				v := *t.ValueNumber
				ds.JitterMS = &v
			case "icmp.packet_loss_pct":
				v := *t.ValueNumber
				ds.LossPct = &v
			}
		}
		s.Devices[t.DeviceID] = ds

		status := t.Tags["threshold.status"]
		if status == "warning" || status == "critical" {
			ifIdx := t.Tags["ifIndex"]
			key := t.DeviceID + "|" + t.Metric + "|" + ifIdx + "|" + t.Tags["threshold.rule"]
			s.Alerts[key] = alertState{
				Device:  t.DeviceID,
				Metric:  t.Metric,
				Value:   formatValue(t),
				Status:  status,
				Rule:    t.Tags["threshold.rule"],
				IfIndex: ifIdx,
				Updated: time.Now().UTC(),
			}
			da := s.DeviceAlerts[t.DeviceID]
			da = append(da, s.Alerts[key])
			if len(da) > 10 {
				da = da[:10]
			}
			s.DeviceAlerts[t.DeviceID] = da
		}

		if strings.HasPrefix(t.Metric, "snmp.host.storage.") {
			idx := t.Tags["ifIndex"]
			if idx != "" {
				byIdx := s.Storage[t.DeviceID]
				if byIdx == nil {
					byIdx = map[string]storageState{}
				}
				st := byIdx[idx]
				switch t.Metric {
				case "snmp.host.storage.type":
					if t.ValueType == "string" && t.ValueString != nil {
						st.TypeOID = *t.ValueString
						st.HaveType = true
					}
				case "snmp.host.storage.description":
					if t.ValueType == "string" && t.ValueString != nil {
						st.Desc = *t.ValueString
						st.HaveDesc = true
					}
				}
				byIdx[idx] = st
				s.Storage[t.DeviceID] = byIdx
			}
		}

		if t.Metric == "snmp.if.name" && t.ValueString != nil {
			ifIdx := t.Tags["ifIndex"]
			if ifIdx == "" {
				continue
			}
			ik := t.DeviceID + "/" + ifIdx
			is := s.Ifaces[ik]
			is.IfName = *t.ValueString
			is.Device = t.DeviceID
			is.IfIndex = ifIdx
			is.LastSeen = time.Now().UTC()
			s.Ifaces[ik] = is
			continue
		}

		if t.ValueType == "number" && t.ValueNumber != nil {
			if strings.HasPrefix(t.Metric, "snmp.host.storage.") {
				idx := t.Tags["ifIndex"]
				if idx != "" {
					byIdx := s.Storage[t.DeviceID]
					if byIdx == nil {
						byIdx = map[string]storageState{}
					}
					st := byIdx[idx]
					switch t.Metric {
					case "snmp.host.storage.allocation_units":
						st.AllocBytes = *t.ValueNumber
						st.HaveAlloc = true
					case "snmp.host.storage.size_units":
						st.SizeUnits = *t.ValueNumber
						st.HaveSize = true
					case "snmp.host.storage.used_units":
						st.UsedUnits = *t.ValueNumber
						st.HaveUsed = true
					}
					byIdx[idx] = st
					s.Storage[t.DeviceID] = byIdx
				}
			}

			res := s.DeviceResourcesMap[t.DeviceID]
			switch t.Metric {
			case "snmp.host.cpu.load_pct":
				res.CPUCores++
				val := *t.ValueNumber
				if res.CPU == nil {
					res.CPU = &val
				} else {
					avg := (*res.CPU*float64(res.CPUCores-1) + val) / float64(res.CPUCores)
					res.CPU = &avg
				}
			case "snmp.host.memory.size_kb":
				val := *t.ValueNumber
				res.MemoryKB = &val
			case "snmp.host.memory.free_kb":
				val := *t.ValueNumber
				res.MemFreeKB = &val
			case "snmp.host.memory.available_kb":
				val := *t.ValueNumber
				res.MemAvailableKB = &val
			case "snmp.host.memory.shared_kb":
				val := *t.ValueNumber
				res.MemSharedKB = &val
			case "snmp.host.memory.buffer_kb":
				val := *t.ValueNumber
				res.MemBufferKB = &val
			case "snmp.host.memory.cached_kb":
				val := *t.ValueNumber
				res.MemCachedKB = &val
			case "snmp.host.swap.total_kb":
				val := *t.ValueNumber
				res.SwapTotalKB = &val
			case "snmp.host.swap.free_kb":
				val := *t.ValueNumber
				res.SwapFreeKB = &val
			}
			s.DeviceResourcesMap[t.DeviceID] = res
		}

		ifIdx := t.Tags["ifIndex"]
		if ifIdx == "" {
			continue
		}
		if !strings.HasPrefix(t.Metric, "snmp.if.") {
			continue
		}
		ik := t.DeviceID + "/" + ifIdx
		is := s.Ifaces[ik]
		is.Device = t.DeviceID
		is.IfIndex = ifIdx
		is.LastSeen = time.Now().UTC()
		if t.ValueType == "number" && t.ValueNumber != nil {
			if t.Metric == "snmp.if.rx_bps" {
				is.RxBps = *t.ValueNumber
			}
			if t.Metric == "snmp.if.tx_bps" {
				is.TxBps = *t.ValueNumber
			}
			if strings.HasSuffix(t.Metric, "_utilization_pct") {
				is.UtilPct = *t.ValueNumber
			}
		}
		s.Ifaces[ik] = is
	}
}

// DeviceCounts returns total, up, down, unknown device counts.
func (s *State) DeviceCounts() (total, up, down, unknown int) {
	total = len(s.Devices)
	for _, d := range s.Devices {
		if d.Reachable == nil {
			unknown++
			continue
		}
		if *d.Reachable {
			up++
		} else {
			down++
		}
	}
	return total, up, down, unknown
}

// AlertCounts returns warning and critical alert counts.
func (s *State) AlertCounts() (warning, critical int) {
	for _, a := range s.Alerts {
		switch a.Status {
		case "warning":
			warning++
		case "critical":
			critical++
		}
	}
	return warning, critical
}

// SortedDevices returns device names sorted alphabetically.
func (s *State) SortedDevices() []string {
	devs := make([]string, 0, len(s.Devices))
	for name := range s.Devices {
		devs = append(devs, name)
	}
	sort.Strings(devs)
	return devs
}

// DeviceAlertCounts returns warning and critical counts for a device.
func (s *State) DeviceAlertCounts(name string) (warning, critical int) {
	for _, a := range s.DeviceAlerts[name] {
		switch a.Status {
		case "warning":
			warning++
		case "critical":
			critical++
		}
	}
	return warning, critical
}

// DeviceResources returns resources for a device.
func (s *State) DeviceResources(name string) deviceResources {
	res, ok := s.DeviceResourcesMap[name]
	if !ok {
		return deviceResources{}
	}
	return res
}

// DeviceInterfaces returns interfaces for a device, sorted by ifName.
func (s *State) DeviceInterfaces(name string) []ifaceState {
	var result []ifaceState
	for ik, s := range s.Ifaces {
		if s.Device == name {
			result = append(result, s)
			_ = ik
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IfName != result[j].IfName {
			return result[i].IfName < result[j].IfName
		}
		return result[i].IfIndex < result[j].IfIndex
	})
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// SortedAlerts returns alerts sorted by severity desc, then time desc.
func (s *State) SortedAlerts() []alertState {
	alerts := make([]alertState, 0, len(s.Alerts))
	for _, a := range s.Alerts {
		alerts = append(alerts, a)
	}
	sort.Slice(alerts, func(i, j int) bool {
		ai, aj := alerts[i], alerts[j]
		if sevRank(ai.Status) != sevRank(aj.Status) {
			return sevRank(ai.Status) > sevRank(aj.Status)
		}
		return ai.Updated.After(aj.Updated)
	})
	if len(alerts) > 200 {
		alerts = alerts[:200]
	}
	return alerts
}

// SortedIfaces returns interfaces sorted by utilization desc, then rx desc.
func (s *State) SortedIfaces() []ifaceState {
	ifaces := make([]ifaceState, 0, len(s.Ifaces))
	for _, s := range s.Ifaces {
		ifaces = append(ifaces, s)
	}
	sort.Slice(ifaces, func(i, j int) bool {
		if ifaces[i].UtilPct != ifaces[j].UtilPct {
			return ifaces[i].UtilPct > ifaces[j].UtilPct
		}
		return ifaces[i].RxBps > ifaces[j].RxBps
	})
	if len(ifaces) > 200 {
		ifaces = ifaces[:200]
	}
	return ifaces
}

// NewStateFromTelemetry creates a State and applies a telemetry batch.
func NewStateFromTelemetry(batch []models.Telemetry) *State {
	s := NewState()
	s.ApplyBatch(batch)
	return s
}
