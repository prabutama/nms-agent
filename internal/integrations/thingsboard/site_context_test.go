package thingsboard

import (
	"testing"
	"time"

	"nms-agent/internal/routes"
)

func TestValidateSiteContext(t *testing.T) {
	if err := ValidateSiteContext(Config{}); err == nil {
		t.Fatalf("expected validation error")
	}
	cfg := Config{API: APIConfig{BaseURL: "https://example.com", APIKey: "key"}, Site: SiteConfig{Key: "branch-b", AssetID: "asset-1"}}
	if err := ValidateSiteContext(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildSiteTopology(t *testing.T) {
	site := SiteConfig{Key: "branch-b", AssetID: "asset-1"}
	snapshots := []routes.RouteSnapshot{{
		DeviceID:      "mikrotik-br-b-router",
		AddressFamily: "ipv4",
		Supported:     true,
		CollectedAt:   time.Unix(10, 0).UTC(),
		Routes: []routes.RouteEntry{
			{DeviceID: "mikrotik-br-b-router", Destination: "172.16.30.0/24", RouteType: "connected"},
			{DeviceID: "mikrotik-br-b-router", Destination: "0.0.0.0/0", NextHop: "10.10.10.1", IsDefault: true, RouteType: "remote"},
		},
	}}
	topo := BuildSiteTopology(site, snapshots)
	if topo.SiteKey != "branch-b" || topo.AssetID != "asset-1" {
		t.Fatalf("unexpected topology header %+v", topo)
	}
	if topo.DeviceCount != 1 || topo.EdgeCount == 0 || topo.Fingerprint == "" {
		t.Fatalf("unexpected topology summary %+v", topo)
	}
}

func TestBuildSiteTopology_ResolvesNextHopToManagedDevice(t *testing.T) {
	site := SiteConfig{Key: "branch-b", AssetID: "asset-1"}
	snapshots := []routes.RouteSnapshot{
		{
			DeviceID:      "linux-br-b-server",
			AddressFamily: "ipv4",
			Supported:     true,
			CollectedAt:   time.Unix(10, 0).UTC(),
			Routes: []routes.RouteEntry{
				{DeviceID: "linux-br-b-server", Destination: "172.16.30.0/24", RouteType: "connected"},
				{DeviceID: "linux-br-b-server", Destination: "0.0.0.0/0", NextHop: "172.16.30.1", IsDefault: true, RouteType: "remote"},
			},
		},
		{
			DeviceID:      "mikrotik-br-b-router",
			AddressFamily: "ipv4",
			Supported:     true,
			CollectedAt:   time.Unix(10, 0).UTC(),
			Routes:        []routes.RouteEntry{{DeviceID: "mikrotik-br-b-router", Destination: "172.16.30.0/24", RouteType: "connected"}},
		},
	}
	topo := BuildSiteTopology(site, snapshots)
	found := false
	for _, edge := range topo.Edges {
		if edge.From == "device:linux-br-b-server" && edge.To == "device:mikrotik-br-b-router" && edge.Reason == "next_hop_match" && edge.Resolved {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected managed next-hop edge, got %+v", topo.Edges)
	}
}

func TestBuildSiteTopology_DefaultRouteFallsBackToExternalNode(t *testing.T) {
	site := SiteConfig{Key: "branch-b", AssetID: "asset-1"}
	snapshots := []routes.RouteSnapshot{{
		DeviceID:      "linux-br-b-server",
		AddressFamily: "ipv4",
		Supported:     true,
		CollectedAt:   time.Unix(10, 0).UTC(),
		Routes: []routes.RouteEntry{
			{DeviceID: "linux-br-b-server", Destination: "172.16.30.0/24", RouteType: "connected"},
			{DeviceID: "linux-br-b-server", Destination: "0.0.0.0/0", NextHop: "10.10.10.1", IsDefault: true, RouteType: "remote"},
		},
	}}
	topo := BuildSiteTopology(site, snapshots)
	found := false
	for _, edge := range topo.Edges {
		if edge.From == "device:linux-br-b-server" && edge.To == "external:10.10.10.1" && edge.Reason == "default_route" && !edge.Resolved {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected external gateway edge, got %+v", topo.Edges)
	}
}
