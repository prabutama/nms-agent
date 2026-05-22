package adapters

import (
	"context"

	"nms-agent/internal/models"
)

// Adapter sends telemetry to a consumer platform.
// All platform-specific behavior must be implemented behind this contract.
type Adapter interface {
	SendBatch(ctx context.Context, batch []models.Telemetry) error
}
