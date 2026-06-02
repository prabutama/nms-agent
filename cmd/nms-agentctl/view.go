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

	socketPath := "/run/nms-agent/view.sock"
	if loaded.Root.Paths.QueueDB != "" {
		socketPath = "/run/nms-agent/view.sock"
	}

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
			fmt.Fprintf(os.Stdout, "=== Snapshot (adapter: %s) ===\n", msg.Adapter)
			for _, t := range msg.Telemetry {
				fmt.Fprintf(os.Stdout, "%s device=%s metric=%s value=%s tags=%s\n",
					formatTS(t.TS), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
			}
			fmt.Fprintln(os.Stdout, "=== End Snapshot ===")
		} else if msg.Type == "telemetry" {
			fmt.Fprintf(os.Stdout, "[%s] device=%s metric=%s value=%s tags=%s\n",
				msg.At.Format(time.RFC3339),
				msg.Telemetry[0].DeviceID,
				msg.Telemetry[0].Metric,
				formatValue(msg.Telemetry[0]),
				formatTags(msg.Telemetry[0].Tags),
			)
		}
	}
}

func formatTS(ts time.Time) string {
	if ts.IsZero() {
		return time.Now().Format(time.RFC3339)
	}
	return ts.Format(time.RFC3339)
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
