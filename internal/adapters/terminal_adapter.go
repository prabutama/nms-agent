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
}

func NewTerminalAdapter() *TerminalAdapter {
	return &TerminalAdapter{Out: os.Stdout}
}

func (a *TerminalAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	_ = ctx
	if a.Out == nil {
		a.Out = os.Stdout
	}
	for _, t := range batch {
		fmt.Fprintf(a.Out, "%s device=%s metric=%s value=%v tags=%s\n",
			formatTS(t.TS), t.DeviceID, t.Metric, t.Value, formatTags(t.Tags))
	}
	return nil
}

func formatTS(ts time.Time) string {
	if ts.IsZero() {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return ts.UTC().Format(time.RFC3339)
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
