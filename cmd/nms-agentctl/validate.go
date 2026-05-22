package main

import (
	"flag"
	"fmt"
	"os"

	"nms-agent/internal/config"
)

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := config.ValidateFiles(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Fprintln(os.Stdout, "config ok")
	return 0
}
