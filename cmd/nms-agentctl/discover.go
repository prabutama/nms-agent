package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	discoveryexplorer "nms-agent/internal/discovery/explorer"
	discoverynetlink "nms-agent/internal/discovery/providers/netlink"
)

var newDiscoveryService = func() discovery.Service {
	return discovery.Service{
		Provider: discoverynetlink.Provider{},
		Prober:   discovery.SNMPProber{},
		Explorer: discoveryexplorer.Explorer{},
	}
}

func runDiscovery(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	switch args[0] {
	case "run":
		return runDiscoveryRun(args[1:])
	case "preview":
		return runDiscoveryPreview(args[1:])
	case "status":
		return runDiscoveryStatus(args[1:])
	default:
		usage()
		return 2
	}
}

func runDiscoveryRun(args []string) int {
	return runDiscoveryCommand(args, true)
}

func runDiscoveryPreview(args []string) int {
	return runDiscoveryCommand(args, false)
}

func runDiscoveryCommand(args []string, apply bool) int {
	name := "discovery preview"
	if apply {
		name = "discovery run"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	configAbs, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	loaded, err := config.LoadFromFile(configAbs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(loaded); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	svc := newDiscoveryService()
	var res discovery.Result
	if apply {
		res, err = svc.RunOnce(context.Background(), configAbs, loaded)
	} else {
		res, err = svc.PreviewOnce(context.Background(), configAbs, loaded)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	printDiscoveryResult(res, apply)
	return 0
}

func runDiscoveryStatus(args []string) int {
	fs := flag.NewFlagSet("discovery status", flag.ContinueOnError)
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
	d := loaded.Root.Discovery
	fmt.Fprintf(os.Stdout, "enabled=%t\n", d.Enabled)
	if !d.Enabled {
		return 0
	}
	fmt.Fprintf(os.Stdout, "interval=%s\n", d.Interval)
	fmt.Fprintf(os.Stdout, "interface=%s\n", d.Interface)
	fmt.Fprintf(os.Stdout, "subnet=%s\n", d.Subnet)
	fmt.Fprintf(os.Stdout, "provider=%s\n", d.Provider)
	fmt.Fprintf(os.Stdout, "snmp.version=%s\n", d.SNMP.Version)
	fmt.Fprintf(os.Stdout, "snmp.timeout=%s\n", d.SNMP.Timeout)
	fmt.Fprintf(os.Stdout, "snmp.retries=%d\n", d.SNMP.Retries)
	fmt.Fprintf(os.Stdout, "snmp.concurrency=%d\n", d.SNMP.Concurrency)
	fmt.Fprintf(os.Stdout, "auto_promote.enabled=%t\n", d.AutoPromote.Enabled)
	fmt.Fprintf(os.Stdout, "auto_promote.require_profile_match=%t\n", d.AutoPromote.RequireProfileMatch)
	fmt.Fprintf(os.Stdout, "auto_promote.max_new_devices_per_cycle=%d\n", d.AutoPromote.MaxNewDevicesPerCycle)
	fmt.Fprintf(os.Stdout, "auto_promote.device_id_template=%s\n", d.AutoPromote.DeviceIDTemplate)
	fmt.Fprintf(os.Stdout, "exploration.enabled=%t\n", d.Exploration.Enabled)
	fmt.Fprintf(os.Stdout, "exploration.run_when=%s\n", d.Exploration.RunWhen)
	fmt.Fprintf(os.Stdout, "exploration.max_oids_per_device=%d\n", d.Exploration.MaxOIDsPerDevice)
	fmt.Fprintf(os.Stdout, "exploration.output_dir=%s\n", d.Exploration.OutputDir)
	return 0
}

func printDiscoveryResult(res discovery.Result, apply bool) {
	mode := "preview"
	if apply {
		mode = "run"
	}
	fmt.Fprintf(os.Stdout, "mode=%s\n", mode)
	fmt.Fprintf(os.Stdout, "candidates=%d\n", res.CandidatesFound)
	fmt.Fprintf(os.Stdout, "existing_skipped=%d\n", res.ExistingSkipped)
	fmt.Fprintf(os.Stdout, "snmp_ok=%d\n", res.SNMPOK)
	fmt.Fprintf(os.Stdout, "profile_matched=%d\n", res.ProfileMatched)
	fmt.Fprintf(os.Stdout, "generated_profiles=%d\n", res.GeneratedProfiles)
	fmt.Fprintf(os.Stdout, "promoted=%d\n", res.Promoted)
	fmt.Fprintf(os.Stdout, "changed=%t\n", res.Changed)
	if len(res.SkippedReasons) > 0 {
		fmt.Fprintln(os.Stdout, "skipped_reasons:")
		for _, reason := range res.SkippedReasons {
			fmt.Fprintf(os.Stdout, "- %s\n", reason)
		}
	}
}
