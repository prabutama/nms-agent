package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/collectors"
	"nms-agent/internal/config"
	"nms-agent/internal/core"
	"nms-agent/internal/models"
	"nms-agent/internal/processors"
	"nms-agent/internal/profiles"
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
	loc, err := config.LoadLocation(loaded.Root.Agent.Output.Timezone)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	adapters.SetOutputLocation(loc)

	fmt.Fprintf(os.Stdout, "nms-agent starting\n")
	fmt.Fprintf(os.Stdout, "config: %s\n", configAbs)
	fmt.Fprintf(os.Stdout, "poll_interval: %s\n", loaded.Root.Agent.PollInterval)
	fmt.Fprintf(os.Stdout, "devices: %d\n", len(loaded.Devices))
	fmt.Fprintf(os.Stdout, "adapter.active: %s\n", loaded.Adapters.Adapters.Active)
	fmt.Fprintf(os.Stdout, "output.timezone: %s\n", loc.String())
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

	buildPipeline := func(cfg config.Loaded, queuePath string, q queue.Queue) (*core.Pipeline, adapters.Adapter, error) {
		coll, err := buildCollector(*collectorMode, cfg)
		if err != nil {
			return nil, nil, err
		}
		profilesDir := filepath.Join(filepath.Dir(configAbs), "..", "profiles")
		profs, err := profiles.LoadDir(filepath.Clean(profilesDir))
		if err != nil {
			return nil, nil, err
		}
		if err := profiles.ValidateAll(profs); err != nil {
			return nil, nil, err
		}
		if sc, ok := coll.(collectors.SNMPCollector); ok {
			sc.Profiles = profs
			coll = sc
		} else if cc, ok := coll.(combinedCollector); ok {
			for i, c := range cc.collectors {
				if sc, ok := c.(collectors.SNMPCollector); ok {
					sc.Profiles = profs
					cc.collectors[i] = sc
				}
			}
			coll = cc
		}

		dc := core.DeliveryConfig{
			MaxBatch:           cfg.Root.Agent.Delivery.MaxBatch,
			DrainEnabled:       cfg.Root.Agent.Delivery.DrainEnabled,
			MaxBatchesPerCycle: cfg.Root.Agent.Delivery.MaxBatchesPerCycle,
			StopOnError:        cfg.Root.Agent.Delivery.StopOnError,
		}
		ad, err := adapters.NewAdapter(cfg.Adapters.Adapters.Active, cfg.Adapters.Adapters.Configs)
		if err != nil {
			return nil, nil, err
		}
		p := core.NewPipeline(
			coll,
			&processors.PreprocessThresholdProcessor{Rules: cfg.Thresholds.Thresholds},
			q,
			ad,
			dc,
		)
		_ = queuePath
		return p, ad, nil
	}

	p, adapter, err := buildPipeline(loaded, queuePath, q)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reloadCh := make(chan os.Signal, 1)
	if sigs := reloadSignals(); len(sigs) > 0 {
		signal.Notify(reloadCh, sigs...)
		defer signal.Stop(reloadCh)
	}

	pollInterval := loaded.Root.Agent.PollInterval
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := p.RunOnce(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		select {
		case <-ctx.Done():
			return 0
		case <-reloadCh:
			newLoaded, err := config.LoadFromFile(configAbs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
				continue
			}
			if err := config.Validate(newLoaded); err != nil {
				fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
				continue
			}
			loc, err := config.LoadLocation(newLoaded.Root.Agent.Output.Timezone)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
				continue
			}
			adapters.SetOutputLocation(loc)

			// Rebuild pipeline (queue remains opened).
			newP, newAdapter, err := buildPipeline(newLoaded, queuePath, q)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
				continue
			}
			if c, ok := adapter.(adapters.Closable); ok {
				_ = c.Close()
			}
			adapter = newAdapter
			p = newP
			if newLoaded.Root.Agent.PollInterval != pollInterval {
				pollInterval = newLoaded.Root.Agent.PollInterval
				ticker.Stop()
				ticker = time.NewTicker(pollInterval)
			}
			fmt.Fprintf(os.Stdout, "reloaded config: devices=%d adapter.active=%s\n", len(newLoaded.Devices), newLoaded.Adapters.Adapters.Active)
			continue
		case <-ticker.C:
		}
	}
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
		t := collectors.Target{DeviceID: d.ID, Address: d.Address, Vendor: d.Vendor, Model: d.Model}
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
