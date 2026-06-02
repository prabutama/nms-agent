package adapters

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	if !m.ready {
		return "Starting NMS Agent TUI..."
	}
	if m.showHelp {
		return m.helpView()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.summaryView(),
		m.bodyView(),
		m.footerView(),
	)
}

func (m tuiModel) headerView() string {
	loc := getOutputLocation()
	last := "never"
	if !m.lastUpdate.IsZero() {
		last = m.lastUpdate.In(loc).Format("15:04:05")
	}
	age := ""
	if !m.lastUpdate.IsZero() {
		age = fmt.Sprintf("%s ago", m.now.Sub(m.lastUpdate).Round(time.Second))
	}
	selected := "none"
	if m.listLen > 0 && m.selected >= 0 && m.selected < m.listLen {
		devs := m.sortedDevices()
		if m.selected < len(devs) {
			selected = devs[m.selected]
		}
	}
	return styleHeader.Render(fmt.Sprintf(" NMS Agent | cycle %d | last %s (%s) | %s ", m.cycle, last, age, selected))
}

func (m tuiModel) summaryView() string {
	total, up, down, unknown := m.state.DeviceCounts()
	wc, cc := m.state.AlertCounts()

	return fmt.Sprintf("  Devices: total %d | up %d | down %d | unknown %d  |  Alerts: warning %d | critical %d\n",
		total, up, down, unknown, wc, cc)
}

func (m tuiModel) bodyView() string {
	// wide: device list left, detail right
	if m.width >= 80 {
		leftW := 38
		if leftW > m.width-30 {
			leftW = m.width / 2
		}
		if leftW < 30 {
			leftW = 30
		}
		rightW := m.width - leftW - 1
		if rightW < 20 {
			rightW = 20
		}
		left := lipgloss.NewStyle().Width(leftW).MaxWidth(leftW).Render(m.deviceListView(leftW))
		right := lipgloss.NewStyle().Width(rightW).MaxWidth(rightW).Render(m.deviceDetailView(rightW))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	}
	// narrow: stacked
	return lipgloss.JoinVertical(lipgloss.Left,
		m.deviceListView(m.width),
		m.deviceDetailView(m.width),
	)
}

func (m tuiModel) deviceListView(width int) string {
	devs := m.state.SortedDevices()
	if len(devs) == 0 {
		return styleDim.Render("  No devices yet")
	}

	var b string
	// keep header <= width to prevent wrapping into the detail pane
	devCol := 18
	statCol := 6
	alertCol := 9
	if width > 0 {
		// clamp device column so total fits
		maxDev := width - (2 + 1 + statCol + 1 + alertCol)
		if maxDev < 10 {
			maxDev = 10
		}
		if maxDev < devCol {
			devCol = maxDev
		}
	}

	b += styleTableHeader.Render(fmt.Sprintf("  %-*s %-*s %-*s\n", devCol, "Device", statCol, "Status", alertCol, "Alerts"))
	b += "  " + styleDivider.Render(strings.Repeat("-", devCol+1+statCol+1+alertCol+2)) + "\n"

	for i, d := range devs {
		ds := m.state.Devices[d]
		status := styleDim.Render("unknown")
		if ds.Reachable != nil {
			if *ds.Reachable {
				status = styleOK.Render("up")
			} else {
				status = styleCrit.Render("down")
			}
		}
		wc, cc := m.state.DeviceAlertCounts(d)
		alerts := "0"
		if wc > 0 || cc > 0 {
			alerts = fmt.Sprintf("W%d C%d", wc, cc)
		}

		name := truncatePlain(d, devCol)
		line := fmt.Sprintf("  %-*s %-*s %-*s", devCol, name, statCol, stripANSI(status), alertCol, alerts)
		if i == m.selected {
			b += styleSelected.Render(" > "+line[2:]) + "\n"
		} else {
			b += "   " + line[2:] + "\n"
		}
	}
	return b
}

func (m tuiModel) deviceDetailView(width int) string {
	if m.listLen == 0 || m.selected < 0 || m.selected >= m.listLen {
		return styleDim.Render("\n  Select a device to see details")
	}

	devs := m.state.SortedDevices()
	if m.selected >= len(devs) {
		m.selected = len(devs) - 1
	}

	name := devs[m.selected]
	ds := m.state.Devices[name]
	if ds.LastSeen.IsZero() {
		return styleDim.Render("\n  No data for this device yet")
	}

	var b string
	devTitle := fmt.Sprintf("  Device: %s", name)
	if width > 0 {
		devTitle = truncatePlain(devTitle, width)
	}
	b += styleTableHeader.Render(devTitle + "\n")
	sepW := 50
	if width > 0 && width-2 < sepW {
		sepW = width - 2
		if sepW < 10 {
			sepW = 10
		}
	}
	b += "  " + styleDivider.Render(strings.Repeat("-", sepW)) + "\n"

	// Health
	b += "\n  Health:\n"
	if ds.Reachable != nil {
		if *ds.Reachable {
			b += fmt.Sprintf("    Reachable: %s\n", styleOK.Render("YES"))
		} else {
			b += fmt.Sprintf("    Reachable: %s\n", styleCrit.Render("NO"))
		}
	} else {
		b += "    Reachable: unknown\n"
	}
	if ds.LatencyMS != nil {
		b += fmt.Sprintf("    Latency:   %.1f ms\n", *ds.LatencyMS)
	} else {
		b += "    Latency:   -\n"
	}
	if ds.JitterMS != nil {
		b += fmt.Sprintf("    Jitter:    %.1f ms\n", *ds.JitterMS)
	} else {
		b += "    Jitter:    -\n"
	}
	if ds.LossPct != nil {
		b += fmt.Sprintf("    Loss:      %.1f%%\n", *ds.LossPct)
	} else {
		b += "    Loss:      -\n"
	}
	loc := getOutputLocation()
	b += fmt.Sprintf("    Last seen: %s\n", ds.LastSeen.In(loc).Format("15:04:05"))

	// Host Resources
	b += "\n  Host Resources:\n"
	res := m.state.DeviceResources(name)
	if res.CPU != nil {
		b += fmt.Sprintf("    CPU Load:    %.1f%%\n", *res.CPU)
	} else {
		b += "    CPU Load:    -\n"
	}
	if res.MemoryKB != nil {
		b += fmt.Sprintf("    Memory Total: %s\n", formatMemoryFromKB(*res.MemoryKB))
	} else {
		b += "    Memory Total: -\n"
	}
	if totalKB, usedKB, freeKB, sharedKB, buffCacheKB, availKB, swapTotalKB, swapUsedKB, swapFreeKB, ok := m.memoryFreeLikeKB(name); ok {
		pct := 0.0
		if totalKB > 0 {
			pct = (usedKB / totalKB) * 100
		}
		b += fmt.Sprintf("    Memory Used:  %s (%.1f%%)\n", formatMemoryFromKB(usedKB), pct)
		head := "    total        used        free      shared  buff/cache   available"
		memLine := fmt.Sprintf("    Mem:   %-10s %-10s %-9s %-8s %-11s %-10s",
			formatMemoryFromKB(totalKB),
			formatMemoryFromKB(usedKB),
			formatMemoryFromKB(freeKB),
			formatMemoryFromKB(sharedKB),
			formatMemoryFromKB(buffCacheKB),
			formatMemoryFromKB(availKB),
		)
		swapLine := fmt.Sprintf("    Swap:  %-10s %-10s %-9s",
			formatMemoryFromKB(swapTotalKB),
			formatMemoryFromKB(swapUsedKB),
			formatMemoryFromKB(swapFreeKB),
		)
		if width > 0 {
			head = truncatePlain(head, width)
			memLine = truncatePlain(memLine, width)
			swapLine = truncatePlain(swapLine, width)
		}
		b += "\n" + head + "\n" + memLine + "\n" + swapLine + "\n"
	} else if used, total, ok := m.memoryUsedBytes(name); ok {
		pct := (used / total) * 100
		b += fmt.Sprintf("    Memory Used:  %s (%.1f%%)\n", formatMemoryFromBytes(used), pct)
	}

	// Alerts
	da := m.state.DeviceAlerts[name]
	if len(da) > 0 {
		b += "\n  Alerts:\n"
		for _, a := range da {
			status := a.Status
			var color string
			if status == "critical" {
				color = styleCrit.Render("CRITICAL")
			} else {
				color = styleWarn.Render("WARNING")
			}
			line := fmt.Sprintf("    - %s=%s %s  %s", a.Metric, a.Value, stripANSI(color), a.Updated.In(loc).Format("15:04:05"))
			if width > 0 {
				line = truncatePlain(line, width)
			}
			// re-apply severity coloring for the status token only
			if status == "critical" {
				line = strings.Replace(line, "CRITICAL", styleCrit.Render("CRITICAL"), 1)
			} else {
				line = strings.Replace(line, "WARNING", styleWarn.Render("WARNING"), 1)
			}
			b += line + "\n"
		}
	}

	// Interfaces (stable identity, sorted by ifName)
	ifaces := m.state.DeviceInterfaces(name)
	if len(ifaces) > 0 {
		b += "\n  Interfaces:\n"
		head := "Name       RX          TX          Util"
		if width > 0 {
			head = truncatePlain(head, width-4)
		}
		b += "    " + styleTableHeader.Render(head+"\n") + "    "
		b += styleDivider.Render(strings.Repeat("-", 42)) + "\n"
		for _, s := range ifaces {
			utilStr := fmt.Sprintf("%.1f%%", s.UtilPct)
			if s.UtilPct > 90 {
				utilStr = styleCrit.Render(utilStr)
			} else if s.UtilPct > 70 {
				utilStr = styleWarn.Render(utilStr)
			}
			ifName := s.IfName
			if ifName == "" {
				ifName = "if#" + s.IfIndex
			}
			line := fmt.Sprintf("    %-11s %-11s %-11s %s", ifName, formatBits(s.RxBps), formatBits(s.TxBps), stripANSI(utilStr))
			if width > 0 {
				line = truncatePlain(line, width)
			}
			if s.UtilPct > 90 {
				line = strings.Replace(line, fmt.Sprintf("%.1f%%", s.UtilPct), styleCrit.Render(fmt.Sprintf("%.1f%%", s.UtilPct)), 1)
			} else if s.UtilPct > 70 {
				line = strings.Replace(line, fmt.Sprintf("%.1f%%", s.UtilPct), styleWarn.Render(fmt.Sprintf("%.1f%%", s.UtilPct)), 1)
			}
			b += line + "\n"
		}
	}

	return b
}

func (m tuiModel) footerView() string {
	return styleFooter.Render("  [h] help  [tab] focus  [↑/↓] select device  [q] quit")
}

// truncatePlain truncates a plain string to max visible width.
// This intentionally assumes ASCII-ish input (device IDs, metrics).
func truncatePlain(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:1]
	}
	// naive byte truncate is fine for our ASCII-ish content.
	return s[:max-1] + "…"
}

// stripANSI removes ANSI escape codes for width-safe formatting.
func stripANSI(s string) string {
	// lipgloss strips ANSI for width calc; for replacement we just remove common sequences.
	// This is minimal and not a full ANSI parser.
	out := ""
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !inEsc {
			if c == 0x1b {
				inEsc = true
				continue
			}
			out += string(c)
			continue
		}
		// consume until 'm' or end
		if c == 'm' {
			inEsc = false
		}
	}
	return out
}

func (m tuiModel) helpView() string {
	return styleHeader.Render("  NMS Agent TUI Help\n\n" +
		"  Navigation:\n" +
		"    ↑ / ↓         select device in list\n" +
		"    tab           change focus\n" +
		"    enter         expand/collapse section\n\n" +
		"  Keys:\n" +
		"    q             quit\n" +
		"    h             toggle this help screen\n" +
		"    ?             toggle extended key hints\n")
}

func (m tuiModel) sortedDevices() []string {
	return m.state.SortedDevices()
}

func (m tuiModel) deviceAlertCounts(name string) (warning, critical int) {
	return m.state.DeviceAlertCounts(name)
}

func (m tuiModel) deviceInterfaces(name string) []ifaceState {
	return m.state.DeviceInterfaces(name)
}

func (m tuiModel) deviceResources(name string) deviceResources {
	return m.state.DeviceResources(name)
}

func (m tuiModel) deviceCounts() (total, up, down, unknown int) {
	return m.state.DeviceCounts()
}

func (m tuiModel) alertCounts() (warning, critical int) {
	return m.state.AlertCounts()
}
