package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"nms-agent/internal/models"
)

// MemoryQueue is an in-memory queue stub for Phase 3 demos.
// It is not durable and is not safe for multi-process usage.
type MemoryQueue struct {
	mu   sync.Mutex
	data []QueueItem
	next int
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (q *MemoryQueue) EnqueueBatch(ctx context.Context, batch []models.Telemetry) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, t := range batch {
		q.next++
		q.data = append(q.data, QueueItem{
			ID:         fmt.Sprintf("mem-%d", q.next),
			Telemetry:  t,
			RetryCount: 0,
			CreatedAt:  time.Now().UTC(),
		})
	}
	return nil
}

func (q *MemoryQueue) PendingBatch(ctx context.Context, limit int) ([]QueueItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if limit <= 0 || limit > len(q.data) {
		limit = len(q.data)
	}
	out := make([]QueueItem, limit)
	copy(out, q.data[:limit])
	return out, nil
}

func (q *MemoryQueue) MarkDelivered(ctx context.Context, ids []string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	toDelete := map[string]struct{}{}
	for _, id := range ids {
		toDelete[id] = struct{}{}
	}
	out := q.data[:0]
	for _, it := range q.data {
		if _, ok := toDelete[it.ID]; ok {
			continue
		}
		out = append(out, it)
	}
	q.data = out
	return nil
}

func (q *MemoryQueue) MarkFailed(ctx context.Context, ids []string, reason string) error {
	_ = reason
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, id := range ids {
		set[id] = struct{}{}
	}
	for i := range q.data {
		if _, ok := set[q.data[i].ID]; ok {
			q.data[i].RetryCount++
		}
	}
	return nil
}
