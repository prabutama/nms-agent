package collectors

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"nms-agent/internal/models"
)

// ICMPCollector collects reachability and basic latency using the system ping command.
// This keeps dependencies small and works in unprivileged environments.
// Output is emitted as multiple RawSamples (one per metric) to avoid contract changes.
type ICMPCollector struct {
	Targets []Target

	// Count controls how many echo requests are attempted per target.
	Count int
	// Timeout bounds one ping command execution per target.
	Timeout time.Duration
	// Concurrency bounds concurrent per-target pings.
	Concurrency int

	// Exec is injectable for tests.
	Exec func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (c ICMPCollector) Collect(ctx context.Context) ([]models.RawSample, error) {
	if len(c.Targets) == 0 {
		return nil, nil
	}
	count := c.Count
	if count <= 0 {
		count = 1
	}
	to := c.Timeout
	if to <= 0 {
		to = 2 * time.Second
	}
	execFn := c.Exec
	if execFn == nil {
		execFn = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			err := cmd.Run()
			return out.Bytes(), err
		}
	}

	now := time.Now().UTC()
	workers := c.Concurrency
	if workers <= 0 {
		workers = 8
	}
	if workers > len(c.Targets) {
		workers = len(c.Targets)
	}
	if workers <= 0 {
		workers = 1
	}

	targetCh := make(chan Target)
	resultCh := make(chan []models.RawSample, len(c.Targets))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range targetCh {
				if t.DeviceID == "" || t.Address == "" {
					continue
				}
				deviceSamples := make([]models.RawSample, 0, 4)
				dctx, cancel := context.WithTimeout(ctx, to)
				b, err := execFn(dctx, "ping", pingArgs(t.Address, count, to)...)
				cancel()

				text := string(b)
				reachable := 1.0
				if err != nil {
					reachable = 0.0
				}
				deviceSamples = append(deviceSamples, rawMetric(t.DeviceID, "icmp", now, "icmp.reachable", reachable, ""))

				if lats := parsePingLatenciesMS(text); len(lats) > 0 {
					avg, jitter := summarizeLatencies(lats)
					deviceSamples = append(deviceSamples, rawMetric(t.DeviceID, "icmp", now, "icmp.latency_ms", avg, "ms"))
					if c.Count > 1 {
						deviceSamples = append(deviceSamples, rawMetric(t.DeviceID, "icmp", now, "icmp.jitter_ms", jitter, "ms"))
					}
				}
				if loss, ok := parsePingLossPct(text); ok {
					deviceSamples = append(deviceSamples, rawMetric(t.DeviceID, "icmp", now, "icmp.packet_loss_pct", loss, "pct"))
				}
				resultCh <- deviceSamples
			}
		}()
	}
	go func() {
		for _, t := range c.Targets {
			targetCh <- t
		}
		close(targetCh)
		wg.Wait()
		close(resultCh)
	}()

	out := make([]models.RawSample, 0, len(c.Targets)*2)
	for batch := range resultCh {
		out = append(out, batch...)
	}
	return out, nil
}

func pingArgs(address string, count int, timeout time.Duration) []string {
	// Keep it simple and portable enough for our supported dev envs.
	// Windows: ping -n <count> -w <timeout_ms>
	// Linux/macOS: ping -c <count> -W <timeout_s>
	if runtime.GOOS == "windows" {
		ms := int(timeout.Milliseconds())
		if ms <= 0 {
			ms = 2000
		}
		return []string{"-n", strconv.Itoa(count), "-w", strconv.Itoa(ms), address}
	}
	sec := int(timeout.Seconds())
	if sec <= 0 {
		sec = 2
	}
	return []string{"-c", strconv.Itoa(count), "-W", strconv.Itoa(sec), address}
}

var (
	// Windows: "time=12ms" or "time<1ms". Linux: "time=12.3 ms".
	rePingTime = regexp.MustCompile(`(?i)time[=<]\s*([0-9]+(?:\.[0-9]+)?)\s*ms`)
	// Windows: "(0% loss)". Linux: "0% packet loss".
	reLossPct = regexp.MustCompile(`(?i)([0-9]+)%\s*(?:loss|packet\s+loss)`)
)

func parsePingLatenciesMS(out string) []float64 {
	ms := rePingTime.FindAllStringSubmatch(out, -1)
	if len(ms) == 0 {
		return nil
	}
	vals := make([]float64, 0, len(ms))
	for _, m := range ms {
		if len(m) != 2 {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64)
		if err != nil {
			continue
		}
		vals = append(vals, v)
	}
	return vals
}

func summarizeLatencies(vals []float64) (avg float64, jitter float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	min := vals[0]
	max := vals[0]
	sum := 0.0
	for _, v := range vals {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = sum / float64(len(vals))
	// Simple jitter proxy: peak-to-peak.
	jitter = max - min
	return avg, jitter
}

func parsePingLossPct(out string) (float64, bool) {
	m := reLossPct.FindStringSubmatch(out)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func rawMetric(deviceID, source string, ts time.Time, metric string, value float64, unit string) models.RawSample {
	fields := map[string]any{"metric": metric, "value_type": "number", "value_number": value}
	if unit != "" {
		fields["unit"] = unit
	}
	return models.RawSample{DeviceID: deviceID, Source: source, TS: ts, Fields: fields}
}
