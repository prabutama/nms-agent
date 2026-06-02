package viewer

import (
	"context"
	"sync"

	"nms-agent/internal/models"
)

// Hub stores the latest telemetry snapshot and broadcasts live updates.
type Hub struct {
	mu          sync.RWMutex
	adapter     string
	snapshot    []models.Telemetry
	subscribers map[chan []models.Telemetry]struct{}
	provider    SnapshotProvider
}

// SnapshotProvider can supply telemetry snapshot on demand.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, limit int) ([]models.Telemetry, error)
}

func NewHub(adapter string) *Hub {
	return &Hub{
		adapter:     adapter,
		subscribers: map[chan []models.Telemetry]struct{}{},
	}
}

func (h *Hub) SetAdapter(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.adapter = name
}

func (h *Hub) SetSnapshot(snap []models.Telemetry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshot = append([]models.Telemetry(nil), snap...)
}

func (h *Hub) Adapter() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.adapter
}

func (h *Hub) Snapshot() []models.Telemetry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]models.Telemetry(nil), h.snapshot...)
}

func (h *Hub) SetProvider(p SnapshotProvider) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.provider = p
}

func (h *Hub) SnapshotFromProvider(ctx context.Context, limit int) []models.Telemetry {
	h.mu.RLock()
	p := h.provider
	h.mu.RUnlock()
	if p == nil {
		return h.Snapshot()
	}
	items, err := p.Snapshot(ctx, limit)
	if err != nil {
		return h.Snapshot()
	}
	return items
}

func (h *Hub) Update(batch []models.Telemetry) {
	if len(batch) == 0 {
		return
	}
	h.mu.Lock()
	// keep only the latest batch for snapshot (small and testable for now)
	h.snapshot = append([]models.Telemetry(nil), batch...)
	for ch := range h.subscribers {
		select {
		case ch <- batch:
		default:
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Subscribe() chan []models.Telemetry {
	ch := make(chan []models.Telemetry, 4)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan []models.Telemetry) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	close(ch)
	h.mu.Unlock()
}
