package queue

import (
	"context"
	"sync"

	"nms-agent/internal/models"
)

// MemoryQueue is an in-memory queue stub for Phase 3 demos.
// It is not durable and is not safe for multi-process usage.
type MemoryQueue struct {
	mu   sync.Mutex
	data []models.Telemetry
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (q *MemoryQueue) EnqueueBatch(ctx context.Context, batch []models.Telemetry) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.data = append(q.data, batch...)
	return nil
}

func (q *MemoryQueue) PendingBatch(ctx context.Context, limit int) ([]models.Telemetry, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if limit <= 0 || limit > len(q.data) {
		limit = len(q.data)
	}
	out := make([]models.Telemetry, limit)
	copy(out, q.data[:limit])
	return out, nil
}

func (q *MemoryQueue) MarkDelivered(ctx context.Context, delivered []models.Telemetry) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(delivered) == 0 {
		return nil
	}
	if len(delivered) >= len(q.data) {
		q.data = nil
		return nil
	}
	q.data = q.data[len(delivered):]
	return nil
}
