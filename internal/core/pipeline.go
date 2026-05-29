package core

import (
	"context"
	"fmt"

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

	MaxBatch           int
	DrainEnabled       bool
	MaxBatchesPerCycle int
	StopOnError        bool
}

type DeliveryConfig struct {
	MaxBatch           int
	DrainEnabled       bool
	MaxBatchesPerCycle int
	StopOnError        bool
}

func NewPipeline(c collectors.Collector, p processors.Processor, q queue.Queue, a adapters.Adapter, dc DeliveryConfig) *Pipeline {
	if dc.MaxBatch <= 0 {
		dc.MaxBatch = 100
	}
	if dc.MaxBatchesPerCycle <= 0 {
		dc.MaxBatchesPerCycle = 10
	}
	return &Pipeline{
		collector:          c,
		processor:          p,
		queue:              q,
		adapter:            a,
		MaxBatch:           dc.MaxBatch,
		DrainEnabled:       dc.DrainEnabled,
		MaxBatchesPerCycle: dc.MaxBatchesPerCycle,
		StopOnError:        dc.StopOnError,
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

	var lastErr error
	batchesDone := 0
	for {
		pending, err := p.queue.PendingBatch(ctx, p.MaxBatch)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			break
		}

		ids := make([]string, 0, len(pending))
		batch := make([]models.Telemetry, 0, len(pending))
		for _, it := range pending {
			ids = append(ids, it.ID)
			batch = append(batch, it.Telemetry)
		}

		if err := p.adapter.SendBatch(ctx, batch); err != nil {
			if err := p.queue.MarkFailed(ctx, ids, err.Error()); err != nil {
				return fmt.Errorf("mark failed: %w (send: %v)", err, err)
			}
			if p.StopOnError {
				return err
			}
			lastErr = err
		} else {
			if err := p.queue.MarkDelivered(ctx, ids); err != nil {
				return err
			}
		}

		batchesDone++
		if !p.DrainEnabled {
			break
		}
		if p.MaxBatchesPerCycle > 0 && batchesDone >= p.MaxBatchesPerCycle {
			break
		}
	}

	return lastErr
}
