package thingsboard

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"nms-agent/internal/routes"
)

type TopologyPublisher struct {
	Client topologyClient
	Site   SiteConfig
	mu     sync.Mutex
	lastFP string
	loaded bool
}

type topologyClient interface {
	SaveAssetServerAttributes(ctx context.Context, assetID string, attrs map[string]any) error
	GetAssetServerAttributes(ctx context.Context, assetID string, keys []string) (map[string]any, error)
}

func NewTopologyPublisher(client *Client, site SiteConfig) *TopologyPublisher {
	return &TopologyPublisher{Client: client, Site: site}
}

func (p *TopologyPublisher) PublishIfChanged(ctx context.Context, snapshots []routes.RouteSnapshot) error {
	if p == nil || p.Client == nil || p.Site.AssetID == "" {
		return nil
	}
	topo := BuildSiteTopology(p.Site, snapshots)
	if err := p.ensureLoaded(ctx); err != nil {
		return err
	}
	p.mu.Lock()
	if topo.Fingerprint == p.lastFP {
		p.mu.Unlock()
		return nil
	}
	p.lastFP = topo.Fingerprint
	p.mu.Unlock()
	b, err := json.Marshal(topo)
	if err != nil {
		return err
	}
	attrs := map[string]any{
		"topology.logical.ipv4.snapshot":     string(b),
		"topology.logical.ipv4.fingerprint":  topo.Fingerprint,
		"topology.logical.ipv4.device_count": topo.DeviceCount,
		"topology.logical.ipv4.edge_count":   topo.EdgeCount,
		"topology.logical.ipv4.updated_at":   topo.GeneratedAt.Format(time.RFC3339),
	}
	return p.Client.SaveAssetServerAttributes(ctx, p.Site.AssetID, attrs)
}

func (p *TopologyPublisher) ensureLoaded(ctx context.Context) error {
	p.mu.Lock()
	if p.loaded {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	attrs, err := p.Client.GetAssetServerAttributes(ctx, p.Site.AssetID, []string{"topology.logical.ipv4.fingerprint"})
	if err != nil {
		p.mu.Lock()
		p.loaded = true
		p.mu.Unlock()
		return nil
	}

	var fp string
	if raw, ok := attrs["topology.logical.ipv4.fingerprint"]; ok {
		if s, ok := raw.(string); ok {
			fp = s
		}
	}
	p.mu.Lock()
	if p.lastFP == "" {
		p.lastFP = fp
	}
	p.loaded = true
	p.mu.Unlock()
	return nil
}
