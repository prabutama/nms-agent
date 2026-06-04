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
	case "reload":
		os.Exit(runReload(os.Args[2:]))
	case "queue":
		os.Exit(runQueue(os.Args[2:]))
	case "device":
		os.Exit(runDevice(os.Args[2:]))
	case "threshold":
		os.Exit(runThreshold(os.Args[2:]))
	case "adapter":
		os.Exit(runAdapter(os.Args[2:]))
	case "view":
		os.Exit(runView(os.Args[2:]))
	case "discovery":
		os.Exit(runDiscovery(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  nms-agentctl validate")
	fmt.Fprintln(os.Stderr, "  nms-agentctl reload --pid <pid>")
	fmt.Fprintln(os.Stderr, "  nms-agentctl device list")
	fmt.Fprintln(os.Stderr, "  nms-agentctl device add --id <id> --address <host> --vendor <v> --model <m> [--snmp=true] [--icmp=true]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl device update --id <id> [--address <host>] [--vendor <v>] [--model <m>] [--snmp=true|false] [--icmp=true|false]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl device remove --id <id>")
	fmt.Fprintln(os.Stderr, "  nms-agentctl device test --id <id> [--snmp=true|false] [--icmp=true|false]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl queue status")
	fmt.Fprintln(os.Stderr, "  nms-agentctl queue retry [--limit 100]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl adapter health")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold list")
	fmt.Fprintln(os.Stderr, "  nms-agentctl threshold set --metric <name> --operator <op> [--warning <val>] [--critical <val>] [--tags k=v,k2=v2]")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery status")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery preview")
	fmt.Fprintln(os.Stderr, "  nms-agentctl discovery run")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Config default: /etc/nms-agent/agent.yml")
	fmt.Fprintln(os.Stderr, "Override with: --config <path>")
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
