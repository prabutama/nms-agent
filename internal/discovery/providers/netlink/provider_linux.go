//go:build linux

package netlink

import (
	"context"
	"fmt"
	"net"

	vnetlink "github.com/vishvananda/netlink"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
)

type Provider struct{}

func (p Provider) Candidates(ctx context.Context, loaded config.Loaded) ([]discovery.Candidate, error) {
	_ = ctx
	link, err := vnetlink.LinkByName(loaded.Root.Discovery.Interface)
	if err != nil {
		return nil, err
	}
	_, cidr, err := net.ParseCIDR(loaded.Root.Discovery.Subnet)
	if err != nil {
		return nil, err
	}
	addrList, err := vnetlink.AddrList(link, vnetlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}
	localIPs := map[string]struct{}{}
	for _, addr := range addrList {
		if addr.IP != nil {
			localIPs[addr.IP.String()] = struct{}{}
		}
	}
	neighs, err := vnetlink.NeighList(link.Attrs().Index, vnetlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]discovery.Candidate, 0, len(neighs))
	for _, n := range neighs {
		if n.IP == nil || !cidr.Contains(n.IP) {
			continue
		}
		if _, ok := localIPs[n.IP.String()]; ok {
			continue
		}
		if _, ok := seen[n.IP.String()]; ok {
			continue
		}
		seen[n.IP.String()] = struct{}{}
		mac := ""
		if n.HardwareAddr != nil {
			mac = n.HardwareAddr.String()
		}
		out = append(out, discovery.Candidate{
			Address:   n.IP.String(),
			MAC:       mac,
			Interface: loaded.Root.Discovery.Interface,
			Source:    fmt.Sprintf("netlink:%s", loaded.Root.Discovery.Interface),
		})
	}
	return out, nil
}
