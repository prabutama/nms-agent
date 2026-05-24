package collectors

import (
	"context"
	"time"

	g "github.com/gosnmp/gosnmp"

	"nms-agent/internal/models"
)

// SNMPCollector collects a minimal set of SNMP metrics (Phase 5 MVP).
// It returns one RawSample per metric to keep the processor contract stable.
type SNMPCollector struct {
	Targets []Target

	Community string
	Version   g.SnmpVersion
	Port      uint16
	Timeout   time.Duration
	Retries   int

	// NewClient is injectable for tests.
	NewClient func(t Target, cfg snmpClientConfig) snmpClient
}

type snmpClientConfig struct {
	Community string
	Version   g.SnmpVersion
	Port      uint16
	Timeout   time.Duration
	Retries   int
}

type snmpClient interface {
	Connect() error
	Close() error
	Get(oids []string) (*g.SnmpPacket, error)
	Walk(rootOid string, walkFn g.WalkFunc) error
}

func (c SNMPCollector) Collect(ctx context.Context) ([]models.RawSample, error) {
	if len(c.Targets) == 0 {
		return nil, nil
	}
	community := c.Community
	if community == "" {
		community = "public"
	}
	ver := c.Version
	if ver == 0 {
		ver = g.Version2c
	}
	port := c.Port
	if port == 0 {
		port = 161
	}
	to := c.Timeout
	if to <= 0 {
		to = 2 * time.Second
	}
	retries := c.Retries
	if retries <= 0 {
		retries = 1
	}
	newClient := c.NewClient
	if newClient == nil {
		newClient = defaultSNMPClient
	}

	deadlineTo := to
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 && d < deadlineTo {
			deadlineTo = d
		}
	}

	now := time.Now().UTC()
	out := make([]models.RawSample, 0, len(c.Targets))
	for _, t := range c.Targets {
		if t.DeviceID == "" || t.Address == "" {
			continue
		}
		cli := newClient(t, snmpClientConfig{
			Community: community,
			Version:   ver,
			Port:      port,
			Timeout:   deadlineTo,
			Retries:   retries,
		})
		func() {
			if err := cli.Connect(); err != nil {
				// Partial snapshot: skip device without failing the whole pass.
				return
			}
			defer func() { _ = cli.Close() }()

			// Uptime.
			pkt, err := cli.Get([]string{oidSysUpTime0})
			if err == nil {
				for _, v := range pkt.Variables {
					if v.Name != oidSysUpTime0 {
						continue
					}
					secs, ok := timeTicksToSeconds(v)
					if !ok {
						continue
					}
					out = append(out, rawMetric(t.DeviceID, "snmp", now, "snmp.uptime_seconds", secs, "s"))
				}
			}

			// Interface oper status.
			_ = cli.Walk(oidIfOperStatus, func(p g.SnmpPDU) error {
				idx, ok := oidIndexSuffix(p.Name)
				if !ok {
					return nil
				}
				v := float64(g.ToBigInt(p.Value).Int64())
				out = append(out, rawMetricWithTags(t.DeviceID, "snmp", now, "snmp.if.oper_status", v, "", map[string]string{"ifIndex": idx}))
				return nil
			})

			// Interface traffic counters (64-bit if available).
			_ = cli.Walk(oidIfHCInOctets, func(p g.SnmpPDU) error {
				idx, ok := oidIndexSuffix(p.Name)
				if !ok {
					return nil
				}
				v := float64(g.ToBigInt(p.Value).Int64())
				out = append(out, rawMetricWithTags(t.DeviceID, "snmp", now, "snmp.if.in_octets", v, "octets", map[string]string{"ifIndex": idx}))
				return nil
			})
			_ = cli.Walk(oidIfHCOutOctets, func(p g.SnmpPDU) error {
				idx, ok := oidIndexSuffix(p.Name)
				if !ok {
					return nil
				}
				v := float64(g.ToBigInt(p.Value).Int64())
				out = append(out, rawMetricWithTags(t.DeviceID, "snmp", now, "snmp.if.out_octets", v, "octets", map[string]string{"ifIndex": idx}))
				return nil
			})
		}()
	}
	return out, nil
}

const oidSysUpTime0 = "1.3.6.1.2.1.1.3.0"

const (
	oidIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
	oidIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"
	oidIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10"
)

func defaultSNMPClient(t Target, cfg snmpClientConfig) snmpClient {
	gs := &g.GoSNMP{
		Target:    t.Address,
		Port:      cfg.Port,
		Community: cfg.Community,
		Version:   cfg.Version,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}
	return &goSNMPClient{gs: gs}
}

type goSNMPClient struct{ gs *g.GoSNMP }

func (c *goSNMPClient) Connect() error { return c.gs.Connect() }
func (c *goSNMPClient) Close() error {
	if c.gs == nil || c.gs.Conn == nil {
		return nil
	}
	return c.gs.Conn.Close()
}
func (c *goSNMPClient) Get(oids []string) (*g.SnmpPacket, error) { return c.gs.Get(oids) }
func (c *goSNMPClient) Walk(rootOid string, walkFn g.WalkFunc) error {
	return c.gs.Walk(rootOid, walkFn)
}

func timeTicksToSeconds(pdu g.SnmpPDU) (float64, bool) {
	// gosnmp.ToBigInt handles multiple integer-like types.
	bi := g.ToBigInt(pdu.Value)
	if bi == nil {
		return 0, false
	}
	// hundredths of seconds -> seconds
	return float64(bi.Int64()) / 100.0, true
}

func oidIndexSuffix(oid string) (string, bool) {
	// Expect ...".<index>".
	last := -1
	for i := len(oid) - 1; i >= 0; i-- {
		if oid[i] == '.' {
			last = i
			break
		}
	}
	if last == -1 || last == len(oid)-1 {
		return "", false
	}
	idx := oid[last+1:]
	for i := 0; i < len(idx); i++ {
		if idx[i] < '0' || idx[i] > '9' {
			return "", false
		}
	}
	return idx, true
}

func rawMetricWithTags(deviceID, source string, ts time.Time, metric string, value float64, unit string, tags map[string]string) models.RawSample {
	fields := map[string]any{"metric": metric, "value": value}
	if unit != "" {
		fields["unit"] = unit
	}
	if len(tags) > 0 {
		fields["tags"] = tags
	}
	return models.RawSample{DeviceID: deviceID, Source: source, TS: ts, Fields: fields}
}
