package thingsboard

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"nms-agent/internal/routes"
)

type TopologyPublisher struct {
	Client *Client
	Site   SiteConfig
	mu     sync.Mutex
	lastFP string
}

func NewTopologyPublisher(client *Client, site SiteConfig) *TopologyPublisher {
	return &TopologyPublisher{Client: client, Site: site}
}

func (p *TopologyPublisher) PublishIfChanged(ctx context.Context, snapshots []routes.RouteSnapshot) error {
	if p == nil || p.Client == nil || p.Site.AssetID == "" {
		return nil
	}
	topo := BuildSiteTopology(p.Site, snapshots)
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
