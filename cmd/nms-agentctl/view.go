package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/config"
	"nms-agent/internal/models"
	"nms-agent/internal/viewer"
)

func runView(args []string) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	mode := fs.String("mode", "summary", "View mode: summary or raw")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *mode != "summary" && *mode != "raw" {
		fmt.Fprintf(os.Stderr, "invalid mode %q (expected summary or raw)\n", *mode)
		return 2
	}

	loaded, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	loc, err := config.LoadLocation(loaded.Root.Agent.Output.Timezone)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	socketPath := "/run/nms-agent/view.sock"
	cli, err := viewer.Dial(socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to daemon: %v\n", err)
		return 1
	}
	defer cli.Close()

	var state *adapters.State
	interactiveSummary := *mode == "summary" && isInteractiveStdout()
	printedSummaryLines := 0
	for {
		msg, err := cli.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
			return 1
		}

		if msg.Type == "snapshot" {
			if *mode == "summary" {
				state = adapters.NewStateFromTelemetry(msg.Telemetry)
				printedSummaryLines = renderSummaryFromState(msg.Adapter, state, loc, interactiveSummary, printedSummaryLines)
			} else {
				renderRawSnapshot(msg.Adapter, msg.Telemetry, loc)
			}
		} else if msg.Type == "telemetry" {
			if *mode == "summary" {
				if state == nil {
					state = adapters.NewState()
				}
				state.ApplyBatch(msg.Telemetry)
				printedSummaryLines = renderSummaryFromState(msg.Adapter, state, loc, interactiveSummary, printedSummaryLines)
			} else {
				renderRawTelemetry(msg.Adapter, msg.Telemetry, msg.At, loc)
			}
		} else if msg.Type == "status" {
			if *mode == "summary" {
				printedSummaryLines = 0
			}
			renderStatus(msg.Adapter, msg.Status, msg.Details, msg.At, loc)
		}
	}
}

func renderSummaryFromState(adapter string, st *adapters.State, loc *time.Location, interactive bool, previousLines int) int {
	if st == nil {
		st = adapters.NewState()
	}
	loc2 := loc
	if loc2 == nil {
		loc2 = time.Local
	}
	if interactive && previousLines > 0 {
		fmt.Fprintf(os.Stdout, "\x1b[%dA\x1b[J", previousLines)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== Summary (adapter: %s) ===\n", adapter))

	total, up, down, unknown := st.DeviceCounts()
	warning, critical := st.AlertCounts()
	b.WriteString(fmt.Sprintf("  Devices: total=%d up=%d down=%d unknown=%d\n", total, up, down, unknown))
	b.WriteString(fmt.Sprintf("  Alerts: warning=%d critical=%d\n", warning, critical))

	if st.LastSeen != (time.Time{}) {
		b.WriteString(fmt.Sprintf("  Last update: %s\n", st.LastSeen.In(loc2).Format(time.RFC3339)))
	}
	if total > 0 {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-24s %-8s %-10s %-10s %-8s %-8s\n", "DEVICE", "STATUS", "LAST SEEN", "LATENCY", "LOSS", "ALERTS"))
		b.WriteString(fmt.Sprintf("  %-24s %-8s %-10s %-10s %-8s %-8s\n", strings.Repeat("-", 24), strings.Repeat("-", 8), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 8)))
		for _, name := range st.SortedDevices() {
			ds := st.Devices[name]
			status := deviceStatus(ds)
			lastSeen := "-"
			if !ds.LastSeen.IsZero() {
				lastSeen = ds.LastSeen.In(loc2).Format("15:04:05")
			}
			latency := "-"
			if ds.LatencyMS != nil {
				latency = fmt.Sprintf("%.1f ms", *ds.LatencyMS)
			}
			loss := "-"
			if ds.LossPct != nil {
				loss = fmt.Sprintf("%.0f%%", *ds.LossPct)
			}
			wc, cc := st.DeviceAlertCounts(name)
			alerts := "0"
			if wc > 0 || cc > 0 {
				parts := make([]string, 0, 2)
				if wc > 0 {
					parts = append(parts, fmt.Sprintf("W%d", wc))
				}
				if cc > 0 {
					parts = append(parts, fmt.Sprintf("C%d", cc))
				}
				alerts = strings.Join(parts, " ")
			}
			b.WriteString(fmt.Sprintf("  %-24s %-8s %-10s %-10s %-8s %-8s\n", truncateText(name, 24), status, lastSeen, latency, loss, alerts))
		}
	}

	b.WriteString("=== End Summary ===\n")
	output := b.String()
	fmt.Fprint(os.Stdout, output)
	return strings.Count(output, "\n")
}

func renderRawSnapshot(adapter string, telemetry []models.Telemetry, loc *time.Location) {
	fmt.Fprintf(os.Stdout, "=== Snapshot (adapter: %s) ===\n", adapter)
	for _, t := range telemetry {
		ts := t.TS
		if loc != nil {
			ts = ts.In(loc)
		}
		fmt.Fprintf(os.Stdout, "  %s device=%s metric=%s value=%s tags=%s\n",
			ts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
	}
	fmt.Fprintln(os.Stdout, "=== End Snapshot ===")
}

func renderRawTelemetry(adapter string, telemetry []models.Telemetry, at time.Time, loc *time.Location) {
	ts := at
	if loc != nil {
		ts = ts.In(loc)
	}
	for _, t := range telemetry {
		pts := t.TS
		if loc != nil {
			pts = pts.In(loc)
		}
		fmt.Fprintf(os.Stdout, "[%s] device=%s metric=%s value=%s tags=%s\n",
			pts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
	}
}

func renderStatus(adapter, status, details string, at time.Time, loc *time.Location) {
	ts := at
	if loc != nil {
		ts = ts.In(loc)
	}
	fmt.Fprintf(os.Stdout, "[%s] adapter=%s status=%s", ts.Format(time.RFC3339), adapter, status)
	if details != "" {
		fmt.Fprintf(os.Stdout, " details=%s", details)
	}
	fmt.Fprintln(os.Stdout)
}

func deviceStatus(ds struct {
	Reachable *bool
	LatencyMS *float64
	JitterMS  *float64
	LossPct   *float64
	LastSeen  time.Time
}) string {
	if ds.Reachable == nil {
		return "unknown"
	}
	if *ds.Reachable {
		return "up"
	}
	return "down"
}

func truncateText(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func isInteractiveStdout() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func formatValue(t models.Telemetry) string {
	if t.ValueType == "string" && t.ValueString != nil {
		return fmt.Sprintf("%q", *t.ValueString)
	}
	if t.ValueType == "number" && t.ValueNumber != nil {
		return fmt.Sprintf("%v", *t.ValueNumber)
	}
	return ""
}

func formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(tags))
	for k, v := range tags {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return "{" + fmt.Sprintf("%v", parts) + "}"
}
