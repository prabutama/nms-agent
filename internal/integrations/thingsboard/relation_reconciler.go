package thingsboard

import (
	"context"
	"sync"
	"time"
)

type relationClient interface {
	GetRelations(ctx context.Context, assetID string) ([]Relation, error)
	GetDeviceByName(ctx context.Context, name string) (*DeviceInfo, error)
	CreateRelation(ctx context.Context, assetID, deviceID, relationType string) error
}

type RelationReconciler struct {
	Client          relationClient
	Site            SiteConfig
	mu              sync.Mutex
	cache           map[string]bool
	deviceIDByName  map[string]string
	lastRefresh     time.Time
	refreshInterval time.Duration
	now             func() time.Time
}

func NewRelationReconciler(client *Client, site SiteConfig) *RelationReconciler {
	return &RelationReconciler{
		Client:          client,
		Site:            site,
		cache:           map[string]bool{},
		deviceIDByName:  map[string]string{},
		refreshInterval: 5 * time.Minute,
		now:             time.Now,
	}
}

func (r *RelationReconciler) EnsureContainsRelations(ctx context.Context, deviceNames []string) error {
	if r == nil || r.Client == nil || r.Site.AssetID == "" {
		return nil
	}
	if err := r.refreshRelationsIfNeeded(ctx); err != nil {
		return err
	}
	for _, name := range deviceNames {
		deviceID, err := r.lookupDeviceID(ctx, name)
		if err != nil {
			continue
		}
		r.mu.Lock()
		exists := r.cache[deviceID]
		r.mu.Unlock()
		if exists {
			continue
		}
		if err := r.Client.CreateRelation(ctx, r.Site.AssetID, deviceID, "Contains"); err != nil {
			continue
		}
		r.mu.Lock()
		r.cache[deviceID] = true
		r.mu.Unlock()
	}
	return nil
}

func (r *RelationReconciler) refreshRelationsIfNeeded(ctx context.Context) error {
	nowFn := r.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()
	r.mu.Lock()
	needRefresh := r.lastRefresh.IsZero() || (r.refreshInterval > 0 && now.Sub(r.lastRefresh) >= r.refreshInterval)
	r.mu.Unlock()
	if !needRefresh {
		return nil
	}
	relations, err := r.Client.GetRelations(ctx, r.Site.AssetID)
	if err != nil {
		return err
	}
	relCache := make(map[string]bool, len(relations))
	for _, rel := range relations {
		if rel.Type == "Contains" && rel.To.EntityType == "DEVICE" {
			relCache[rel.To.ID] = true
		}
	}
	r.mu.Lock()
	for id, exists := range relCache {
		r.cache[id] = exists
	}
	r.lastRefresh = now
	r.mu.Unlock()
	return nil
}

func (r *RelationReconciler) lookupDeviceID(ctx context.Context, name string) (string, error) {
	r.mu.Lock()
	if id := r.deviceIDByName[name]; id != "" {
		r.mu.Unlock()
		return id, nil
	}
	r.mu.Unlock()
	device, err := r.Client.GetDeviceByName(ctx, name)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.deviceIDByName[name] = device.ID.ID
	r.mu.Unlock()
	return device.ID.ID, nil
}
