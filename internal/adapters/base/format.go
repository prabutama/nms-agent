package base

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"nms-agent/internal/models"
)

func FormatValue(t models.Telemetry) string {
	if t.ValueType == "string" && t.ValueString != nil {
		return fmt.Sprintf("%q", *t.ValueString)
	}
	if t.ValueType == "number" && t.ValueNumber != nil {
		return fmt.Sprintf("%v", *t.ValueNumber)
	}
	return ""
}

func FormatTags(tags map[string]string) string {
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

func FormatTS(ts time.Time) string {
	loc := GetOutputLocation()
	if ts.IsZero() {
		return time.Now().In(loc).Format(time.RFC3339)
	}
	return ts.In(loc).Format(time.RFC3339)
}
