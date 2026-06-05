package providers

import (
	"testing"

	"nms-agent/internal/config"
	activeprovider "nms-agent/internal/discovery/providers/active"
	discoverynetlink "nms-agent/internal/discovery/providers/netlink"
)

func TestNew_SelectsProviderByConfig(t *testing.T) {
	activeLoaded := config.Loaded{Root: config.Root{Discovery: config.Discovery{Provider: "active"}}}
	if _, ok := New(activeLoaded).(activeprovider.Provider); !ok {
		t.Fatalf("expected active provider")
	}
	defaultLoaded := config.Loaded{Root: config.Root{Discovery: config.Discovery{Provider: "netlink"}}}
	if _, ok := New(defaultLoaded).(discoverynetlink.Provider); !ok {
		t.Fatalf("expected netlink provider")
	}
}
