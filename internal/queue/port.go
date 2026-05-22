package queue

import (
	"context"

	"nms-agent/internal/models"
)

// Queue persists telemetry for store-and-forward delivery.
// Implementations (e.g., SQLite) must guarantee durability before adapter delivery.
type Queue interface {
	EnqueueBatch(ctx context.Context, batch []models.Telemetry) error
	PendingBatch(ctx context.Context, limit int) ([]models.Telemetry, error)
	MarkDelivered(ctx context.Context, delivered []models.Telemetry) error
}
