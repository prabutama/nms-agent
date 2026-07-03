package thingsboard

import (
	"context"
	"testing"

	"nms-agent/internal/routes"
)

type fakeTopologyClient struct {
	attrs          map[string]any
	getCalls       int
	saveCalls      int
	lastSavedAsset string
	lastSavedAttrs map[string]any
}

func (f *fakeTopologyClient) SaveAssetServerAttributes(_ context.Context, assetID string, attrs map[string]any) error {
	f.saveCalls++
	f.lastSavedAsset = assetID
	f.lastSavedAttrs = attrs
	return nil
}

func (f *fakeTopologyClient) GetAssetServerAttributes(_ context.Context, assetID string, keys []string) (map[string]any, error) {
	f.getCalls++
	out := make(map[string]any, len(f.attrs))
	for k, v := range f.attrs {
		out[k] = v
	}
	return out, nil
}

func TestTopologyPublisher_FirstRunSkipsWhenRemoteFingerprintMatches(t *testing.T) {
	site := SiteConfig{Key: "site-a", AssetID: "asset-1"}
	snapshots := []routes.RouteSnapshot{sampleRouteSnapshot()}
	topo := BuildSiteTopology(site, snapshots)
	cli := &fakeTopologyClient{attrs: map[string]any{"topology.logical.ipv4.fingerprint": topo.Fingerprint}}
	p := &TopologyPublisher{Client: cli, Site: site}

	if err := p.PublishIfChanged(context.Background(), snapshots); err != nil {
		t.Fatalf("PublishIfChanged: %v", err)
	}

	if cli.getCalls != 1 {
		t.Fatalf("expected 1 attribute read, got %d", cli.getCalls)
	}
	if cli.saveCalls != 0 {
		t.Fatalf("expected 0 publishes, got %d", cli.saveCalls)
	}
}

func TestTopologyPublisher_FirstRunPublishesWhenRemoteFingerprintDiffers(t *testing.T) {
	site := SiteConfig{Key: "site-a", AssetID: "asset-1"}
	snapshots := []routes.RouteSnapshot{sampleRouteSnapshot()}
	cli := &fakeTopologyClient{attrs: map[string]any{"topology.logical.ipv4.fingerprint": "old-fingerprint"}}
	p := &TopologyPublisher{Client: cli, Site: site}

	if err := p.PublishIfChanged(context.Background(), snapshots); err != nil {
		t.Fatalf("PublishIfChanged: %v", err)
	}

	if cli.getCalls != 1 {
		t.Fatalf("expected 1 attribute read, got %d", cli.getCalls)
	}
	if cli.saveCalls != 1 {
		t.Fatalf("expected 1 publish, got %d", cli.saveCalls)
	}
	if cli.lastSavedAsset != "asset-1" {
		t.Fatalf("expected asset-1, got %q", cli.lastSavedAsset)
	}
	if cli.lastSavedAttrs["topology.logical.ipv4.fingerprint"] == "old-fingerprint" {
		t.Fatalf("expected updated fingerprint to be saved")
	}
}

func sampleRouteSnapshot() routes.RouteSnapshot {
	return routes.RouteSnapshot{
		DeviceID:      "router-1",
		AddressFamily: "ipv4",
		Supported:     true,
		Routes: []routes.RouteEntry{
			{DeviceID: "router-1", Destination: "192.168.1.0/24", RouteType: "connected"},
			{DeviceID: "router-1", Destination: "0.0.0.0/0", RouteType: "static", IsDefault: true, NextHop: "192.168.1.1"},
		},
	}
}
