package routes

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
)

const (
	oidIfName           = ".1.3.6.1.2.1.31.1.1.1.1"
	oidIfDescr          = ".1.3.6.1.2.1.2.2.1.2"
	oidIPCidrRouteDest  = ".1.3.6.1.2.1.4.24.4.1.1"
	oidIPCidrRouteMask  = ".1.3.6.1.2.1.4.24.4.1.2"
	oidIPCidrNextHop    = ".1.3.6.1.2.1.4.24.4.1.4"
	oidIPCidrIfIndex    = ".1.3.6.1.2.1.4.24.4.1.5"
	oidIPCidrRouteType  = ".1.3.6.1.2.1.4.24.4.1.6"
	oidIPCidrRouteProto = ".1.3.6.1.2.1.4.24.4.1.7"
	oidIPCidrMetric1    = ".1.3.6.1.2.1.4.24.4.1.11"
	oidIPRouteDest      = ".1.3.6.1.2.1.4.21.1.1"
	oidIPRouteIfIndex   = ".1.3.6.1.2.1.4.21.1.2"
	oidIPRouteMetric1   = ".1.3.6.1.2.1.4.21.1.3"
	oidIPRouteNextHop   = ".1.3.6.1.2.1.4.21.1.7"
	oidIPRouteType      = ".1.3.6.1.2.1.4.21.1.8"
	oidIPRouteProto     = ".1.3.6.1.2.1.4.21.1.9"
	oidIPRouteMask      = ".1.3.6.1.2.1.4.21.1.11"
)

type snmpClient interface {
	Connect() error
	Close() error
	Walk(rootOid string, walkFn g.WalkFunc) error
}

type SNMPProvider struct {
	Community string
	Version   g.SnmpVersion
	Port      uint16
	Timeout   time.Duration
	Retries   int
	NewClient func(address string, cfg snmpClientConfig) snmpClient
}

type snmpClientConfig struct {
	Community string
	Version   g.SnmpVersion
	Port      uint16
	Timeout   time.Duration
	Retries   int
}

func (p SNMPProvider) Collect(ctx context.Context, deviceID, address string) (RouteSnapshot, error) {
	collectedAt := time.Now().UTC()
	base := RouteSnapshot{DeviceID: deviceID, AddressFamily: "ipv4", CollectedAt: collectedAt}
	cli, err := p.connectClient(address)
	if err != nil {
		return base, nil
	}
	defer func() { _ = cli.Close() }()
	ifNames := p.walkInterfaceNames(cli)
	if rows, err := p.walkIPCidrRoutes(ctx, cli); err == nil && len(rows) > 0 {
		base.Source = SourceSNMPIPCidrRouteTable
		base.Supported = true
		base.Routes = resolveRoutes(deviceID, base.Source, rows, ifNames, collectedAt)
		return summarizeSnapshot(base), nil
	}
	if rows, err := p.walkInetCidrRoutes(ctx, cli); err == nil && len(rows) > 0 {
		base.Source = SourceSNMPInetCidrRouteTable
		base.Supported = true
		base.Routes = resolveRoutes(deviceID, base.Source, rows, ifNames, collectedAt)
		return summarizeSnapshot(base), nil
	}
	if rows, err := p.walkLegacyIPRoutes(ctx, cli); err == nil && len(rows) > 0 {
		base.Source = SourceSNMPIPRouteTableLegacy
		base.Supported = true
		base.Routes = resolveRoutes(deviceID, base.Source, rows, ifNames, collectedAt)
		return summarizeSnapshot(base), nil
	}
	base.Source = SourceSNMPIPRouteTableLegacy
	base.Supported = false
	return summarizeSnapshot(base), nil
}

func (p SNMPProvider) connectClient(address string) (snmpClient, error) {
	community := p.Community
	if community == "" {
		community = "public"
	}
	version := p.Version
	if version == 0 {
		version = g.Version2c
	}
	port := p.Port
	if port == 0 {
		port = 161
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	retries := p.Retries
	if retries <= 0 {
		retries = 1
	}
	newClient := p.NewClient
	if newClient == nil {
		newClient = defaultSNMPClient
	}
	cli := newClient(address, snmpClientConfig{Community: community, Version: version, Port: port, Timeout: timeout, Retries: retries})
	if err := cli.Connect(); err != nil {
		return nil, err
	}
	return cli, nil
}

func defaultSNMPClient(address string, cfg snmpClientConfig) snmpClient {
	gs := &g.GoSNMP{Target: address, Port: cfg.Port, Community: cfg.Community, Version: cfg.Version, Timeout: cfg.Timeout, Retries: cfg.Retries}
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
func (c *goSNMPClient) Walk(rootOid string, walkFn g.WalkFunc) error {
	return c.gs.Walk(rootOid, walkFn)
}

func (p SNMPProvider) walkInterfaceNames(cli snmpClient) map[string]string {
	ifNames := walkStringColumn(cli, oidIfName)
	if len(ifNames) > 0 {
		return ifNames
	}
	return walkStringColumn(cli, oidIfDescr)
}

func (p SNMPProvider) walkIPCidrRoutes(ctx context.Context, cli snmpClient) ([]rawRouteRow, error) {
	_ = ctx
	rows := map[string]*rawRouteRow{}
	if err := walkRouteStringColumn(cli, oidIPCidrRouteDest, rows, func(row *rawRouteRow, v string) { row.destIP = v }); err != nil {
		return nil, err
	}
	if err := walkRouteStringColumn(cli, oidIPCidrRouteMask, rows, func(row *rawRouteRow, v string) { row.mask = v }); err != nil {
		return nil, err
	}
	if err := walkRouteStringColumn(cli, oidIPCidrNextHop, rows, func(row *rawRouteRow, v string) { row.nextHop = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPCidrIfIndex, rows, func(row *rawRouteRow, v int) { row.ifIndex = strconv.Itoa(v) }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPCidrRouteType, rows, func(row *rawRouteRow, v int) { row.routeType = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPCidrRouteProto, rows, func(row *rawRouteRow, v int) { row.proto = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPCidrMetric1, rows, func(row *rawRouteRow, v int) { row.metric1 = v }); err != nil {
		return nil, err
	}
	return finalizeRouteRows(rows), nil
}

func (p SNMPProvider) walkInetCidrRoutes(ctx context.Context, cli snmpClient) ([]rawRouteRow, error) {
	_ = ctx
	_ = cli
	return nil, nil
}

func (p SNMPProvider) walkLegacyIPRoutes(ctx context.Context, cli snmpClient) ([]rawRouteRow, error) {
	_ = ctx
	rows := map[string]*rawRouteRow{}
	if err := walkRouteStringColumn(cli, oidIPRouteDest, rows, func(row *rawRouteRow, v string) { row.destIP = v }); err != nil {
		return nil, err
	}
	if err := walkRouteStringColumn(cli, oidIPRouteMask, rows, func(row *rawRouteRow, v string) { row.mask = v }); err != nil {
		return nil, err
	}
	if err := walkRouteStringColumn(cli, oidIPRouteNextHop, rows, func(row *rawRouteRow, v string) { row.nextHop = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPRouteIfIndex, rows, func(row *rawRouteRow, v int) { row.ifIndex = strconv.Itoa(v) }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPRouteType, rows, func(row *rawRouteRow, v int) { row.routeType = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPRouteProto, rows, func(row *rawRouteRow, v int) { row.proto = v }); err != nil {
		return nil, err
	}
	if err := walkRouteIntColumn(cli, oidIPRouteMetric1, rows, func(row *rawRouteRow, v int) { row.metric1 = v }); err != nil {
		return nil, err
	}
	return finalizeRouteRows(rows), nil
}

func walkStringColumn(cli snmpClient, oid string) map[string]string {
	out := map[string]string{}
	_ = cli.Walk(oid, func(pdu g.SnmpPDU) error {
		idx := oidSuffix(oid, pdu.Name)
		if idx == "" {
			return nil
		}
		if s, ok := pduString(pdu); ok {
			out[idx] = s
		}
		return nil
	})
	return out
}

func walkRouteStringColumn(cli snmpClient, oid string, rows map[string]*rawRouteRow, set func(*rawRouteRow, string)) error {
	return cli.Walk(oid, func(pdu g.SnmpPDU) error {
		key := oidSuffix(oid, pdu.Name)
		if key == "" {
			return nil
		}
		row := ensureRouteRow(rows, key)
		if s, ok := pduString(pdu); ok {
			set(row, s)
		}
		return nil
	})
}

func walkRouteIntColumn(cli snmpClient, oid string, rows map[string]*rawRouteRow, set func(*rawRouteRow, int)) error {
	return cli.Walk(oid, func(pdu g.SnmpPDU) error {
		key := oidSuffix(oid, pdu.Name)
		if key == "" {
			return nil
		}
		row := ensureRouteRow(rows, key)
		if v, ok := pduInt(pdu); ok {
			set(row, v)
		}
		return nil
	})
}

func ensureRouteRow(rows map[string]*rawRouteRow, key string) *rawRouteRow {
	row := rows[key]
	if row == nil {
		row = &rawRouteRow{key: key}
		rows[key] = row
	}
	return row
}

func finalizeRouteRows(rows map[string]*rawRouteRow) []rawRouteRow {
	out := make([]rawRouteRow, 0, len(rows))
	for _, row := range rows {
		if row.destIP == "" {
			continue
		}
		if row.mask == "" {
			row.mask = "255.255.255.255"
		}
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

func oidSuffix(base, name string) string {
	base = strings.TrimPrefix(base, ".")
	name = strings.TrimPrefix(name, ".")
	if !strings.HasPrefix(name, base) {
		return ""
	}
	suffix := strings.TrimPrefix(name[len(base):], ".")
	return suffix
}

func pduString(pdu g.SnmpPDU) (string, bool) {
	switch v := pdu.Value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case []byte:
		return strings.TrimSpace(string(v)), true
	default:
		return strings.TrimSpace(fmt.Sprint(pdu.Value)), pdu.Value != nil
	}
}

func pduInt(pdu g.SnmpPDU) (int, bool) {
	bi := g.ToBigInt(pdu.Value)
	if bi == nil {
		return 0, false
	}
	return int(bi.Int64()), true
}

func summarizeSnapshot(snapshot RouteSnapshot) RouteSnapshot {
	snapshot.RouteCount = 0
	snapshot.DefaultRouteCount = 0
	snapshot.ConnectedRouteCount = 0
	snapshot.RemoteRouteCount = 0
	snapshot.RouteCount = len(snapshot.Routes)
	for _, route := range snapshot.Routes {
		if route.IsDefault {
			snapshot.DefaultRouteCount++
		}
		if route.RouteType == "connected" {
			snapshot.ConnectedRouteCount++
		}
		if route.RouteType == "remote" {
			snapshot.RemoteRouteCount++
		}
	}
	return snapshot
}
