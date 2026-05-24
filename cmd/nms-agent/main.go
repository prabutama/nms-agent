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
	"nms-agent/internal/models"
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
	collectorMode := fs.String("collector-mode", "auto", "Collector mode: auto|dummy|real")
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
	fmt.Fprintf(os.Stdout, "collector.mode: %s\n", *collectorMode)
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

	coll, err := buildCollector(*collectorMode, loaded)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	p := core.NewPipeline(
		coll,
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

func buildCollector(mode string, loaded config.Loaded) (collectors.Collector, error) {
	switch mode {
	case "auto", "dummy", "real":
		// ok
	default:
		return nil, fmt.Errorf("invalid --collector-mode %q (expected auto|dummy|real)", mode)
	}

	icmpTargets, snmpTargets := buildTargets(loaded)
	hasRealTargets := len(icmpTargets) > 0 || len(snmpTargets) > 0

	if mode == "dummy" || (mode == "auto" && !hasRealTargets) {
		return collectors.DummyCollector{DeviceID: firstDeviceID(loaded)}, nil
	}
	if mode == "real" && !hasRealTargets {
		return nil, fmt.Errorf("collector-mode=real but no real targets enabled (set devices.d/*.yml icmp.enabled or snmp.enabled)")
	}

	// real or auto-with-targets: combine enabled collectors.
	colList := make([]collectors.Collector, 0, 2)
	if len(icmpTargets) > 0 {
		colList = append(colList, collectors.ICMPCollector{Targets: icmpTargets, Count: 2})
	}
	if len(snmpTargets) > 0 {
		colList = append(colList, collectors.SNMPCollector{Targets: snmpTargets})
	}
	return combinedCollector{collectors: colList}, nil
}

func buildTargets(loaded config.Loaded) (icmp []collectors.Target, snmp []collectors.Target) {
	for _, d := range loaded.Devices {
		t := collectors.Target{DeviceID: d.ID, Address: d.Address}
		if d.ICMP.Enabled {
			icmp = append(icmp, t)
		}
		if d.SNMP.Enabled {
			snmp = append(snmp, t)
		}
	}
	return icmp, snmp
}

type combinedCollector struct {
	collectors []collectors.Collector
}

func (c combinedCollector) Collect(ctx context.Context) ([]models.RawSample, error) {
	var out []models.RawSample
	for _, col := range c.collectors {
		batch, err := col.Collect(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
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
