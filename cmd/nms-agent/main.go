package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"nms-agent/internal/adapters"
	"nms-agent/internal/collectors"
	"nms-agent/internal/config"
	"nms-agent/internal/core"
	"nms-agent/internal/processors"
	"nms-agent/internal/queue"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(run(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  nms-agent run --config configs/agent.yml")
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	loaded, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Fprintf(os.Stdout, "nms-agent starting\n")
	fmt.Fprintf(os.Stdout, "config: %s\n", *configPath)
	fmt.Fprintf(os.Stdout, "poll_interval: %s\n", loaded.Root.Agent.PollInterval)
	fmt.Fprintf(os.Stdout, "devices: %d\n", len(loaded.Devices))
	fmt.Fprintf(os.Stdout, "adapter.active: %s\n", loaded.Adapters.Adapters.Active)
	fmt.Fprintf(os.Stdout, "queue.db: %s\n", loaded.Root.Paths.QueueDB)
	fmt.Fprintln(os.Stdout)

	queuePath := loaded.Root.Paths.QueueDB
	queueDir := filepath.Dir(queuePath)
	if queueDir != "." && queueDir != "" {
		if err := os.MkdirAll(queueDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "create queue dir %s: %v\n", queueDir, err)
			return 1
		}
	}
	q, err := queue.OpenSQLite(queuePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	defer q.Close()

	// Phase 3: run a single demo pass through the pipeline using dummy components.
	p := core.NewPipeline(
		collectors.DummyCollector{DeviceID: firstDeviceID(loaded)},
		processors.PassthroughProcessor{},
		q,
		adapters.NewTerminalAdapter(),
	)
	if err := p.RunOnce(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func firstDeviceID(cfg config.Loaded) string {
	if len(cfg.Devices) == 0 {
		return "dummy-1"
	}
	if cfg.Devices[0].ID == "" {
		return "dummy-1"
	}
	return cfg.Devices[0].ID
}
