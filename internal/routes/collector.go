package routes

import (
	"context"
	"time"

	"nms-agent/internal/collectors"
	"nms-agent/internal/models"
)

type Collector struct {
	Targets  []collectors.Target
	Provider Provider
	Cache    *ChangeCache
}

func (c Collector) Collect(ctx context.Context) ([]models.RawSample, error) {
	if len(c.Targets) == 0 {
		return nil, nil
	}
	provider := c.Provider
	if provider == nil {
		provider = SNMPProvider{Timeout: 2 * time.Second, Retries: 1}
	}
	cache := c.Cache
	if cache == nil {
		cache = NewChangeCache()
	}
	out := make([]models.RawSample, 0, len(c.Targets)*8)
	for _, target := range c.Targets {
		snapshot, err := provider.Collect(ctx, target.DeviceID, target.Address)
		if err != nil {
			continue
		}
		sortRouteEntries(snapshot.Routes)
		snapshot.Fingerprint = Fingerprint(snapshot.Routes)
		snapshot.Changed = cache.Update(target.DeviceID, snapshot.AddressFamily, snapshot.Fingerprint)
		snapshot = summarizeSnapshot(snapshot)
		raw, err := NormalizeSnapshot(snapshot)
		if err != nil {
			continue
		}
		out = append(out, raw...)
	}
	return out, nil
}
