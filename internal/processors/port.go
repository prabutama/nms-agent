package processors

import (
	"context"

	"nms-agent/internal/models"
)

// Processor validates, preprocesses, and normalizes raw samples into canonical telemetry.
type Processor interface {
	Normalize(ctx context.Context, raw []models.RawSample) ([]models.Telemetry, error)
}
