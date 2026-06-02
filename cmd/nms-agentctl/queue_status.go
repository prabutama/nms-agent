package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"nms-agent/internal/config"
	"nms-agent/internal/queue"
)

func runQueueStatus(args []string) int {
	fs := flag.NewFlagSet("queue status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
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

	q, err := queue.OpenSQLite(cfg.Root.Paths.QueueDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	defer q.Close()

	st, err := q.Stats(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	// Stable, short output.
	fmt.Fprintf(os.Stdout, "queue_db=%s pending=%d max_retry=%d\n", cfg.Root.Paths.QueueDB, st.PendingCount, st.MaxRetry)
	return 0
}
