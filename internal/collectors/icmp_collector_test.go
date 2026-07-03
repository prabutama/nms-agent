package collectors

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	if v, _ := samples[0].Fields["value_number"].(float64); v != 0.0 {
		t.Fatalf("expected reachable=0, got %v", v)
	}
}

func TestICMPCollector_CollectsTargetsConcurrently(t *testing.T) {
	var current int32
	var maxSeen int32
	var mu sync.Mutex
	started := make(chan struct{}, 8)
	release := make(chan struct{})

	c := ICMPCollector{
		Targets: []Target{
			{DeviceID: "d1", Address: "1.1.1.1"},
			{DeviceID: "d2", Address: "1.1.1.2"},
			{DeviceID: "d3", Address: "1.1.1.3"},
		},
		Count:       1,
		Timeout:     500 * time.Millisecond,
		Concurrency: 3,
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			_ = ctx
			_ = name
			_ = args
			cur := atomic.AddInt32(&current, 1)
			for {
				max := atomic.LoadInt32(&maxSeen)
				if cur <= max || atomic.CompareAndSwapInt32(&maxSeen, max, cur) {
					break
				}
			}
			started <- struct{}{}
			<-release
			atomic.AddInt32(&current, -1)
			mu.Lock()
			defer mu.Unlock()
			return []byte("Reply from 1.1.1.1: bytes=32 time=10ms TTL=57\nLost = 0 (0% loss)\n"), nil
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = c.Collect(context.Background())
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for worker %d", i+1)
		}
	}
	if atomic.LoadInt32(&maxSeen) < 2 {
		t.Fatalf("expected concurrent exec, maxSeen=%d", atomic.LoadInt32(&maxSeen))
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("collect did not finish")
	}
}
