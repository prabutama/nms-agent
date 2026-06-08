package routes

import "time"

const (
	SourceSNMPInetCidrRouteTable = "snmp_inet_cidr_route_table"
	SourceSNMPIPCidrRouteTable   = "snmp_ip_cidr_route_table"
	SourceSNMPIPRouteTableLegacy = "snmp_ip_route_table_legacy"
)

type RouteEntry struct {
	DeviceID            string    `json:"device_id"`
	AddressFamily       string    `json:"address_family"`
	Destination         string    `json:"destination"`
	DestinationIP       string    `json:"destination_ip"`
	PrefixLength        int       `json:"prefix_length"`
	Mask                string    `json:"mask"`
	NextHop             string    `json:"next_hop"`
	InterfaceID         string    `json:"interface_id"`
	InterfaceName       string    `json:"interface_name"`
	Metric              int       `json:"metric"`
	Protocol            string    `json:"protocol"`
	RouteType           string    `json:"route_type"`
	IsDefault           bool      `json:"is_default"`
	Source              string    `json:"source"`
	CollectedAt         time.Time `json:"collected_at"`
	InterfaceResolvedBy string    `json:"interface_resolved_by,omitempty"`
}

type RouteSnapshot struct {
	DeviceID            string       `json:"device_id"`
	AddressFamily       string       `json:"address_family"`
	Source              string       `json:"source"`
	Supported           bool         `json:"supported"`
	CollectedAt         time.Time    `json:"collected_at"`
	Routes              []RouteEntry `json:"routes"`
	RouteCount          int          `json:"route_count"`
	DefaultRouteCount   int          `json:"default_route_count"`
	ConnectedRouteCount int          `json:"connected_route_count"`
	RemoteRouteCount    int          `json:"remote_route_count"`
	Fingerprint         string       `json:"fingerprint"`
	Changed             bool         `json:"changed"`
}

type rawRouteRow struct {
	key       string
	destIP    string
	mask      string
	nextHop   string
	ifIndex   string
	metric1   int
	routeType int
	proto     int
}
