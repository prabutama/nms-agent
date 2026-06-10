package thingsboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"sort"
	"time"

	"nms-agent/internal/routes"
)

func BuildSiteTopology(site SiteConfig, snapshots []routes.RouteSnapshot) TopologySnapshot {
	ts := TopologySnapshot{SiteKey: site.Key, AssetID: site.AssetID, GeneratedAt: time.Now().UTC()}
	nodeSeen := map[string]bool{}
	edges := make([]TopologyEdge, 0)
	connectedPrefixes := make([]routes.RouteEntry, 0)
	for _, snap := range snapshots {
		if !snap.Supported {
			continue
		}
		deviceNodeID := "device:" + snap.DeviceID
		if !nodeSeen[deviceNodeID] {
			nodeSeen[deviceNodeID] = true
			ts.Nodes = append(ts.Nodes, TopologyNode{ID: deviceNodeID, Kind: "device", Name: snap.DeviceID, DeviceID: snap.DeviceID})
		}
		for _, route := range snap.Routes {
			if route.RouteType == "connected" {
				subnetNodeID := "subnet:" + route.Destination
				if !nodeSeen[subnetNodeID] {
					nodeSeen[subnetNodeID] = true
					ts.Nodes = append(ts.Nodes, TopologyNode{ID: subnetNodeID, Kind: "subnet", Name: route.Destination, Subnet: route.Destination})
				}
				edges = append(edges, TopologyEdge{From: deviceNodeID, To: subnetNodeID, Reason: "connected_subnet", Resolved: true})
				connectedPrefixes = append(connectedPrefixes, route)
			}
		}
	}
	for _, snap := range snapshots {
		deviceNodeID := "device:" + snap.DeviceID
		for _, route := range snap.Routes {
			if !route.IsDefault || route.NextHop == "" || route.NextHop == "0.0.0.0" {
				continue
			}
			nextHop, err := netip.ParseAddr(route.NextHop)
			if err != nil {
				continue
			}
			resolved := false
			for _, cand := range connectedPrefixes {
				if route.DeviceID == cand.DeviceID {
					continue
				}
				prefix, err := netip.ParsePrefix(cand.Destination)
				if err != nil {
					continue
				}
				if prefix.Contains(nextHop) {
					edges = append(edges, TopologyEdge{From: deviceNodeID, To: "device:" + cand.DeviceID, Reason: "next_hop_match", Resolved: true})
					resolved = true
					break
				}
			}
			if !resolved {
				externalNodeID := "external:" + route.NextHop
				if !nodeSeen[externalNodeID] {
					nodeSeen[externalNodeID] = true
					ts.Nodes = append(ts.Nodes, TopologyNode{ID: externalNodeID, Kind: "external_gateway", Name: route.NextHop})
				}
				edges = append(edges, TopologyEdge{From: deviceNodeID, To: externalNodeID, Reason: "default_route", Resolved: false})
			}
		}
	}
	sort.Slice(ts.Nodes, func(i, j int) bool { return ts.Nodes[i].ID < ts.Nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Reason < edges[j].Reason
	})
	ts.Edges = edges
	ts.DeviceCount = 0
	for _, n := range ts.Nodes {
		if n.Kind == "device" {
			ts.DeviceCount++
		}
	}
	ts.EdgeCount = len(ts.Edges)
	b, _ := json.Marshal(struct {
		Nodes []TopologyNode `json:"nodes"`
		Edges []TopologyEdge `json:"edges"`
	}{ts.Nodes, ts.Edges})
	sum := sha256.Sum256(b)
	ts.Fingerprint = hex.EncodeToString(sum[:])
	return ts
}
