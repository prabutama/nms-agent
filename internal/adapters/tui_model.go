package adapters

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nms-agent/internal/models"
)

const hrStorageRamOID = "1.3.6.1.2.1.25.2.1.2"

func normalizeOIDString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, ".")
	return s
}

func (m tuiModel) memoryUsedBytes(deviceID string) (usedBytes float64, totalBytes float64, ok bool) {
	res, okRes := m.state.DeviceResourcesMap[deviceID]
	if !okRes || res.MemoryKB == nil {
		return 0, 0, false
	}
	memTotalBytes := *res.MemoryKB * 1024.0
	if memTotalBytes <= 0 {
		return 0, 0, false
	}
	byIdx := m.state.Storage[deviceID]
	if len(byIdx) == 0 {
		return 0, 0, false
	}

	// Prefer hrStorageRam entries when hrStorageType is available.
	bestDiff := math.MaxFloat64
	var best storageState
	found := false
	for _, st := range byIdx {
		if !st.HaveType {
			continue
		}
		if normalizeOIDString(st.TypeOID) != hrStorageRamOID {
			continue
		}
		if !st.HaveAlloc || !st.HaveSize || !st.HaveUsed {
			continue
		}
		sz := st.AllocBytes * st.SizeUnits
		if sz <= 0 {
			continue
		}
		diff := math.Abs(sz-memTotalBytes) / memTotalBytes
		if diff < bestDiff {
			bestDiff = diff
			best = st
			found = true
		}
	}

	// Fallback heuristic: closest total size to hrMemorySize.
	if !found {
		bestDiff = math.MaxFloat64
		for _, st := range byIdx {
			if !st.HaveAlloc || !st.HaveSize || !st.HaveUsed {
				continue
			}
			sz := st.AllocBytes * st.SizeUnits
			if sz <= 0 {
				continue
			}
			diff := math.Abs(sz-memTotalBytes) / memTotalBytes
			if diff < bestDiff {
				bestDiff = diff
				best = st
				found = true
			}
		}
		if !found || bestDiff > 0.30 {
			return 0, 0, false
		}
	}

	totalBytes = best.AllocBytes * best.SizeUnits
	usedBytes = best.AllocBytes * best.UsedUnits
	if totalBytes <= 0 {
		return 0, 0, false
	}
	return usedBytes, totalBytes, true
}

func (m tuiModel) memoryFreeLikeKB(deviceID string) (totalKB, usedKB, freeKB, sharedKB, buffCacheKB, availableKB, swapTotalKB, swapUsedKB, swapFreeKB float64, ok bool) {
	res, okRes := m.state.DeviceResourcesMap[deviceID]
	if !okRes || res.MemoryKB == nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	if *res.MemoryKB <= 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	if res.MemFreeKB == nil {
		// Without at least MemFree, this doesn't map to `free` output.
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, false
	}

	totalKB = *res.MemoryKB
	freeKB = *res.MemFreeKB
	if res.MemSharedKB != nil {
		sharedKB = *res.MemSharedKB
	}
	var bufferKB, cachedKB float64
	if res.MemBufferKB != nil {
		bufferKB = *res.MemBufferKB
	}
	if res.MemCachedKB != nil {
		cachedKB = *res.MemCachedKB
	}
	buffCacheKB = bufferKB + cachedKB

	// `free` semantics: used = total - available.
	if res.MemAvailableKB != nil {
		availableKB = *res.MemAvailableKB
	} else {
		// Fallback: best-effort estimate when memSysAvail is missing.
		availableKB = freeKB + buffCacheKB
	}
	if availableKB < 0 {
		availableKB = 0
	}
	if availableKB > totalKB {
		availableKB = totalKB
	}
	usedKB = totalKB - availableKB
	if usedKB < 0 {
		usedKB = 0
	}

	if res.SwapTotalKB != nil {
		swapTotalKB = *res.SwapTotalKB
	}
	if res.SwapFreeKB != nil {
		swapFreeKB = *res.SwapFreeKB
	}
	if swapTotalKB < 0 {
		swapTotalKB = 0
	}
	if swapFreeKB < 0 {
		swapFreeKB = 0
	}
	if swapFreeKB > swapTotalKB {
		swapFreeKB = swapTotalKB
	}
	swapUsedKB = swapTotalKB - swapFreeKB

	return totalKB, usedKB, freeKB, sharedKB, buffCacheKB, availableKB, swapTotalKB, swapUsedKB, swapFreeKB, true
}

type tuiFocus int

const (
	focusAlerts tuiFocus = iota
	focusIfaces
)

type deviceState struct {
	Reachable *bool
	LatencyMS *float64
	JitterMS  *float64
	LossPct   *float64
	LastSeen  time.Time
}

type ifaceState struct {
	Device   string
	IfIndex  string
	IfName   string
	RxBps    float64
	TxBps    float64
	UtilPct  float64
	LastSeen time.Time
}

type alertState struct {
	Device  string
	Metric  string
	Value   string
	Status  string
	Rule    string
	IfIndex string
	Updated time.Time
}

type deviceResources struct {
	CPU      *float64
	MemoryKB *float64

	// UCD-SNMP-MIB memory breakdown (optional, Linux/Proxmox).
	MemFreeKB      *float64
	MemAvailableKB *float64
	MemSharedKB    *float64
	MemBufferKB    *float64
	MemCachedKB    *float64
	SwapTotalKB    *float64
	SwapFreeKB     *float64

	CPUCores int
}

type storageState struct {
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

type telemetryBatchMsg []models.Telemetry
type tuiTickMsg time.Time

type tuiModel struct {
	keys tuiKeyMap
	help help.Model

	// Shared state (reducer)
	state *State

	// UI state
	ready    bool
	width    int
	height   int
	focus    tuiFocus
	showHelp bool
	fullHelp bool
	selected int // index into sorted device list
	listLen  int

	// Time / cycle
	cycle      int
	lastUpdate time.Time
	now        time.Time
	refresh    time.Duration

	// Derived view models
	alertsTable table.Model
	ifacesTable table.Model
}

func newTUIModel(refresh time.Duration) tuiModel {
	// Tables are created early; dimensions will be set after WindowSizeMsg.
	alertsCols := []table.Column{
		{Title: "Time", Width: 8},
		{Title: "Device", Width: 12},
		{Title: "Metric", Width: 26},
		{Title: "Value", Width: 10},
		{Title: "Status", Width: 8},
	}
	ifacesCols := []table.Column{
		{Title: "Device", Width: 12},
		{Title: "If", Width: 4},
		{Title: "RX", Width: 10},
		{Title: "TX", Width: 10},
		{Title: "Util", Width: 6},
	}

	alerts := table.New(table.WithColumns(alertsCols), table.WithFocused(true), table.WithHeight(10))
	ifaces := table.New(table.WithColumns(ifacesCols), table.WithFocused(false), table.WithHeight(10))

	// Improve visual selection a bit.
	as := table.DefaultStyles()
	as.Header = as.Header.Bold(true).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	as.Selected = as.Selected.Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	alerts.SetStyles(as)

	is := table.DefaultStyles()
	is.Header = is.Header.Bold(true).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	is.Selected = is.Selected.Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	ifaces.SetStyles(is)

	km := newTUIKeyMap()
	h := help.New()
	h.ShowAll = false

	return tuiModel{
		keys: km,
		help: h,

		state: NewState(),

		focus:       focusAlerts,
		refresh:     refresh,
		now:         time.Now().UTC(),
		alertsTable: alerts,
		ifacesTable: ifaces,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tuiTickMsg(t.UTC()) })
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layoutTables()
		return m, nil

	case tuiTickMsg:
		m.now = time.Time(msg)
		return m, tea.Tick(m.refresh, func(t time.Time) tea.Msg { return tuiTickMsg(t.UTC()) })

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.ToggleFullHelp):
			m.fullHelp = !m.fullHelp
			m.help.ShowAll = m.fullHelp
			return m, nil
		case key.Matches(msg, m.keys.SelectUp):
			m.selected--
			if m.selected < 0 {
				m.selected = 0
			}
			return m, nil
		case key.Matches(msg, m.keys.SelectDown):
			m.selected++
			if m.selected >= m.listLen {
				m.selected = m.listLen - 1
			}
			return m, nil
		case key.Matches(msg, m.keys.FocusNext):
			m.toggleFocus(true)
			return m, nil
		case key.Matches(msg, m.keys.FocusPrev):
			m.toggleFocus(false)
			return m, nil
		}

		// Pass keys to the focused table.
		var cmd tea.Cmd
		if m.focus == focusAlerts {
			m.alertsTable, cmd = m.alertsTable.Update(msg)
			return m, cmd
		}
		m.ifacesTable, cmd = m.ifacesTable.Update(msg)
		return m, cmd

	case telemetryBatchMsg:
		m.cycle++
		m.lastUpdate = time.Now().UTC()
		m.applyBatch([]models.Telemetry(msg))
		m.listLen = len(m.state.Devices)
		m.rebuildTables()
		return m, nil
	}

	// Let tables react to non-key msgs if needed.
	var cmd tea.Cmd
	if m.focus == focusAlerts {
		m.alertsTable, cmd = m.alertsTable.Update(msg)
	} else {
		m.ifacesTable, cmd = m.ifacesTable.Update(msg)
	}
	return m, cmd
}

func (m *tuiModel) toggleFocus(next bool) {
	if next {
		if m.focus == focusAlerts {
			m.focus = focusIfaces
		} else {
			m.focus = focusAlerts
		}
	} else {
		if m.focus == focusAlerts {
			m.focus = focusIfaces
		} else {
			m.focus = focusAlerts
		}
	}
}

func (m *tuiModel) applyBatch(batch []models.Telemetry) {
	m.state.ApplyBatch(batch)
}

func (m *tuiModel) rebuildTables() {
	// Alerts: sort by severity (critical first), then time desc.
	alerts := m.state.SortedAlerts()

	rows := make([]table.Row, 0, len(alerts))
	loc := getOutputLocation()
	for _, a := range alerts {
		rows = append(rows, table.Row{
			a.Updated.In(loc).Format("15:04:05"),
			a.Device,
			truncate(a.Metric, 26),
			truncate(a.Value, 10),
			strings.ToUpper(a.Status),
		})
	}
	m.alertsTable.SetRows(rows)

	// Ifaces: sort by utilization desc, then rx desc.
	ifaces := m.state.SortedIfaces()

	rows2 := make([]table.Row, 0, len(ifaces))
	for _, s := range ifaces {
		rows2 = append(rows2, table.Row{
			s.Device,
			s.IfIndex,
			formatBits(s.RxBps),
			formatBits(s.TxBps),
			fmt.Sprintf("%.1f%%", s.UtilPct),
		})
	}
	m.ifacesTable.SetRows(rows2)
}

func (m *tuiModel) layoutTables() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	available := m.height - 1 - 5 - 2
	if available < 6 {
		available = 6
	}

	m.alertsTable.SetHeight(available)
	m.alertsTable.SetWidth(m.width - 2)
	m.ifacesTable.SetHeight(available)
	m.ifacesTable.SetWidth(m.width - 2)
}

func sevRank(status string) int {
	switch status {
	case "critical":
		return 2
	case "warning":
		return 1
	default:
		return 0
	}
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	// naive truncate; keep ASCII-friendly.
	if max <= 1 {
		return s[:1]
	}
	return s[:max-1] + "…"
}
