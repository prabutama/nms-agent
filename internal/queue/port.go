package queue

import (
	"context"
	"time"

	"nms-agent/internal/models"
)

// QueueItem is a persisted telemetry record with queue metadata.
// IDs are required so deliveries can be acknowledged precisely.
type QueueItem struct {
	ID            string
	Telemetry     models.Telemetry
	RetryCount    int
	CreatedAt     time.Time
	NextAttemptAt time.Time
}

// Queue persists telemetry for store-and-forward delivery.
// Implementations (e.g., SQLite) must guarantee durability before adapter delivery.
type Queue interface {
	EnqueueBatch(ctx context.Context, batch []models.Telemetry) error
	PendingBatch(ctx context.Context, limit int) ([]QueueItem, error)
	MarkDelivered(ctx context.Context, ids []string) error
	MarkFailed(ctx context.Context, ids []string, reason string) error
}

// Observer allows read-only access to queue telemetry for local viewing.
type Observer interface {
	Snapshot(ctx context.Context, limit int) ([]QueueItem, error)
}
