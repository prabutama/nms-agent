package main

import (
	"flag"
	"fmt"
	"os"
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
	for {
		msg, err := cli.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
			return 1
		}

		if msg.Type == "snapshot" {
			if *mode == "summary" {
				state = adapters.NewStateFromTelemetry(msg.Telemetry)
				renderSummaryFromState(msg.Adapter, state, loc)
			} else {
				renderRawSnapshot(msg.Adapter, msg.Telemetry, loc)
			}
		} else if msg.Type == "telemetry" {
			if *mode == "summary" {
				if state == nil {
					state = adapters.NewState()
				}
				state.ApplyBatch(msg.Telemetry)
				renderSummaryFromState(msg.Adapter, state, loc)
			} else {
				renderRawTelemetry(msg.Adapter, msg.Telemetry, msg.At, loc)
			}
		} else if msg.Type == "status" {
			renderStatus(msg.Adapter, msg.Status, msg.Details, msg.At, loc)
		}
	}
}

func renderSummaryFromState(adapter string, st *adapters.State, loc *time.Location) {
	if st == nil {
		st = adapters.NewState()
	}
	loc2 := loc
	if loc2 == nil {
		loc2 = time.Local
	}

	fmt.Fprint(os.Stdout, "=== Summary (adapter: ")
	fmt.Fprint(os.Stdout, adapter)
	fmt.Fprint(os.Stdout, ") ===\n")

	total, up, down, unknown := st.DeviceCounts()
	fmt.Fprintf(os.Stdout, "  Devices: total=%d up=%d down=%d unknown=%d\n", total, up, down, unknown)

	if st.LastSeen != (time.Time{}) {
		fmt.Fprintf(os.Stdout, "  Last update: %s\n", st.LastSeen.In(loc2).Format(time.RFC3339))
	}

	fmt.Fprintln(os.Stdout, "=== End Summary ===")
}

func renderSummary(adapter string, telemetry []models.Telemetry, loc *time.Location) {
	st := adapters.NewStateFromTelemetry(telemetry)
	loc2 := loc
	if loc2 == nil {
		loc2 = time.Local
	}

	fmt.Fprintf(os.Stdout, "=== Summary (adapter: %s) ===\n", adapter)

	total, up, down, unknown := st.DeviceCounts()
	fmt.Fprintf(os.Stdout, "  Devices: total=%d up=%d down=%d unknown=%d\n", total, up, down, unknown)

	if len(telemetry) > 0 {
		fmt.Fprintf(os.Stdout, "  Last update: %s\n", st.LastSeen.In(loc2).Format(time.RFC3339))
	}

	fmt.Fprintln(os.Stdout, "=== End Summary ===")
}

func renderSummaryUpdate(adapter string, telemetry []models.Telemetry, at time.Time, loc *time.Location) {
	if len(telemetry) == 0 {
		return
	}
	loc2 := loc
	if loc2 == nil {
		loc2 = time.Local
	}
	ts := at.In(loc2)
	fmt.Fprintf(os.Stdout, "[%s] adapter=%s batch=%d devices=%s\n",
		ts.Format(time.RFC3339),
		adapter,
		len(telemetry),
		uniqueDevices(telemetry),
	)
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

func uniqueDevices(telemetry []models.Telemetry) string {
	devices := make(map[string]bool)
	for _, t := range telemetry {
		devices[t.DeviceID] = true
	}
	parts := make([]string, 0, len(devices))
	for d := range devices {
		parts = append(parts, d)
	}
	return fmt.Sprintf("%v", parts)
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
