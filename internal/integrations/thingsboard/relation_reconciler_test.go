package thingsboard

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRelationClient struct {
	relations        []Relation
	deviceByName     map[string]*DeviceInfo
	getRelationsHits int
	getDeviceHits    map[string]int
	createCalls      []string
	relationsErr     error
	deviceErr        error
	createErr        error
}

func newFakeRelationClient() *fakeRelationClient {
	return &fakeRelationClient{
		deviceByName:  map[string]*DeviceInfo{},
		getDeviceHits: map[string]int{},
	}
}

func (f *fakeRelationClient) GetRelations(_ context.Context, assetID string) ([]Relation, error) {
	f.getRelationsHits++
	if f.relationsErr != nil {
		return nil, f.relationsErr
	}
	out := make([]Relation, len(f.relations))
	copy(out, f.relations)
	return out, nil
}

func (f *fakeRelationClient) GetDeviceByName(_ context.Context, name string) (*DeviceInfo, error) {
	f.getDeviceHits[name]++
	if f.deviceErr != nil {
		return nil, f.deviceErr
	}
	device, ok := f.deviceByName[name]
	if !ok {
		return nil, errors.New("device not found")
	}
	return device, nil
}

func (f *fakeRelationClient) CreateRelation(_ context.Context, assetID, deviceID, relationType string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createCalls = append(f.createCalls, deviceID)
	f.relations = append(f.relations, Relation{
		From: EntityRef{EntityType: "ASSET", ID: assetID},
		To:   EntityRef{EntityType: "DEVICE", ID: deviceID},
		Type: relationType,
	})
	return nil
}

func TestRelationReconciler_CachesDeviceLookupAndCreatesOnce(t *testing.T) {
	cli := newFakeRelationClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	r := &RelationReconciler{
		Client:          cli,
		Site:            SiteConfig{AssetID: "asset-1"},
		cache:           map[string]bool{},
		deviceIDByName:  map[string]string{},
		refreshInterval: 5 * time.Minute,
		now:             func() time.Time { return time.Unix(100, 0) },
	}

	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("first EnsureContainsRelations: %v", err)
	}
	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("second EnsureContainsRelations: %v", err)
	}

	if cli.getRelationsHits != 1 {
		t.Fatalf("expected 1 relations refresh, got %d", cli.getRelationsHits)
	}
	if cli.getDeviceHits["router-1"] != 1 {
		t.Fatalf("expected 1 device lookup, got %d", cli.getDeviceHits["router-1"])
	}
	if len(cli.createCalls) != 1 {
		t.Fatalf("expected 1 relation create, got %d", len(cli.createCalls))
	}
}

func TestRelationReconciler_UsesExistingRelationFromRefresh(t *testing.T) {
	cli := newFakeRelationClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	cli.relations = []Relation{{
		From: EntityRef{EntityType: "ASSET", ID: "asset-1"},
		To:   EntityRef{EntityType: "DEVICE", ID: "dev-1"},
		Type: "Contains",
	}}
	r := &RelationReconciler{
		Client:          cli,
		Site:            SiteConfig{AssetID: "asset-1"},
		cache:           map[string]bool{},
		deviceIDByName:  map[string]string{},
		refreshInterval: 5 * time.Minute,
		now:             func() time.Time { return time.Unix(100, 0) },
	}

	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("EnsureContainsRelations: %v", err)
	}

	if len(cli.createCalls) != 0 {
		t.Fatalf("expected no relation create, got %d", len(cli.createCalls))
	}
}

func TestRelationReconciler_RefreshesAfterInterval(t *testing.T) {
	cli := newFakeRelationClient()
	cli.deviceByName["router-1"] = &DeviceInfo{ID: EntityRef{EntityType: "DEVICE", ID: "dev-1"}, Name: "router-1"}
	times := []time.Time{time.Unix(100, 0), time.Unix(101, 0), time.Unix(500, 0)}
	idx := 0
	r := &RelationReconciler{
		Client:          cli,
		Site:            SiteConfig{AssetID: "asset-1"},
		cache:           map[string]bool{},
		deviceIDByName:  map[string]string{},
		refreshInterval: 2 * time.Minute,
		now: func() time.Time {
			if idx >= len(times) {
				return times[len(times)-1]
			}
			v := times[idx]
			idx++
			return v
		},
	}

	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("first EnsureContainsRelations: %v", err)
	}
	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("second EnsureContainsRelations: %v", err)
	}
	if err := r.EnsureContainsRelations(context.Background(), []string{"router-1"}); err != nil {
		t.Fatalf("third EnsureContainsRelations: %v", err)
	}

	if cli.getRelationsHits != 2 {
		t.Fatalf("expected 2 relation refreshes, got %d", cli.getRelationsHits)
	}
}
