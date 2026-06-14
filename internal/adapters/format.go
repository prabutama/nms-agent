package adapters

import (
	"time"

	"nms-agent/internal/adapters/base"
	"nms-agent/internal/models"
)

func FormatValue(t models.Telemetry) string {
	return base.FormatValue(t)
}

func FormatTags(tags map[string]string) string {
	return base.FormatTags(tags)
}

func FormatTS(ts time.Time) string {
	return base.FormatTS(ts)
}
