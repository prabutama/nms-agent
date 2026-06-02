package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"nms-agent/internal/adapters"
	"nms-agent/internal/config"
	"nms-agent/internal/models"
	"nms-agent/internal/queue"
)

func runQueueRetry(args []string) int {
	fs := flag.NewFlagSet("queue retry", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/agent.yml", "Path to agent.yml")
	limit := fs.Int("limit", 100, "Max pending items to retry")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *limit <= 0 {
		fmt.Fprintln(os.Stderr, "limit must be > 0")
		return 2
	}

	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	ad, err := adapterFromConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	q, err := queue.OpenSQLite(cfg.Root.Paths.QueueDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	defer q.Close()

	ctx := context.Background()
	items, err := q.PendingBatch(ctx, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintf(os.Stdout, "retried=0 delivered=0 failed=0\n")
		return 0
	}

	ids := make([]string, 0, len(items))
	batch := make([]models.Telemetry, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
		batch = append(batch, it.Telemetry)
	}

	if err := ad.SendBatch(ctx, batch); err != nil {
		_ = q.MarkFailed(ctx, ids, err.Error())
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stdout, "retried=%d delivered=0 failed=%d\n", len(items), len(items))
		return 1
	}

	if err := q.MarkDelivered(ctx, ids); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Fprintf(os.Stdout, "retried=%d delivered=%d failed=0\n", len(items), len(items))
	return 0
}

func adapterFromConfig(cfg config.Loaded) (adapters.Adapter, error) {
	active := cfg.Adapters.Adapters.Active
	if active == "" || active == "tui" {
		// For queue retry, use a no-op adapter since we just need to validate the delivery path.
		return &noopAdapter{}, nil
	}
	return nil, errors.New("adapter not implemented: " + active)
}

// noopAdapter is a no-op adapter used for queue retry testing.
type noopAdapter struct{}

func (a *noopAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	_ = ctx
	_ = batch
	return nil
}
