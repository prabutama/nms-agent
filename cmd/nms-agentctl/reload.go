package main

import (
	"flag"
	"fmt"
	"os"

	"nms-agent/internal/config"
)

func runReload(args []string) int {
	fs := flag.NewFlagSet("reload", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	pid := fs.Int("pid", 0, "PID of running nms-agent process")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pid <= 0 {
		fmt.Fprintln(os.Stderr, "--pid is required and must be > 0")
		return 2
	}

	// Validate config first so we never trigger reload with an invalid config.
	if err := config.ValidateFiles(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	if err := sendReloadSignal(*pid); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Fprintf(os.Stdout, "reload signal sent pid=%d\n", *pid)
	return 0
}
