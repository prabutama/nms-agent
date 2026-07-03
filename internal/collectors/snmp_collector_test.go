package collectors

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	g "github.com/gosnmp/gosnmp"

	"nms-agent/internal/profiles"
)

type fakeSNMPClient struct {
	connectErr error
	packet     *g.SnmpPacket
	getErr     error
	getFn      func(oids []string) (*g.SnmpPacket, error)
	walkFn     func(root string, fn g.WalkFunc) error
}

func (f fakeSNMPClient) Connect() error { return f.connectErr }
func (f fakeSNMPClient) Close() error   { return nil }
func (f fakeSNMPClient) Get(oids []string) (*g.SnmpPacket, error) {
	if f.getFn != nil {
		return f.getFn(oids)
	}
	return f.packet, f.getErr
}
func (f fakeSNMPClient) Walk(root string, fn g.WalkFunc) error {
	if f.walkFn == nil {
		return nil
	}
	return f.walkFn(root, fn)
}

func TestSNMPCollector_EmitsUptimeSeconds(t *testing.T) {
	c := SNMPCollector{
		Targets: []Target{{DeviceID: "d1", Address: "127.0.0.1", Vendor: "example", Model: "router"}},
		Profiles: []profiles.Profile{{
			Name:  "standard",
			Match: profiles.Match{},
			Metrics: []profiles.Metric{{Metric: "snmp.uptime_seconds", OID: "1.3.6.1.2.1.1.3.0", Type: "get", Unit: "s"},
				{Metric: "snmp.if.oper_status", OID: "1.3.6.1.2.1.2.2.1.8", Type: "walk", Index: true},
				{Metric: "snmp.if.in_octets", OID: "1.3.6.1.2.1.31.1.1.1.6", Type: "walk", Unit: "octets", Index: true},
				{Metric: "snmp.if.out_octets", OID: "1.3.6.1.2.1.31.1.1.1.10", Type: "walk", Unit: "octets", Index: true}},
		}},
		Community: "public",
		Timeout:   200 * time.Millisecond,
		Retries:   1,
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{packet: &g.SnmpPacket{Variables: []g.SnmpPDU{{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  g.TimeTicks,
				Value: uint32(12345), // 123.45 seconds
			}}}, walkFn: func(root string, fn g.WalkFunc) error {
				// One interface index=1 sample for each walked OID.
				switch root {
				case "1.3.6.1.2.1.2.2.1.8":
					return fn(g.SnmpPDU{Name: root + ".1", Type: g.Integer, Value: int(1)})
				case "1.3.6.1.2.1.31.1.1.1.6":
					return fn(g.SnmpPDU{Name: root + ".1", Type: g.Counter64, Value: uint64(10)})
				case "1.3.6.1.2.1.31.1.1.1.10":
					return fn(g.SnmpPDU{Name: root + ".1", Type: g.Counter64, Value: uint64(20)})
				default:
					return nil
				}
			}}
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(samples) != 4 {
		t.Fatalf("expected 4 samples (uptime+3 if metrics), got %d", len(samples))
	}
	if m, _ := samples[0].Fields["metric"].(string); m != "snmp.uptime_seconds" {
		t.Fatalf("metric=%q", m)
	}
	if vt, _ := samples[0].Fields["value_type"].(string); vt != "number" {
		t.Fatalf("value_type=%q", vt)
	}
}

func TestSNMPCollector_EmitsStringMetric(t *testing.T) {
	c := SNMPCollector{
		Targets: []Target{{DeviceID: "d1", Address: "127.0.0.1", Vendor: "mikrotik", Model: "routeros"}},
		Profiles: []profiles.Profile{{
			Name:    "mikrotik-routeros",
			Match:   profiles.Match{Vendor: "mikrotik", Model: "routeros"},
			Metrics: []profiles.Metric{{Metric: "snmp.system.description", OID: "1.3.6.1.2.1.1.1.0", Type: "get"}},
		}},
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{packet: &g.SnmpPacket{Variables: []g.SnmpPDU{{
				Name:  "1.3.6.1.2.1.1.1.0",
				Type:  g.OctetString,
				Value: []byte("RouterOS CHR"),
			}}}}
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(samples))
	}
	if vt, _ := samples[0].Fields["value_type"].(string); vt != "string" {
		t.Fatalf("value_type=%q", vt)
	}
	if vs, _ := samples[0].Fields["value_string"].(string); vs != "RouterOS CHR" {
		t.Fatalf("value_string=%q", vs)
	}
}

func TestSNMPCollector_PartialSnapshotSkipsDeviceOnConnectError(t *testing.T) {
	c := SNMPCollector{
		Targets: []Target{{DeviceID: "d1", Address: "127.0.0.1"}},
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{connectErr: errors.New("no route")}
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(samples) != 0 {
		t.Fatalf("expected 0 samples, got %d", len(samples))
	}
}

func TestSNMPCollector_UsesLoadedProfileMetrics(t *testing.T) {
	tmp := t.TempDir()
	profileYml := "name: standard\n" +
		"match:\n  vendor: \"\"\n  model: \"\"\n" +
		"metrics:\n" +
		"  - metric: snmp.uptime_seconds\n    oid: 1.3.6.1.2.1.1.3.0\n    type: get\n    unit: s\n" +
		"  - metric: snmp.if.oper_status\n    oid: 1.3.6.1.2.1.2.2.1.8\n    type: walk\n    index: true\n"
	if err := os.WriteFile(filepath.Join(tmp, "standard.yml"), []byte(profileYml), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := profiles.LoadDir(tmp)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if err := profiles.ValidateAll(loaded); err != nil {
		t.Fatalf("ValidateAll: %v", err)
	}

	var gotGet []string
	var gotWalk []string
	c := SNMPCollector{
		Targets:  []Target{{DeviceID: "d1", Address: "127.0.0.1", Vendor: "example", Model: "router"}},
		Profiles: loaded,
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{
				getFn: func(oids []string) (*g.SnmpPacket, error) {
					gotGet = append(gotGet, oids...)
					return &g.SnmpPacket{Variables: []g.SnmpPDU{{
						Name:  "1.3.6.1.2.1.1.3.0",
						Type:  g.TimeTicks,
						Value: uint32(12345),
					}}}, nil
				},
				walkFn: func(root string, fn g.WalkFunc) error {
					gotWalk = append(gotWalk, root)
					return fn(g.SnmpPDU{Name: root + ".7", Type: g.Integer, Value: int(1)})
				},
			}
		},
	}

	samples, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}

	if len(gotGet) != 1 || gotGet[0] != "1.3.6.1.2.1.1.3.0" {
		t.Fatalf("expected Get OID 1.3.6.1.2.1.1.3.0, got %v", gotGet)
	}
	if len(gotWalk) != 1 || gotWalk[0] != "1.3.6.1.2.1.2.2.1.8" {
		t.Fatalf("expected Walk OID 1.3.6.1.2.1.2.2.1.8, got %v", gotWalk)
	}

	metrics := map[string]map[string]string{}
	for _, s := range samples {
		metric, _ := s.Fields["metric"].(string)
		tags, _ := s.Fields["tags"].(map[string]string)
		metrics[metric] = tags
	}
	if _, ok := metrics["snmp.uptime_seconds"]; !ok {
		t.Fatalf("missing snmp.uptime_seconds")
	}
	if tags, ok := metrics["snmp.if.oper_status"]; !ok {
		t.Fatalf("missing snmp.if.oper_status")
	} else if tags["ifIndex"] != "7" {
		t.Fatalf("expected ifIndex=7, got %v", tags)
	}
}

func TestSNMPCollector_CollectsTargetsConcurrently(t *testing.T) {
	var current int32
	var maxSeen int32
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	c := SNMPCollector{
		Targets: []Target{
			{DeviceID: "d1", Address: "127.0.0.1", Vendor: "example", Model: "router"},
			{DeviceID: "d2", Address: "127.0.0.2", Vendor: "example", Model: "router"},
			{DeviceID: "d3", Address: "127.0.0.3", Vendor: "example", Model: "router"},
		},
		Profiles: []profiles.Profile{{
			Name:    "standard",
			Match:   profiles.Match{},
			Metrics: []profiles.Metric{{Metric: "snmp.system.description", OID: "1.3.6.1.2.1.1.1.0", Type: "get"}},
		}},
		Concurrency: 3,
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{
				getFn: func(oids []string) (*g.SnmpPacket, error) {
					_ = oids
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
					return &g.SnmpPacket{Variables: []g.SnmpPDU{{
						Name:  "1.3.6.1.2.1.1.1.0",
						Type:  g.OctetString,
						Value: []byte("RouterOS CHR"),
					}}}, nil
				},
			}
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
		t.Fatalf("expected concurrent SNMP get, maxSeen=%d", atomic.LoadInt32(&maxSeen))
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("collect did not finish")
	}
}
