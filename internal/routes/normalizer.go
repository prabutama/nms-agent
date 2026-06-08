package routes

import (
	"encoding/json"
	"fmt"
	"time"

	"nms-agent/internal/models"
)

const maxSnapshotRoutes = 128

func NormalizeSnapshot(snapshot RouteSnapshot) ([]models.RawSample, error) {
	collectedAt := snapshot.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}
	out := make([]models.RawSample, 0, 14)
	appendNumber := func(metric string, value float64) {
		out = append(out, rawNumber(snapshot.DeviceID, metric, collectedAt, value))
	}
	appendString := func(metric, value string) {
		out = append(out, rawString(snapshot.DeviceID, metric, collectedAt, value))
	}
	if snapshot.Supported {
		appendNumber("route.ipv4.supported", 1)
	} else {
		appendNumber("route.ipv4.supported", 0)
	}
	appendNumber("route.ipv4.route_count", float64(snapshot.RouteCount))
	appendNumber("route.ipv4.default_route_count", float64(snapshot.DefaultRouteCount))
	appendNumber("route.ipv4.connected_route_count", float64(snapshot.ConnectedRouteCount))
	appendNumber("route.ipv4.remote_route_count", float64(snapshot.RemoteRouteCount))
	if snapshot.Changed {
		appendNumber("route.ipv4.changed", 1)
	} else {
		appendNumber("route.ipv4.changed", 0)
	}
	appendString("route.ipv4.source", snapshot.Source)
	if def, ok := firstDefaultRoute(snapshot.Routes); ok {
		appendString("route.ipv4.default.destination", def.Destination)
		appendString("route.ipv4.default.next_hop", def.NextHop)
		appendString("route.ipv4.default.interface_id", def.InterfaceID)
		appendString("route.ipv4.default.interface_name", def.InterfaceName)
		appendString("route.ipv4.default.protocol", def.Protocol)
		appendString("route.ipv4.default.route_type", def.RouteType)
	} else {
		appendString("route.ipv4.default.destination", "")
		appendString("route.ipv4.default.next_hop", "")
		appendString("route.ipv4.default.interface_id", "")
		appendString("route.ipv4.default.interface_name", "")
		appendString("route.ipv4.default.protocol", "")
		appendString("route.ipv4.default.route_type", "")
	}
	snapJSON, err := snapshotJSON(snapshot)
	if err != nil {
		return nil, err
	}
	appendString("route.ipv4.snapshot", snapJSON)
	return out, nil
}

func snapshotJSON(snapshot RouteSnapshot) (string, error) {
	copySnapshot := snapshot
	if len(copySnapshot.Routes) > maxSnapshotRoutes {
		copySnapshot.Routes = append([]RouteEntry(nil), copySnapshot.Routes[:maxSnapshotRoutes]...)
	}
	b, err := json.Marshal(copySnapshot)
	if err != nil {
		return "", fmt.Errorf("marshal route snapshot: %w", err)
	}
	return string(b), nil
}

func firstDefaultRoute(routes []RouteEntry) (RouteEntry, bool) {
	for _, route := range routes {
		if route.IsDefault {
			return route, true
		}
	}
	return RouteEntry{}, false
}

func rawNumber(deviceID, metric string, ts time.Time, value float64) models.RawSample {
	return models.RawSample{DeviceID: deviceID, Source: "route_inventory", TS: ts, Fields: map[string]any{
		"metric":       metric,
		"value_type":   "number",
		"value_number": value,
	}}
}

func rawString(deviceID, metric string, ts time.Time, value string) models.RawSample {
	return models.RawSample{DeviceID: deviceID, Source: "route_inventory", TS: ts, Fields: map[string]any{
		"metric":       metric,
		"value_type":   "string",
		"value_string": value,
	}}
}
