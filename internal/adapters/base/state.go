package base

import (
	"sort"
	"strings"
	"time"

	"nms-agent/internal/models"
)

type DeviceState struct {
	Reachable *bool
	LatencyMS *float64
	JitterMS  *float64
	LossPct   *float64
	LastSeen  time.Time
}

type IfaceState struct {
	Device   string
	IfIndex  string
	IfName   string
	RxBps    float64
	TxBps    float64
	UtilPct  float64
	LastSeen time.Time
}

type AlertState struct {
	Device  string
	Metric  string
	Value   string
	Status  string
	Rule    string
	IfIndex string
	Updated time.Time
}

type DeviceResources struct {
	CPU      *float64
	MemoryKB *float64

	MemFreeKB      *float64
	MemAvailableKB *float64
	MemSharedKB    *float64
	MemBufferKB    *float64
	MemCachedKB    *float64
	SwapTotalKB    *float64
	SwapFreeKB     *float64

	CPUCores int
}

type StorageState struct {
	AllocBytes float64
	SizeUnits  float64
	UsedUnits  float64
	HaveAlloc  bool
	HaveSize   bool
	HaveUsed   bool
	TypeOID    string
	HaveType   bool
	Desc       string
	HaveDesc   bool
}

type State struct {
	Devices               map[string]DeviceState
	Ifaces                map[string]IfaceState
	DeviceAlerts          map[string][]AlertState
	DeviceResourcesMap    map[string]DeviceResources
	Storage               map[string]map[string]StorageState
	Alerts                map[string]AlertState
	LastBatchMetricCounts map[string]int
	LastBatchMetricTotal  int
	Cycle                 int
	LastSeen              time.Time
}

func NewState() *State {
	return &State{
		Devices:               make(map[string]DeviceState),
		Ifaces:                make(map[string]IfaceState),
		DeviceAlerts:          make(map[string][]AlertState),
		DeviceResourcesMap:    make(map[string]DeviceResources),
		Storage:               make(map[string]map[string]StorageState),
		Alerts:                make(map[string]AlertState),
		LastBatchMetricCounts: make(map[string]int),
	}
}

func NewStateFromTelemetry(batch []models.Telemetry) *State {
	s := NewState()
	s.ApplyBatch(batch)
	return s
}

func (s *State) ApplyBatch(batch []models.Telemetry) {
	s.LastSeen = time.Now().UTC()
	s.LastBatchMetricCounts = make(map[string]int)
	s.LastBatchMetricTotal = 0
	for _, t := range batch {
		if t.DeviceID == "" {
			continue
		}
		s.LastBatchMetricCounts[t.DeviceID]++
		s.LastBatchMetricTotal++
		s.Cycle++

		ds := s.Devices[t.DeviceID]
		ds.LastSeen = time.Now().UTC()

		if t.ValueType == "number" && t.ValueNumber != nil {
			if t.Metric == "icmp.reachable" {
				v := *t.ValueNumber > 0
				ds.Reachable = &v
			}
			if t.Metric == "icmp.latency_ms" {
				v := *t.ValueNumber
				ds.LatencyMS = &v
			}
			if t.Metric == "icmp.jitter_ms" {
				v := *t.ValueNumber
				ds.JitterMS = &v
			}
			if t.Metric == "icmp.packet_loss_pct" {
				v := *t.ValueNumber
				ds.LossPct = &v
			}
		}
		s.Devices[t.DeviceID] = ds

		status := t.Tags["threshold.status"]
		if status == "warning" || status == "critical" {
			ifIdx := t.Tags["ifIndex"]
			key := t.DeviceID + "|" + t.Metric + "|" + ifIdx + "|" + t.Tags["threshold.rule"]
			s.Alerts[key] = AlertState{
				Device:  t.DeviceID,
				Metric:  t.Metric,
				Value:   FormatValue(t),
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

		if t.ValueType == "number" && t.ValueNumber != nil {
			v := *t.ValueNumber
			switch t.Metric {
			case "snmp.if.rx_bps":
				ifIdx := t.Tags["ifIndex"]
				if ifIdx != "" {
					key := t.DeviceID + "|" + ifIdx
					is := s.Ifaces[key]
					is.Device = t.DeviceID
					is.IfIndex = ifIdx
					is.RxBps = v
					is.LastSeen = time.Now().UTC()
					if v := t.Tags["ifName"]; v != "" {
						is.IfName = v
					}
					s.Ifaces[key] = is
				}
			case "snmp.if.tx_bps":
				ifIdx := t.Tags["ifIndex"]
				if ifIdx != "" {
					key := t.DeviceID + "|" + ifIdx
					is := s.Ifaces[key]
					is.Device = t.DeviceID
					is.IfIndex = ifIdx
					is.TxBps = v
					is.LastSeen = time.Now().UTC()
					if v := t.Tags["ifName"]; v != "" {
						is.IfName = v
					}
					s.Ifaces[key] = is
				}
			case "snmp.if.rx_utilization_pct":
				ifIdx := t.Tags["ifIndex"]
				if ifIdx != "" {
					key := t.DeviceID + "|" + ifIdx
					is := s.Ifaces[key]
					is.UtilPct = v
					s.Ifaces[key] = is
				}
			case "snmp.host.cpu.load_pct":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.CPU = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.size_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemoryKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.free_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemFreeKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.available_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemAvailableKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.shared_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemSharedKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.buffer_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemBufferKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.memory.cached_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.MemCachedKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.swap.total_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.SwapTotalKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			case "snmp.host.swap.free_kb":
				res := s.DeviceResourcesMap[t.DeviceID]
				res.SwapFreeKB = &v
				s.DeviceResourcesMap[t.DeviceID] = res
			}
		}

		if strings.HasPrefix(t.Metric, "snmp.host.storage.") {
			index := strings.TrimSpace(t.Tags["ifIndex"])
			if index != "" {
				store := s.Storage[t.DeviceID]
				if store == nil {
					store = make(map[string]StorageState)
					s.Storage[t.DeviceID] = store
				}
				st := store[index]
				switch t.Metric {
				case "snmp.host.storage.allocation_units":
					if t.ValueType == "number" && t.ValueNumber != nil {
						st.AllocBytes = *t.ValueNumber
						st.HaveAlloc = true
					}
				case "snmp.host.storage.size_units":
					if t.ValueType == "number" && t.ValueNumber != nil {
						st.SizeUnits = *t.ValueNumber
						st.HaveSize = true
					}
				case "snmp.host.storage.used_units":
					if t.ValueType == "number" && t.ValueNumber != nil {
						st.UsedUnits = *t.ValueNumber
						st.HaveUsed = true
					}
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
				store[index] = st
			}
		}
	}
}

func (s *State) DeviceCounts() (total, up, down, unknown int) {
	total = len(s.Devices)
	for _, d := range s.Devices {
		if d.Reachable != nil {
			if *d.Reachable {
				up++
			} else {
				down++
			}
		} else {
			unknown++
		}
	}
	return
}

func (s *State) AlertCounts() (warning, critical int) {
	for _, a := range s.Alerts {
		switch a.Status {
		case "warning":
			warning++
		case "critical":
			critical++
		}
	}
	return
}

func (s *State) DeviceAlertCounts(deviceID string) (warning, critical int) {
	for _, a := range s.Alerts {
		if a.Device != deviceID {
			continue
		}
		switch a.Status {
		case "warning":
			warning++
		case "critical":
			critical++
		}
	}
	return
}

func (s *State) DeviceMetricCount(deviceID string) int {
	return s.LastBatchMetricCounts[deviceID]
}

func (s *State) TotalMetricCount() int {
	return s.LastBatchMetricTotal
}

func (s *State) SortedDevices() []string {
	out := make([]string, 0, len(s.Devices))
	for name := range s.Devices {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *State) DeviceResources(deviceID string) DeviceResources {
	return s.DeviceResourcesMap[deviceID]
}

func (s *State) DeviceInterfaces(deviceID string) []IfaceState {
	matched := make([]IfaceState, 0, len(s.Ifaces))
	for _, ifs := range s.Ifaces {
		if ifs.Device != deviceID {
			continue
		}
		matched = append(matched, ifs)
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].UtilPct != matched[j].UtilPct {
			return matched[i].UtilPct > matched[j].UtilPct
		}
		return matched[i].RxBps > matched[j].RxBps
	})
	if len(matched) > 200 {
		matched = matched[:200]
	}
	return matched
}

func (s *State) SortedAlerts() []AlertState {
	sorted := make([]AlertState, 0, len(s.Alerts))
	for _, a := range s.Alerts {
		sorted = append(sorted, a)
	}
	sort.Slice(sorted, func(i, j int) bool {
		ri := severityRank(sorted[i].Status)
		rj := severityRank(sorted[j].Status)
		if ri != rj {
			return ri > rj
		}
		return sorted[i].Updated.After(sorted[j].Updated)
	})
	if len(sorted) > 50 {
		sorted = sorted[:50]
	}
	return sorted
}

func (s *State) SortedIfaces() []IfaceState {
	ifaces := make([]IfaceState, 0, len(s.Ifaces))
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

func severityRank(status string) int {
	switch status {
	case "critical":
		return 2
	case "warning":
		return 1
	default:
		return 0
	}
}
