package core

import (
	"context"
	"fmt"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/collectors"
	"nms-agent/internal/logger"
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
	observer  interface{ Update([]models.Telemetry) }
	log       *logger.Logger

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

func (p *Pipeline) SetObserver(obs interface{ Update([]models.Telemetry) }) {
	p.observer = obs
}

// SetLogger attaches a structured logger to the pipeline.
// If log is nil, logging is silently skipped (nil-safe).
func (p *Pipeline) SetLogger(l *logger.Logger) {
	p.log = l
}

func (p *Pipeline) logInfo(msg string, keysAndValues ...any) {
	if p.log != nil {
		p.log.Info(msg, keysAndValues...)
	}
}

func (p *Pipeline) logWarn(msg string, keysAndValues ...any) {
	if p.log != nil {
		p.log.Warn(msg, keysAndValues...)
	}
}

func (p *Pipeline) logError(msg string, keysAndValues ...any) {
	if p.log != nil {
		p.log.Error(msg, keysAndValues...)
	}
}

// RunOnce performs a single orchestration pass.
// Reliability rule: telemetry is persisted to the queue before any adapter delivery attempt.
func (p *Pipeline) RunOnce(ctx context.Context) error {
	cycleStart := time.Now()
	p.logInfo("cycle_start")

	raw, err := p.collector.Collect(ctx)
	if err != nil {
		p.logError("collect_failed", "error", err.Error(), "duration", time.Since(cycleStart).String())
		return err
	}
	p.logInfo("collect_done", "samples", len(raw), "duration", time.Since(cycleStart).String())

	telemetry, err := p.processor.Normalize(ctx, raw)
	if err != nil {
		p.logError("normalize_failed", "error", err.Error(), "duration", time.Since(cycleStart).String())
		return err
	}
	p.logInfo("normalize_done", "items", len(telemetry))

	if err := p.queue.EnqueueBatch(ctx, telemetry); err != nil {
		p.logError("enqueue_failed", "error", err.Error())
		return err
	}
	p.logInfo("enqueue_done", "items", len(telemetry))

	if p.observer != nil {
		p.observer.Update(telemetry)
	}

	var lastErr error
	batchesDone := 0
	totalDelivered := 0
	totalFailed := 0
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
			p.logWarn("delivery_failed", "batch", batchesDone, "items", len(batch), "error", err.Error())
			totalFailed += len(batch)
			if p.StopOnError {
				return err
			}
			lastErr = err
		} else {
			if err := p.queue.MarkDelivered(ctx, ids); err != nil {
				return err
			}
			p.logInfo("delivery_ok", "batch", batchesDone, "items", len(batch))
			totalDelivered += len(batch)
		}

		batchesDone++
		if !p.DrainEnabled {
			break
		}
		if p.MaxBatchesPerCycle > 0 && batchesDone >= p.MaxBatchesPerCycle {
			break
		}
	}

	p.logInfo("cycle_end",
		"delivered", totalDelivered,
		"failed", totalFailed,
		"batches", batchesDone,
		"last_error", lastErr,
		"duration", time.Since(cycleStart).String(),
	)

	return lastErr
}
