package routes

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	g "github.com/gosnmp/gosnmp"

	"nms-agent/internal/collectors"
	"nms-agent/internal/models"
)

type fakeSNMPClient struct {
	walks map[string][]g.SnmpPDU
}

func (f fakeSNMPClient) Connect() error { return nil }
func (f fakeSNMPClient) Close() error   { return nil }
func (f fakeSNMPClient) Walk(rootOid string, walkFn g.WalkFunc) error {
	for _, pdu := range f.walks[rootOid] {
		if err := walkFn(pdu); err != nil {
			return err
		}
	}
	return nil
}

func TestWalkIPCidrRoutesParsing(t *testing.T) {
	p := SNMPProvider{}
	cli := fakeSNMPClient{walks: map[string][]g.SnmpPDU{
		oidIPCidrRouteDest:  {{Name: oidIPCidrRouteDest + ".0.0.0.0", Value: "0.0.0.0"}},
		oidIPCidrRouteMask:  {{Name: oidIPCidrRouteMask + ".0.0.0.0", Value: "0.0.0.0"}},
		oidIPCidrNextHop:    {{Name: oidIPCidrNextHop + ".0.0.0.0", Value: "10.10.10.1"}},
		oidIPCidrIfIndex:    {{Name: oidIPCidrIfIndex + ".0.0.0.0", Value: 0}},
		oidIPCidrRouteType:  {{Name: oidIPCidrRouteType + ".0.0.0.0", Value: 4}},
		oidIPCidrRouteProto: {{Name: oidIPCidrRouteProto + ".0.0.0.0", Value: 3}},
		oidIPCidrMetric1:    {{Name: oidIPCidrMetric1 + ".0.0.0.0", Value: 1}},
	}}
	routes, err := p.walkIPCidrRoutes(context.Background(), cli)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].nextHop != "10.10.10.1" || routes[0].destIP != "0.0.0.0" {
		t.Fatalf("unexpected routes %+v", routes)
	}
}

func TestWalkLegacyIPRoutesParsing(t *testing.T) {
	p := SNMPProvider{}
	cli := fakeSNMPClient{walks: map[string][]g.SnmpPDU{
		oidIPRouteDest:    {{Name: oidIPRouteDest + ".172.16.30.0", Value: "172.16.30.0"}},
		oidIPRouteMask:    {{Name: oidIPRouteMask + ".172.16.30.0", Value: "255.255.255.0"}},
		oidIPRouteNextHop: {{Name: oidIPRouteNextHop + ".172.16.30.0", Value: "0.0.0.0"}},
		oidIPRouteIfIndex: {{Name: oidIPRouteIfIndex + ".172.16.30.0", Value: 2}},
	}}
	routes, err := p.walkLegacyIPRoutes(context.Background(), cli)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].ifIndex != "2" || routes[0].mask != "255.255.255.0" {
		t.Fatalf("unexpected routes %+v", routes)
	}
}

func TestResolveIfIndexToIfNameAndDefaultRouteResolution(t *testing.T) {
	routes := resolveRoutes("r1", SourceSNMPIPCidrRouteTable, []rawRouteRow{
		{destIP: "0.0.0.0", mask: "0.0.0.0", nextHop: "10.10.10.1", ifIndex: "0", routeType: 4, proto: 3},
		{destIP: "10.10.10.0", mask: "255.255.255.0", nextHop: "0.0.0.0", ifIndex: "8", routeType: 3, proto: 2},
		{destIP: "172.16.30.0", mask: "255.255.255.0", nextHop: "0.0.0.0", ifIndex: "9", routeType: 3, proto: 2},
	}, map[string]string{"8": "ether7", "9": "ether8"}, time.Unix(10, 0).UTC())
	def, ok := firstDefaultRoute(routes)
	if !ok {
		t.Fatalf("expected default route")
	}
	if def.InterfaceID != "8" || def.InterfaceName != "ether7" || def.InterfaceResolvedBy != "next_hop_connected_route" {
		t.Fatalf("unexpected resolved default route %+v", def)
	}
}

func TestFingerprintStableWhenOrderChanges(t *testing.T) {
	r1 := []RouteEntry{{Destination: "0.0.0.0/0", NextHop: "10.10.10.1", InterfaceID: "8", Metric: 1, Protocol: "netmgmt", RouteType: "remote"}, {Destination: "10.10.10.0/24", NextHop: "0.0.0.0", InterfaceID: "8", Metric: 0, Protocol: "local", RouteType: "connected"}}
	r2 := []RouteEntry{{Destination: "10.10.10.0/24", NextHop: "0.0.0.0", InterfaceID: "8", Metric: 0, Protocol: "local", RouteType: "connected"}, {Destination: "0.0.0.0/0", NextHop: "10.10.10.1", InterfaceID: "8", Metric: 1, Protocol: "netmgmt", RouteType: "remote"}}
	sortRouteEntries(r1)
	sortRouteEntries(r2)
	if Fingerprint(r1) != Fingerprint(r2) {
		t.Fatalf("expected stable fingerprint")
	}
}

func TestChangeFlagBehavior(t *testing.T) {
	cache := NewChangeCache()
	if cache.Update("d1", "ipv4", "abc") {
		t.Fatalf("first fingerprint must not be changed")
	}
	if cache.Update("d1", "ipv4", "abc") {
		t.Fatalf("same fingerprint must not be changed")
	}
	if !cache.Update("d1", "ipv4", "def") {
		t.Fatalf("changed fingerprint must set changed")
	}
}

func TestUnsupportedRouteTableDoesNotReturnFatalCollectionError(t *testing.T) {
	collector := Collector{Targets: []collectors.Target{{DeviceID: "d1", Address: "127.0.0.1"}}, Provider: stubProvider{snapshot: RouteSnapshot{DeviceID: "d1", AddressFamily: "ipv4", Supported: false, Source: SourceSNMPIPRouteTableLegacy, CollectedAt: time.Unix(10, 0).UTC()}}}
	raw, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatalf("expected summary raw samples")
	}
}

func TestRouteSnapshotJSONRespectsMaxSnapshotRoutes(t *testing.T) {
	routes := make([]RouteEntry, 0, maxSnapshotRoutes+10)
	for i := 0; i < maxSnapshotRoutes+10; i++ {
		routes = append(routes, RouteEntry{Destination: "10.0.0.0/24"})
	}
	snapshot := RouteSnapshot{DeviceID: "d1", AddressFamily: "ipv4", Source: SourceSNMPIPCidrRouteTable, Supported: true, Routes: routes}
	s, err := snapshotJSON(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var out RouteSnapshot
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Routes) != maxSnapshotRoutes {
		t.Fatalf("expected %d routes, got %d", maxSnapshotRoutes, len(out.Routes))
	}
}

func TestNormalizerEmitsCorrectRouteKeysAndValueTypes(t *testing.T) {
	snapshot := RouteSnapshot{
		DeviceID:            "d1",
		AddressFamily:       "ipv4",
		Source:              SourceSNMPIPCidrRouteTable,
		Supported:           true,
		CollectedAt:         time.Unix(10, 0).UTC(),
		RouteCount:          2,
		DefaultRouteCount:   1,
		ConnectedRouteCount: 1,
		RemoteRouteCount:    1,
		Changed:             true,
		Routes:              []RouteEntry{{Destination: "0.0.0.0/0", NextHop: "10.10.10.1", InterfaceID: "8", InterfaceName: "ether7", Protocol: "netmgmt", RouteType: "remote", IsDefault: true}},
	}
	raw, err := NormalizeSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assertMetric := func(metric, valueType string) {
		for _, sample := range raw {
			if sample.Fields["metric"] == metric {
				if sample.Fields["value_type"] != valueType {
					t.Fatalf("metric %s got value_type %v want %s", metric, sample.Fields["value_type"], valueType)
				}
				return
			}
		}
		t.Fatalf("metric %s not found", metric)
	}
	assertMetric("route.ipv4.route_count", "number")
	assertMetric("route.ipv4.snapshot", "string")
	assertMetric("route.ipv4.default.next_hop", "string")
}

func TestSummarizeSnapshotDoesNotDoubleCountExistingValues(t *testing.T) {
	snapshot := RouteSnapshot{
		RouteCount:          99,
		DefaultRouteCount:   99,
		ConnectedRouteCount: 99,
		RemoteRouteCount:    99,
		Routes: []RouteEntry{
			{Destination: "0.0.0.0/0", IsDefault: true, RouteType: "remote"},
			{Destination: "172.16.20.0/24", RouteType: "connected"},
		},
	}
	snapshot = summarizeSnapshot(snapshot)
	if snapshot.RouteCount != 2 || snapshot.DefaultRouteCount != 1 || snapshot.ConnectedRouteCount != 1 || snapshot.RemoteRouteCount != 1 {
		t.Fatalf("unexpected counts %+v", snapshot)
	}
}

type stubProvider struct{ snapshot RouteSnapshot }

func (s stubProvider) Collect(ctx context.Context, deviceID, address string) (RouteSnapshot, error) {
	_ = ctx
	_ = address
	s.snapshot.DeviceID = deviceID
	if s.snapshot.CollectedAt.IsZero() {
		s.snapshot.CollectedAt = time.Unix(10, 0).UTC()
	}
	return s.snapshot, nil
}

var _ = models.RawSample{}
