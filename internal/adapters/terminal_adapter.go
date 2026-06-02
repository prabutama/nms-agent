package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"nms-agent/internal/models"
)

// TerminalAdapter prints canonical telemetry to a writer (stdout by default).
type TerminalAdapter struct {
	Out io.Writer
	obs AdapterObserver
}

func NewTerminalAdapter() *TerminalAdapter {
	return &TerminalAdapter{Out: os.Stdout}
}

func (a *TerminalAdapter) SetObserver(hub AdapterObserver) {
	a.obs = hub
}

func (a *TerminalAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	_ = ctx
	if a.Out == nil {
		a.Out = os.Stdout
	}
	for _, t := range batch {
		fmt.Fprintf(a.Out, "%s device=%s metric=%s value=%s tags=%s\n",
			formatTS(t.TS), t.DeviceID, t.Metric, formatValue(t), formatTags(t.Tags))
	}
	if a.obs != nil {
		a.obs.Update(batch)
	}
	return nil
}

func formatTS(ts time.Time) string {
	loc := getOutputLocation()
	if ts.IsZero() {
		return time.Now().In(loc).Format(time.RFC3339)
	}
	return ts.In(loc).Format(time.RFC3339)
}

func formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, tags[k]))
	}
	return "{" + strings.Join(parts, ",") + "}"
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
