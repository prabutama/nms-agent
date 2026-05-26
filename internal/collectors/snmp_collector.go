package collectors

import (
	"context"
	"time"

	g "github.com/gosnmp/gosnmp"

	"nms-agent/internal/models"
	"nms-agent/internal/profiles"
)

// SNMPCollector collects a minimal set of SNMP metrics (Phase 5 MVP).
// It returns one RawSample per metric to keep the processor contract stable.
type SNMPCollector struct {
	Targets  []Target
	Profiles []profiles.Profile

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
	profs := c.Profiles
	if len(profs) == 0 {
		return nil, nil
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
		profile, ok := profiles.SelectProfile(profs, t.Vendor, t.Model)
		if !ok {
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

			for _, m := range profile.Metrics {
				if m.Type == "get" {
					pkt, err := cli.Get([]string{m.OID})
					if err != nil {
						continue
					}
					for _, v := range pkt.Variables {
						val, ok := pduToFloat(v)
						if !ok {
							continue
						}
						out = append(out, rawMetricWithTags(t.DeviceID, "snmp", now, m.Metric, val, m.Unit, nil))
					}
					continue
				}

				if m.Type == "walk" {
					_ = cli.Walk(m.OID, func(p g.SnmpPDU) error {
						val, ok := pduToFloat(p)
						if !ok {
							return nil
						}
						var tags map[string]string
						if m.Index {
							if idx, ok := oidIndexSuffix(p.Name); ok {
								tags = map[string]string{"ifIndex": idx}
							}
						}
						out = append(out, rawMetricWithTags(t.DeviceID, "snmp", now, m.Metric, val, m.Unit, tags))
						return nil
					})
				}
			}
		}()
	}
	return out, nil
}

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

func pduToFloat(pdu g.SnmpPDU) (float64, bool) {
	// Prefer TimeTicks conversion when available.
	if pdu.Type == g.TimeTicks {
		return timeTicksToSeconds(pdu)
	}
	bi := g.ToBigInt(pdu.Value)
	if bi == nil {
		return 0, false
	}
	return float64(bi.Int64()), true
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
