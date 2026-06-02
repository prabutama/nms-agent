package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/config"
)

func runAdapter(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	switch args[0] {
	case "health":
		return runAdapterHealth(args[1:])
	default:
		usage()
		return 2
	}
}

func runAdapterHealth(args []string) int {
	fs := flag.NewFlagSet("adapter health", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
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

	loc, err := config.LoadLocation(loaded.Root.Agent.Output.Timezone)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	adapters.SetOutputLocation(loc)

	active := loaded.Adapters.Adapters.Active
	if active == "terminal" || active == "tui" {
		fmt.Fprintf(os.Stdout, "adapter=%s status=ok\n", active)
		return 0
	}

	ad, err := adapters.NewAdapter(active, loaded.Adapters.Adapters.Configs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stdout, "adapter=%s status=error\n", active)
		return 1
	}
	if c, ok := ad.(adapters.Closable); ok {
		defer func() { _ = c.Close() }()
	}

	hc, ok := ad.(adapters.HealthChecker)
	if !ok {
		fmt.Fprintf(os.Stdout, "adapter=%s status=error reason=health_check_not_supported\n", active)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := hc.HealthCheck(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stdout, "adapter=%s status=error reason=%q\n", active, err.Error())
		return 1
	}

	fmt.Fprintf(os.Stdout, "adapter=%s status=ok\n", active)
	return 0
}
