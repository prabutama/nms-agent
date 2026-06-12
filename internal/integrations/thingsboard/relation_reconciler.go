package thingsboard

import (
	"context"
	"sync"
)

type RelationReconciler struct {
	Client *Client
	Site   SiteConfig
	mu     sync.Mutex
	cache  map[string]bool
}

func NewRelationReconciler(client *Client, site SiteConfig) *RelationReconciler {
	return &RelationReconciler{Client: client, Site: site, cache: map[string]bool{}}
}

func (r *RelationReconciler) EnsureContainsRelations(ctx context.Context, deviceNames []string) error {
	if r == nil || r.Client == nil || r.Site.AssetID == "" {
		return nil
	}
	relations, err := r.Client.GetRelations(ctx, r.Site.AssetID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	for _, rel := range relations {
		if rel.Type == "Contains" && rel.To.EntityType == "DEVICE" {
			r.cache[rel.To.ID] = true
		}
	}
	r.mu.Unlock()
	for _, name := range deviceNames {
		device, err := r.Client.GetDeviceByName(ctx, name)
		if err != nil {
			continue
		}
		r.mu.Lock()
		exists := r.cache[device.ID.ID]
		r.mu.Unlock()
		if exists {
			continue
		}
		if err := r.Client.CreateRelation(ctx, r.Site.AssetID, device.ID.ID, "Contains"); err != nil {
			continue
		}
		r.mu.Lock()
		r.cache[device.ID.ID] = true
		r.mu.Unlock()
	}
	return nil
}
