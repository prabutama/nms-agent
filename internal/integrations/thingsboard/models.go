package thingsboard

import "time"

type Config struct {
	API  APIConfig
	Site SiteConfig
}

type APIConfig struct {
	BaseURL string
	APIKey  string
}

type SiteConfig struct {
	Key       string
	AssetID   string
	AssetName string
}

type Relation struct {
	From EntityRef `json:"from"`
	To   EntityRef `json:"to"`
	Type string    `json:"type"`
}

type EntityRef struct {
	EntityType string `json:"entityType"`
	ID         string `json:"id"`
}

type DeviceInfo struct {
	ID   EntityRef `json:"id"`
	Name string    `json:"name"`
}

type TopologySnapshot struct {
	SiteKey     string         `json:"site_key"`
	AssetID     string         `json:"asset_id"`
	GeneratedAt time.Time      `json:"generated_at"`
	DeviceCount int            `json:"device_count"`
	EdgeCount   int            `json:"edge_count"`
	Fingerprint string         `json:"fingerprint"`
	Nodes       []TopologyNode `json:"nodes"`
	Edges       []TopologyEdge `json:"edges"`
}

type TopologyNode struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	DeviceID      string `json:"device_id,omitempty"`
	InterfaceID   string `json:"interface_id,omitempty"`
	InterfaceName string `json:"interface_name,omitempty"`
	Subnet        string `json:"subnet,omitempty"`
}

type TopologyEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Reason   string `json:"reason"`
	Resolved bool   `json:"resolved"`
}

type AlarmRequest struct {
	Type              string         `json:"type"`
	Originator        EntityRef      `json:"originator"`
	Severity          string         `json:"severity"`
	Acknowledged      bool           `json:"acknowledged"`
	Cleared           bool           `json:"cleared"`
	Propagate         bool           `json:"propagate"`
	PropagateToOwner  bool           `json:"propagateToOwner"`
	PropagateToTenant bool           `json:"propagateToTenant"`
	Details           map[string]any `json:"details,omitempty"`
	StartTs           int64          `json:"startTs,omitempty"`
	EndTs             int64          `json:"endTs,omitempty"`
	Name              string         `json:"name,omitempty"`
}

type Alarm struct {
	ID         EntityRef `json:"id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Status     string    `json:"status"`
	Cleared    bool      `json:"cleared"`
	Originator EntityRef `json:"originator"`
}
