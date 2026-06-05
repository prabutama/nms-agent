package providers

import (
	"strings"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
	activeprovider "nms-agent/internal/discovery/providers/active"
	discoverynetlink "nms-agent/internal/discovery/providers/netlink"
)

func New(loaded config.Loaded) discovery.CandidateProvider {
	switch strings.TrimSpace(loaded.Root.Discovery.Provider) {
	case "active":
		return activeprovider.Provider{}
	case "", "netlink":
		fallthrough
	default:
		return discoverynetlink.Provider{}
	}
}
