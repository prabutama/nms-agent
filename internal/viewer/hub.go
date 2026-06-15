package viewer

import (
	"context"
	"sync"
	"time"

	"nms-agent/internal/models"
)

// Hub stores the latest telemetry snapshot and broadcasts live updates.
type Hub struct {
	mu          sync.RWMutex
	adapter     string
	snapshot    []models.Telemetry
	active      map[string]struct{}
	subscribers map[chan Message]struct{}
	provider    SnapshotProvider
}

// SnapshotProvider can supply telemetry snapshot on demand.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, limit int) ([]models.Telemetry, error)
}

func NewHub(adapter string) *Hub {
	return &Hub{
		adapter:     adapter,
		active:      map[string]struct{}{},
		subscribers: map[chan Message]struct{}{},
	}
}

func (h *Hub) SetAdapter(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.adapter = name
}

func (h *Hub) SetSnapshot(snap []models.Telemetry) {
	filtered := h.filterActive(snap)
	h.mu.Lock()
	h.snapshot = append([]models.Telemetry(nil), filtered...)
	adapter := h.adapter
	h.mu.Unlock()
	h.broadcast(Message{Type: "snapshot", Adapter: adapter, Telemetry: filtered, At: time.Now().UTC()})
}

func (h *Hub) SetActiveDevices(ids []string) {
	active := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		active[id] = struct{}{}
	}
	h.mu.Lock()
	h.active = active
	h.snapshot = filterTelemetryByActive(h.snapshot, active)
	adapter := h.adapter
	snapshot := append([]models.Telemetry(nil), h.snapshot...)
	h.mu.Unlock()
	h.broadcast(Message{Type: "snapshot", Adapter: adapter, Telemetry: snapshot, At: time.Now().UTC()})
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
	return h.filterActive(items)
}

func (h *Hub) Update(batch []models.Telemetry) {
	batch = h.filterActive(batch)
	if len(batch) == 0 {
		return
	}
	h.mergeSnapshot(batch)
	h.broadcast(Message{Type: "telemetry", Adapter: h.Adapter(), Telemetry: batch, At: time.Now().UTC()})
}

func (h *Hub) mergeSnapshot(batch []models.Telemetry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	snap := make([]models.Telemetry, 0, len(h.snapshot)+len(batch))
	keyMap := make(map[string]bool, len(h.snapshot))
	for _, t := range h.snapshot {
		k := telemetryKey(t)
		keyMap[k] = true
		snap = append(snap, t)
	}
	for _, t := range batch {
		k := telemetryKey(t)
		if keyMap[k] {
			for i, existing := range snap {
				if telemetryKey(existing) == k {
					snap[i] = t
					break
				}
			}
		} else {
			keyMap[k] = true
			snap = append(snap, t)
		}
	}
	h.snapshot = snap
}

func telemetryKey(t models.Telemetry) string {
	if len(t.Tags) == 0 {
		return t.DeviceID + "|" + t.Metric
	}
	return t.DeviceID + "|" + t.Metric + "|" + t.Tags["ifIndex"]
}

func (h *Hub) Subscribe() chan Message {
	ch := make(chan Message, 8)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan Message) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *Hub) UpdateStatus(status string, details string) {
	h.broadcast(Message{Type: "status", Adapter: h.Adapter(), Status: status, Details: details, At: time.Now().UTC()})
}

func (h *Hub) broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Hub) filterActive(batch []models.Telemetry) []models.Telemetry {
	h.mu.RLock()
	active := make(map[string]struct{}, len(h.active))
	for id := range h.active {
		active[id] = struct{}{}
	}
	h.mu.RUnlock()
	return filterTelemetryByActive(batch, active)
}

func filterTelemetryByActive(batch []models.Telemetry, active map[string]struct{}) []models.Telemetry {
	if len(batch) == 0 {
		return nil
	}
	if len(active) == 0 {
		return append([]models.Telemetry(nil), batch...)
	}
	out := make([]models.Telemetry, 0, len(batch))
	for _, t := range batch {
		if _, ok := active[t.DeviceID]; ok {
			out = append(out, t)
		}
	}
	return out
}
