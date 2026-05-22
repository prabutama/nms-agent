package core

import (
	"context"

	"nms-agent/internal/adapters"
	"nms-agent/internal/collectors"
	"nms-agent/internal/models"
	"nms-agent/internal/processors"
	"nms-agent/internal/queue"
)

// Pipeline is the platform-agnostic orchestrator enforcing the required flow:
// collect -> preprocess/normalize -> queue -> adapter send.
type Pipeline struct {
	collector collectors.Collector
	processor processors.Processor
	queue     queue.Queue
	adapter   adapters.Adapter

	// MaxDeliveryBatch bounds one delivery attempt.
	MaxDeliveryBatch int
}

func NewPipeline(c collectors.Collector, p processors.Processor, q queue.Queue, a adapters.Adapter) *Pipeline {
	return &Pipeline{
		collector:        c,
		processor:        p,
		queue:            q,
		adapter:          a,
		MaxDeliveryBatch: 100,
	}
}

// RunOnce performs a single orchestration pass.
// Reliability rule: telemetry is persisted to the queue before any adapter delivery attempt.
func (p *Pipeline) RunOnce(ctx context.Context) error {
	raw, err := p.collector.Collect(ctx)
	if err != nil {
		return err
	}

	telemetry, err := p.processor.Normalize(ctx, raw)
	if err != nil {
		return err
	}

	if err := p.queue.EnqueueBatch(ctx, telemetry); err != nil {
		return err
	}

	pending, err := p.queue.PendingBatch(ctx, p.MaxDeliveryBatch)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	ids := make([]string, 0, len(pending))
	batch := make([]models.Telemetry, 0, len(pending))
	for _, it := range pending {
		ids = append(ids, it.ID)
		batch = append(batch, it.Telemetry)
	}

	if err := p.adapter.SendBatch(ctx, batch); err != nil {
		// Delivery failed: keep data pending and increment retry count.
		_ = p.queue.MarkFailed(ctx, ids, err.Error())
		return err
	}

	return p.queue.MarkDelivered(ctx, ids)
}
