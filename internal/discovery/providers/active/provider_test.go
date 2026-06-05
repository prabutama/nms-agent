package active

import (
	"net"
	"testing"
)

func TestHostsInSubnet_SkipsNetworkBroadcastAndLocal(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.10.0/30")
	if err != nil {
		t.Fatal(err)
	}
	hosts := hostsInSubnet(subnet, map[string]struct{}{"192.168.10.1": {}})
	if len(hosts) != 1 || hosts[0] != "192.168.10.2" {
		t.Fatalf("got %v want [192.168.10.2]", hosts)
	}
}
