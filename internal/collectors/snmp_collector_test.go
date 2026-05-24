package collectors

import (
	"context"
	"errors"
	"testing"
	"time"

	g "github.com/gosnmp/gosnmp"
)

type fakeSNMPClient struct {
	connectErr error
	packet     *g.SnmpPacket
	getErr     error
	walkFn     func(root string, fn g.WalkFunc) error
}

func (f fakeSNMPClient) Connect() error                      { return f.connectErr }
func (f fakeSNMPClient) Close() error                        { return nil }
func (f fakeSNMPClient) Get([]string) (*g.SnmpPacket, error) { return f.packet, f.getErr }
func (f fakeSNMPClient) Walk(root string, fn g.WalkFunc) error {
	if f.walkFn == nil {
		return nil
	}
	return f.walkFn(root, fn)
}

func TestSNMPCollector_EmitsUptimeSeconds(t *testing.T) {
	c := SNMPCollector{
		Targets:   []Target{{DeviceID: "d1", Address: "127.0.0.1"}},
		Community: "public",
		Timeout:   200 * time.Millisecond,
		Retries:   1,
		NewClient: func(t Target, cfg snmpClientConfig) snmpClient {
			_ = t
			_ = cfg
			return fakeSNMPClient{packet: &g.SnmpPacket{Variables: []g.SnmpPDU{{
				Name:  oidSysUpTime0,
				Type:  g.TimeTicks,
				Value: uint32(12345), // 123.45 seconds
			}}}, walkFn: func(root string, fn g.WalkFunc) error {
				// One interface index=1 sample for each walked OID.
				switch root {
				case oidIfOperStatus:
					return fn(g.SnmpPDU{Name: root + ".1", Type: g.Integer, Value: int(1)})
				case oidIfHCInOctets:
					return fn(g.SnmpPDU{Name: root + ".1", Type: g.Counter64, Value: uint64(10)})
				case oidIfHCOutOctets:
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
