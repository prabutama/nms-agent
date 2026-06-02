package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nms-agent/internal/config"
	"nms-agent/internal/models"
	"nms-agent/internal/viewer"
)

func runView(args []string) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
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

	for {
		msg, err := cli.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
			return 1
		}

		if msg.Type == "snapshot" {
			renderSnapshot(msg.Adapter, msg.Telemetry, loc)
		} else if msg.Type == "telemetry" {
			renderTelemetry(msg.Adapter, msg.Telemetry, msg.At, loc)
		} else if msg.Type == "status" {
			renderStatus(msg.Adapter, msg.Status, msg.Details, msg.At, loc)
		}
	}
}

func renderSnapshot(adapter string, telemetry []models.Telemetry, loc *time.Location) {
	fmt.Fprintf(os.Stdout, "=== Snapshot (adapter: %s) ===\n", adapter)
	if adapter == "tui" {
		for _, t := range telemetry {
			ts := t.TS
			if loc != nil {
				ts = ts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "  [%s] device=%s metric=%s value=%s\n",
				ts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t))
		}
		fmt.Fprintln(os.Stdout, "  (TUI mode - use nms-agentctl view for terminal output)")
	} else if adapter == "terminal" {
		for _, t := range telemetry {
			ts := t.TS
			if loc != nil {
				ts = ts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "  %s device=%s metric=%s value=%s tags=%s\n",
				ts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
		}
	} else if adapter == "mqtt_generic" || adapter == "thingsboard_mqtt" {
		fmt.Fprintf(os.Stdout, "  adapter=%s telemetry_count=%d\n", adapter, len(telemetry))
		for _, t := range telemetry {
			ts := t.TS
			if loc != nil {
				ts = ts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "    %s device=%s metric=%s value=%s\n",
				ts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t))
		}
	} else {
		for _, t := range telemetry {
			ts := t.TS
			if loc != nil {
				ts = ts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "  %s device=%s metric=%s value=%s\n",
				ts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t))
		}
	}
	fmt.Fprintln(os.Stdout, "=== End Snapshot ===")
}

func renderTelemetry(adapter string, telemetry []models.Telemetry, at time.Time, loc *time.Location) {
	ts := at
	if loc != nil {
		ts = ts.In(loc)
	}
	if adapter == "tui" {
		fmt.Fprintf(os.Stdout, "[%s] TUI adapter received batch (count=%d)\n",
			ts.Format(time.RFC3339), len(telemetry))
	} else if adapter == "terminal" {
		for _, t := range telemetry {
			pts := t.TS
			if loc != nil {
				pts = pts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "[%s] device=%s metric=%s value=%s tags=%s\n",
				pts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
		}
	} else if adapter == "mqtt_generic" || adapter == "thingsboard_mqtt" {
		fmt.Fprintf(os.Stdout, "[%s] adapter=%s batch=%d devices=%s\n",
			ts.Format(time.RFC3339), adapter, len(telemetry), uniqueDevices(telemetry))
	} else {
		for _, t := range telemetry {
			pts := t.TS
			if loc != nil {
				pts = pts.In(loc)
			}
			fmt.Fprintf(os.Stdout, "[%s] device=%s metric=%s value=%s\n",
				pts.Format(time.RFC3339), t.DeviceID, t.Metric, formatValue(t))
		}
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
