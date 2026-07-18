package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"nms-agent/internal/config"
	"nms-agent/internal/status"
)

func runStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "/etc/nms-agent/agent.yml", "Path to agent.yml")
	watch := fs.Bool("watch", false, "Continuously watch status output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	absCfg, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	dbPath := config.ResolvePath(filepath.Dir(absCfg), cfg.Root.Paths.NMSAgentDB)
	statusPath := filepath.Join(filepath.Dir(dbPath), "status.json")

	for {
		st, err := status.ReadFile(statusPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			fmt.Fprintf(os.Stdout, "agent_status=unreachable file=%s\n", statusPath)
			if !*watch {
				return 1
			}
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Fprintf(os.Stdout, "adapter=%s\n", st.AdapterActive)
		fmt.Fprintf(os.Stdout, "poll_interval=%s\n", st.PollInterval)
		fmt.Fprintf(os.Stdout, "devices=%d\n", st.DeviceCount)
		fmt.Fprintf(os.Stdout, "uptime=%s\n", st.Uptime)
		fmt.Fprintf(os.Stdout, "last_cycle_ok=%t\n", st.LastCycleOK)
		if st.LastCycleErr != "" {
			fmt.Fprintf(os.Stdout, "last_cycle_error=%s\n", st.LastCycleErr)
		}
		fmt.Fprintf(os.Stdout, "last_cycle_start=%s\n", st.LastCycleStart.Format(time.RFC3339))
		fmt.Fprintf(os.Stdout, "queue_pending=%d\n", st.QueuePending)
		fmt.Fprintf(os.Stdout, "queue_dead_letter=%d\n", st.QueueDeadLetter)
		fmt.Fprintf(os.Stdout, "queue_max_retry=%d\n", st.QueueMaxRetry)
		fmt.Fprintf(os.Stdout, "status_file=%s\n", statusPath)

		if !*watch {
			return 0
		}
		time.Sleep(2 * time.Second)
	}
}
