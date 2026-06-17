package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"nms-agent/internal/adapters"
	"nms-agent/internal/collectors"
	"nms-agent/internal/config"
	"nms-agent/internal/configwatch"
	"nms-agent/internal/core"
	"nms-agent/internal/logger"
	"nms-agent/internal/models"
	"nms-agent/internal/processors"
	"nms-agent/internal/profiles"
	"nms-agent/internal/queue"
	"nms-agent/internal/routes"
	"nms-agent/internal/status"
	"nms-agent/internal/viewer"
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

	logCfg := loaded.Root.Agent.Logging.WithDefaults()
	log, err := logger.New(logger.Config{Level: logCfg.Level, Format: logCfg.Format})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	log.Info("starting",
		"config", configAbs,
		"poll_interval", loaded.Root.Agent.PollInterval.String(),
		"devices", len(loaded.Devices),
		"adapter.active", loaded.Adapters.Adapters.Active,
		"output.timezone", loc.String(),
		"queue.db", loaded.Root.Paths.QueueDB,
		"collector.mode", *collectorMode,
	)

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

	// Wire retry backoff config if queue supports it.
	retryCfg := loaded.Root.Agent.Delivery.Retry.WithDefaults()
	if retryCfg.Enabled {
		q.SetRetryConfig(queue.RetryConfig{
			Enabled:     true,
			BaseBackoff: retryCfg.BaseBackoff,
			MaxBackoff:  retryCfg.MaxBackoff,
			MaxRetries:  retryCfg.MaxRetries,
		})
		log.Info("queue_retry_enabled", "base_backoff", retryCfg.BaseBackoff.String(), "max_backoff", retryCfg.MaxBackoff.String())
	}

	runtimeStatus := status.NewRuntime()
	statusFilePath := filepath.Join(filepath.Dir(queuePath), "status.json")
	cleanupInterval := loaded.Root.Agent.Delivery.Retry.RetentionDays
	cleanupCounter := 0

	buildPipeline := func(cfg config.Loaded, queuePath string, q queue.Queue, hub *viewer.Hub, log *logger.Logger) (*core.Pipeline, adapters.Adapter, error) {
		coll, err := buildCollector(*collectorMode, cfg)
		if err != nil {
			return nil, nil, err
		}
		profilesDir := cfg.ProfilesDir
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
		if hub != nil {
			hub.SetAdapter(cfg.Adapters.Adapters.Active)
			hub.SetActiveDevices(activeDeviceIDs(cfg.Devices))

			// Wire queue snapshot provider
			items, _ := q.(*queue.SQLiteQueue).Snapshot(context.Background(), 200)
			var telemetry []models.Telemetry
			activeIDs := activeDeviceIDSet(cfg.Devices)
			for _, item := range items {
				if _, ok := activeIDs[item.Telemetry.DeviceID]; ok {
					telemetry = append(telemetry, item.Telemetry)
				}
			}
			hub.SetSnapshot(telemetry)

			// Wire live observer
			if p, ok := ad.(interface {
				SetObserver(interface{ Update([]models.Telemetry) })
			}); ok {
				p.SetObserver(hub)
			}
		}
		p := core.NewPipeline(
			coll,
			&processors.PreprocessThresholdProcessor{Rules: cfg.Thresholds.Thresholds},
			q,
			ad,
			dc,
		)
		p.SetLogger(log)
		if hub != nil {
			p.SetObserver(hub)
		}
		_ = queuePath
		return p, ad, nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	viewHub := viewer.NewHub(loaded.Adapters.Adapters.Active)
	viewServer := &viewer.Server{SocketPath: "/run/nms-agent/view.sock", Hub: viewHub}
	if err := viewServer.Listen(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "viewer server error: %v\n", err)
	}

	p, adapter, err := buildPipeline(loaded, queuePath, q, viewHub, log)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runtimeStatus.Update(loaded.Adapters.Adapters.Active, loaded.Root.Agent.PollInterval.String(), len(loaded.Devices), true, "", time.Now(), 0, 0, 0)
	_ = runtimeStatus.WriteFile(statusFilePath)

	reloadCh := make(chan os.Signal, 1)
	if sigs := reloadSignals(); len(sigs) > 0 {
		signal.Notify(reloadCh, sigs...)
		defer signal.Stop(reloadCh)
	}

	var devicesWatcher *configwatch.DevicesWatcher
	var devicesChangedCh <-chan struct{}
	watcherPath := config.ResolvePath(filepath.Dir(configAbs), loaded.Root.Paths.DevicesDir)
	if watcherPath != "" {
		devicesWatcher = configwatch.NewDevicesWatcher(watcherPath, 500*time.Millisecond)
		if err := devicesWatcher.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not start devices watcher: %v\n", err)
		} else {
			devicesChangedCh = devicesWatcher.Changes()
		}
	}
	defer func() {
		if devicesWatcher != nil {
			devicesWatcher.Stop()
		}
	}()

	pollInterval := loaded.Root.Agent.PollInterval
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	reloadPipeline := func(newLoaded config.Loaded) bool {
		loc, err := config.LoadLocation(newLoaded.Root.Agent.Output.Timezone)
		if err != nil {
			log.Warn("reload_failed", "step", "timezone", "error", err.Error())
			return false
		}
		adapters.SetOutputLocation(loc)
		newP, newAdapter, err := buildPipeline(newLoaded, queuePath, q, viewHub, log)
		if err != nil {
			log.Warn("reload_failed", "step", "build_pipeline", "error", err.Error())
			return false
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
		newWatcherPath := config.ResolvePath(filepath.Dir(configAbs), newLoaded.Root.Paths.DevicesDir)
		if newWatcherPath != watcherPath {
			if devicesWatcher != nil {
				devicesWatcher.Stop()
				devicesWatcher = nil
				devicesChangedCh = nil
			}
			watcherPath = newWatcherPath
			if watcherPath != "" {
				devicesWatcher = configwatch.NewDevicesWatcher(watcherPath, 500*time.Millisecond)
				if err := devicesWatcher.Start(); err != nil {
					log.Warn("devices_watcher_restart_failed", "error", err.Error())
				} else {
					devicesChangedCh = devicesWatcher.Changes()
				}
			}
		}
		loaded = newLoaded
		return true
	}

	for {
		cycleStart := time.Now()
		cycleErr := p.RunOnce(ctx)
		if cycleErr != nil {
			log.Error("cycle_error", "error", cycleErr.Error())
		}
		st, _ := q.Stats(ctx)
		runtimeStatus.Update(
			loaded.Adapters.Adapters.Active,
			loaded.Root.Agent.PollInterval.String(),
			len(loaded.Devices),
			cycleErr == nil,
			func() string {
				if cycleErr != nil {
					return cycleErr.Error()
				}
				return ""
			}(),
			cycleStart,
			st.PendingCount,
			st.DeadLetterCount,
			st.MaxRetry,
		)
		_ = runtimeStatus.WriteFile(statusFilePath)

		cleanupCounter++
		if cleanupInterval > 0 && cleanupCounter >= 100 {
			cleanupCounter = 0
			if deleted, err := q.CleanupDeleted(ctx, time.Duration(cleanupInterval)*24*time.Hour); err != nil {
				log.Warn("queue_cleanup_error", "error", err.Error())
			} else if deleted > 0 {
				log.Info("queue_cleanup_done", "deleted", deleted, "retention_days", cleanupInterval)
			}
		}

		select {
		case <-ctx.Done():
			log.Info("shutting_down")
			return 0
		case <-reloadCh:
			newLoaded, err := config.LoadFromFile(configAbs)
			if err != nil {
				log.Warn("reload_failed", "step", "load", "error", err.Error())
				continue
			}
			if err := config.Validate(newLoaded); err != nil {
				log.Warn("reload_failed", "step", "validate", "error", err.Error())
				continue
			}
			if !reloadPipeline(newLoaded) {
				continue
			}
			log.Info("reload_completed", "devices", len(loaded.Devices), "adapter.active", loaded.Adapters.Adapters.Active)
			runtimeStatus.Update(loaded.Adapters.Adapters.Active, loaded.Root.Agent.PollInterval.String(), len(loaded.Devices), true, "", time.Now(), 0, 0, 0)
			_ = runtimeStatus.WriteFile(statusFilePath)
			continue
		case <-devicesChangedCh:
			newLoaded, err := config.LoadFromFile(configAbs)
			if err != nil {
				log.Warn("devices_watcher_reload_failed", "step", "load", "error", err.Error())
				continue
			}
			if err := config.Validate(newLoaded); err != nil {
				log.Warn("devices_watcher_reload_failed", "step", "validate", "error", err.Error())
				continue
			}
			if reloadPipeline(newLoaded) {
				log.Info("devices_watcher_reload_completed", "devices", len(loaded.Devices), "adapter.active", loaded.Adapters.Adapters.Active)
				runtimeStatus.Update(loaded.Adapters.Adapters.Active, loaded.Root.Agent.PollInterval.String(), len(loaded.Devices), true, "", time.Now(), 0, 0, 0)
				_ = runtimeStatus.WriteFile(statusFilePath)
			}
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
		colList = append(colList, routes.Collector{Targets: snmpTargets, Cache: routes.NewChangeCache()})
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

func activeDeviceIDs(devices []config.Device) []string {
	out := make([]string, 0, len(devices))
	for _, d := range devices {
		if d.ID == "" {
			continue
		}
		out = append(out, d.ID)
	}
	return out
}

func activeDeviceIDSet(devices []config.Device) map[string]struct{} {
	out := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		if d.ID == "" {
			continue
		}
		out[d.ID] = struct{}{}
	}
	return out
}

type combinedCollector struct {
	collectors []collectors.Collector
}

func (c combinedCollector) Collect(ctx context.Context) ([]models.RawSample, error) {
	var out []models.RawSample
	var errs []string
	for _, col := range c.collectors {
		batch, err := col.Collect(ctx)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		out = append(out, batch...)
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("collector errors: %s", strings.Join(errs, "; "))
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
