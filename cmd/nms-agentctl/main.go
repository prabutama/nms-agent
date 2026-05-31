package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		os.Exit(runValidate(os.Args[2:]))
	case "queue":
		os.Exit(runQueue(os.Args[2:]))
	case "threshold":
		os.Exit(runThreshold(os.Args[2:]))
	case "adapter":
		os.Exit(runAdapter(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  nms-agentctl validate --config configs/agent.yml")
	fmt.Fprintln(os.Stderr, "  nms-agentctl queue status --config configs/agent.yml")
	fmt.Fprintln(os.Stderr, "  nms-agentctl queue retry --config configs/agent.yml [--limit 100]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl adapter health --config configs/agent.yml")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold list --config configs/agent.yml")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold set --config configs/agent.yml --metric <name> --operator <op> [--warning <val>] [--critical <val>] [--tags k=v,k2=v2]")
}

func runQueue(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	switch args[0] {
	case "status":
		return runQueueStatus(args[1:])
	case "retry":
		return runQueueRetry(args[1:])
	default:
		usage()
		return 2
	}
}
