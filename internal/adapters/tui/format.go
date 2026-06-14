package tui

import (
	"fmt"
	"math"
)

func formatBits(bps float64) string {
	switch {
	case bps >= 1_000_000_000:
		return fmt.Sprintf("%.1f Gbps", bps/1_000_000_000)
	case bps >= 1_000_000:
		return fmt.Sprintf("%.1f Mbps", bps/1_000_000)
	case bps >= 1_000:
		return fmt.Sprintf("%.0f Kbps", bps/1_000)
	default:
		return fmt.Sprintf("%.0f bps", bps)
	}
}

func formatMemoryFromKB(kb float64) string {
	if kb < 0 {
		kb = 0
	}
	val := kb
	unit := "Ki"
	if val >= 1024 {
		val = val / 1024
		unit = "Mi"
	}
	if val >= 1024 {
		val = val / 1024
		unit = "Gi"
	}
	if val >= 1024 {
		val = val / 1024
		unit = "Ti"
	}

	if unit == "Gi" || unit == "Ti" {
		if math.Abs(val-math.Round(val)) < 0.05 {
			return fmt.Sprintf("%.0f%s", math.Round(val), unit)
		}
		return fmt.Sprintf("%.1f%s", val, unit)
	}
	if unit == "Mi" {
		return fmt.Sprintf("%.0f%s", math.Round(val), unit)
	}
	return fmt.Sprintf("%.0f%s", math.Round(val), unit)
}

func formatMemoryFromBytes(bytes float64) string {
	return formatMemoryFromKB(bytes / 1024.0)
}
