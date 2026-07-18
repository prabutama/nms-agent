package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	dbPath := config.ResolvePath(filepath.Dir(absCfg), cfg.Root.Paths.NMSAgentDB)
	q, err := queue.OpenSQLite(dbPath)
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
	fmt.Fprintf(os.Stdout, "nms_agent_db=%s pending=%d dead_letter=%d max_retry=%d\n", dbPath, st.PendingCount, st.DeadLetterCount, st.MaxRetry)
	return 0
}
