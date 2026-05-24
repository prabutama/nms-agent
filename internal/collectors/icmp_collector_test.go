package collectors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestICMPCollector_ParsesLatencyLossAndJitter(t *testing.T) {
	out := "Reply from 1.1.1.1: bytes=32 time=10ms TTL=57\n" +
		"Reply from 1.1.1.1: bytes=32 time=20ms TTL=57\n" +
		"Lost = 0 (0% loss)\n"

	c := ICMPCollector{
		Targets: []Target{{DeviceID: "d1", Address: "1.1.1.1"}},
		Count:   2,
		Timeout: 500 * time.Millisecond,
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(out), nil
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Expect: reachable + latency + jitter + loss
	if len(samples) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(samples))
	}
}

func TestICMPCollector_PartialSnapshotOnErrorStillEmitsReachable0(t *testing.T) {
	c := ICMPCollector{
		Targets: []Target{{DeviceID: "d1", Address: "1.1.1.1"}},
		Count:   1,
		Timeout: 10 * time.Millisecond,
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("timeout"), errors.New("exit 1")
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(samples) < 1 {
		t.Fatalf("expected at least 1 sample")
	}
	if m, _ := samples[0].Fields["metric"].(string); m != "icmp.reachable" {
		t.Fatalf("expected first metric icmp.reachable, got %q", m)
	}
	if v, _ := samples[0].Fields["value"].(float64); v != 0.0 {
		t.Fatalf("expected reachable=0, got %v", v)
	}
}
