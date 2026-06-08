package routes

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

func resolveRoutes(deviceID, source string, rows []rawRouteRow, ifNames map[string]string, collectedAt time.Time) []RouteEntry {
	out := make([]RouteEntry, 0, len(rows))
	for _, row := range rows {
		prefixLen := maskToPrefix(row.mask)
		destination := routeDestination(row.destIP, prefixLen)
		entry := RouteEntry{
			DeviceID:      deviceID,
			AddressFamily: "ipv4",
			Destination:   destination,
			DestinationIP: row.destIP,
			PrefixLength:  prefixLen,
			Mask:          row.mask,
			NextHop:       row.nextHop,
			InterfaceID:   normalizeZeroString(row.ifIndex),
			Metric:        row.metric1,
			Protocol:      mapRouteProtocol(source, row.proto),
			RouteType:     mapRouteType(source, row.routeType),
			IsDefault:     row.destIP == "0.0.0.0" && prefixLen == 0,
			Source:        source,
			CollectedAt:   collectedAt,
		}
		if entry.InterfaceID != "" {
			entry.InterfaceName = ifNames[entry.InterfaceID]
		}
		out = append(out, entry)
	}
	resolveMissingInterfaces(out)
	sortRouteEntries(out)
	return out
}

func resolveMissingInterfaces(routes []RouteEntry) {
	connected := make([]RouteEntry, 0, len(routes))
	for _, route := range routes {
		if route.RouteType == "connected" && route.InterfaceID != "" {
			connected = append(connected, route)
		}
	}
	sort.Slice(connected, func(i, j int) bool {
		return connected[i].PrefixLength > connected[j].PrefixLength
	})
	for i := range routes {
		if routes[i].InterfaceID != "" || routes[i].NextHop == "" || routes[i].NextHop == "0.0.0.0" {
			continue
		}
		nextHop, err := netip.ParseAddr(routes[i].NextHop)
		if err != nil {
			continue
		}
		for _, candidate := range connected {
			prefix, err := netip.ParsePrefix(candidate.Destination)
			if err != nil {
				continue
			}
			if prefix.Contains(nextHop) {
				routes[i].InterfaceID = candidate.InterfaceID
				routes[i].InterfaceName = candidate.InterfaceName
				routes[i].InterfaceResolvedBy = "next_hop_connected_route"
				break
			}
		}
	}
}

func routeDestination(destIP string, prefixLen int) string {
	if destIP == "" {
		return ""
	}
	if prefixLen < 0 {
		return destIP
	}
	return fmt.Sprintf("%s/%d", destIP, prefixLen)
}

func maskToPrefix(mask string) int {
	prefix, err := netip.ParsePrefix("0.0.0.0/0")
	_ = prefix
	addr, err := netip.ParseAddr(mask)
	if err != nil || !addr.Is4() {
		return -1
	}
	b := addr.As4()
	p := 0
	for _, oct := range b {
		for bit := 7; bit >= 0; bit-- {
			if oct&(1<<bit) != 0 {
				p++
			} else {
				return p
			}
		}
	}
	return p
}

func normalizeZeroString(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "0" {
		return ""
	}
	return v
}

func mapRouteType(source string, raw int) string {
	_ = source
	switch raw {
	case 1:
		return "other"
	case 2:
		return "invalid"
	case 3:
		return "connected"
	case 4:
		return "remote"
	default:
		return fmt.Sprintf("unknown(%d)", raw)
	}
}

func mapRouteProtocol(source string, raw int) string {
	_ = source
	switch raw {
	case 1:
		return "other"
	case 2:
		return "local"
	case 3:
		return "netmgmt"
	case 4:
		return "icmp"
	case 5:
		return "egp"
	case 6:
		return "ggp"
	case 7:
		return "hello"
	case 8:
		return "rip"
	case 9:
		return "isis"
	case 10:
		return "esis"
	case 11:
		return "cisco_igrp"
	case 12:
		return "bbn_spf_igp"
	case 13:
		return "ospf"
	case 14:
		return "bgp"
	default:
		return fmt.Sprintf("unknown(%d)", raw)
	}
}

func sortRouteEntries(routes []RouteEntry) {
	sort.Slice(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]
		if a.Destination != b.Destination {
			return a.Destination < b.Destination
		}
		if a.PrefixLength != b.PrefixLength {
			return a.PrefixLength < b.PrefixLength
		}
		if a.NextHop != b.NextHop {
			return a.NextHop < b.NextHop
		}
		if a.InterfaceID != b.InterfaceID {
			return a.InterfaceID < b.InterfaceID
		}
		if a.Metric != b.Metric {
			return a.Metric < b.Metric
		}
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		return a.RouteType < b.RouteType
	})
}
